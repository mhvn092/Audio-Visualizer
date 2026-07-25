package audio

import (
	"math"
	"math/cmplx"
	"sync/atomic"
)

type AudioFeatures struct {
	Bass   float32
	Mid    float32
	Treble float32
	Onset  float32
}

type AtomicFeatures struct {
	bassBits   uint32
	midBits    uint32
	trebleBits uint32
	onsetBits  uint32
}

func (af *AtomicFeatures) Store(f AudioFeatures) {
	atomic.StoreUint32(&af.bassBits, math.Float32bits(f.Bass))
	atomic.StoreUint32(&af.midBits, math.Float32bits(f.Mid))
	atomic.StoreUint32(&af.trebleBits, math.Float32bits(f.Treble))
	atomic.StoreUint32(&af.onsetBits, math.Float32bits(f.Onset))
}

func (af *AtomicFeatures) Load() AudioFeatures {
	return AudioFeatures{
		Bass:   math.Float32frombits(atomic.LoadUint32(&af.bassBits)),
		Mid:    math.Float32frombits(atomic.LoadUint32(&af.midBits)),
		Treble: math.Float32frombits(atomic.LoadUint32(&af.trebleBits)),
		Onset:  math.Float32frombits(atomic.LoadUint32(&af.onsetBits)),
	}
}

type Analyzer struct {
	fftSize        int
	hopSize        int
	sampleRate     float64
	hannWindow     []float64
	sampleBuffer   []float32
	prevBass       float64
	currentOnset   float64
	maxEnergyPeak  float64 // Auto-gain normalization
	atomicFeatures AtomicFeatures
}

func NewAnalyzer(fftSize int, sampleRate float64) *Analyzer {
	window := make([]float64, fftSize)
	for i := 0; i < fftSize; i++ {
		window[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(fftSize-1)))
	}
	return &Analyzer{
		fftSize:       fftSize,
		hopSize:       fftSize / 2, // 50% overlap for 90Hz FFT updates
		sampleRate:    sampleRate,
		hannWindow:    window,
		sampleBuffer:  make([]float32, 0, 4096),
		maxEnergyPeak: 1.0,
	}
}

func (a *Analyzer) GetFeatures() AudioFeatures {
	return a.atomicFeatures.Load()
}

func (a *Analyzer) ProcessSamples(samples []float32) {
	a.sampleBuffer = append(a.sampleBuffer, samples...)

	for len(a.sampleBuffer) >= a.fftSize {
		window := a.sampleBuffer[:a.fftSize]
		a.analyzeWindow(window)
		a.sampleBuffer = a.sampleBuffer[a.hopSize:]
	}

	if len(a.sampleBuffer) > 8192 {
		a.sampleBuffer = a.sampleBuffer[len(a.sampleBuffer)-a.fftSize:]
	}
}

func (a *Analyzer) analyzeWindow(samples []float32) {
	complexBuf := make([]complex128, a.fftSize)
	for i := 0; i < a.fftSize; i++ {
		windowedSample := float64(samples[i]) * a.hannWindow[i]
		complexBuf[i] = complex(windowedSample, 0)
	}

	fftOut := fftCooleyTukey(complexBuf)

	numBins := a.fftSize / 2
	binWidth := a.sampleRate / float64(a.fftSize)

	var bassMax, bassSum float64
	var midMax, midSum float64
	var trebleMax, trebleSum float64
	var bassCnt, midCnt, trebleCnt int

	for i := 0; i < numBins; i++ {
		mag := cmplx.Abs(fftOut[i])
		freq := float64(i) * binWidth

		if freq >= 20.0 && freq < 180.0 { // Kick drum & sub bass band
			if mag > bassMax {
				bassMax = mag
			}
			bassSum += mag
			bassCnt++
		} else if freq >= 180.0 && freq < 1800.0 { // Vocals, snare, mid synth band
			if mag > midMax {
				midMax = mag
			}
			midSum += mag
			midCnt++
		} else if freq >= 1800.0 && freq < 16000.0 { // Hi-hats, cymbals, treble band
			if mag > trebleMax {
				trebleMax = mag
			}
			trebleSum += mag
			trebleCnt++
		}
	}

	// Calculate weighted band energies combining Peak + RMS
	var bass, mid, treble float64
	if bassCnt > 0 {
		bass = (bassMax * 0.7) + ((bassSum / float64(bassCnt)) * 0.3)
	}
	if midCnt > 0 {
		mid = (midMax * 0.75) + ((midSum / float64(midCnt)) * 0.25)
	}
	if trebleCnt > 0 {
		treble = (trebleMax * 0.8) + ((trebleSum / float64(trebleCnt)) * 0.2)
	}

	// Auto-Gain Control (AGC): track peak energy with slow decay
	currentMax := math.Max(bass, math.Max(mid, treble))
	if currentMax > a.maxEnergyPeak {
		a.maxEnergyPeak = currentMax
	} else {
		a.maxEnergyPeak = math.Max(0.5, a.maxEnergyPeak*0.995) // Slow AGC decay
	}

	// Normalize energies using AGC peak
	bass = math.Min(1.0, (bass/a.maxEnergyPeak)*2.2)
	mid = math.Min(1.0, (mid/a.maxEnergyPeak)*2.8)
	treble = math.Min(1.0, (treble/a.maxEnergyPeak)*3.2)

	// Spectral Flux Beat / Onset Detection (Positive acceleration of kick energy)
	bassDelta := bass - a.prevBass
	a.prevBass = bass

	if bassDelta > 0.08 && bass > 0.15 {
		a.currentOnset = 1.0 // Instantaneous beat hit!
	} else {
		a.currentOnset *= 0.78 // Snappy beat decay
	}

	features := AudioFeatures{
		Bass:   float32(bass),
		Mid:    float32(mid),
		Treble: float32(treble),
		Onset:  float32(a.currentOnset),
	}

	a.atomicFeatures.Store(features)
}

func fftCooleyTukey(x []complex128) []complex128 {
	n := len(x)
	if n <= 1 {
		out := make([]complex128, n)
		copy(out, x)
		return out
	}

	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	fftEven := fftCooleyTukey(even)
	fftOdd := fftCooleyTukey(odd)

	out := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		angle := -2.0 * math.Pi * float64(k) / float64(n)
		twiddle := cmplx.Rect(1.0, angle) * fftOdd[k]
		out[k] = fftEven[k] + twiddle
		out[k+n/2] = fftEven[k] - twiddle
	}
	return out
}
