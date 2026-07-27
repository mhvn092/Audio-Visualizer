package audio

import (
	"math"
	"sort"
	"sync"
	"time"
)

type BPMEngine struct {
	mu           sync.Mutex
	peakTimes    []time.Time
	lastOnset    time.Time
	estimatedBPM float32
	beatPhase    float32
	beatCount    uint64
	barCount     uint64
	lastBeatTick time.Time
}

func NewBPMEngine() *BPMEngine {
	return &BPMEngine{
		peakTimes:    make([]time.Time, 0, 64),
		estimatedBPM: 120.0, // Default fallback tempo
		lastBeatTick: time.Now(),
	}
}

// ProcessSample receives onset energy (0.0 to 1.0+) and updates BPM estimation & rhythm phase
func (b *BPMEngine) ProcessSample(onset float32, bass float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	// Peak onset threshold detection (minimum 250ms spacing = 240 BPM max)
	if onset > 0.75 && now.Sub(b.lastOnset) > 250*time.Millisecond {
		b.lastOnset = now
		b.peakTimes = append(b.peakTimes, now)

		// Retain only last 8 seconds of peak timestamps
		cutoff := now.Add(-8 * time.Second)
		validIdx := 0
		for i, t := range b.peakTimes {
			if t.After(cutoff) {
				validIdx = i
				break
			}
		}
		if validIdx > 0 {
			b.peakTimes = b.peakTimes[validIdx:]
		}

		// Re-estimate BPM if we have sufficient peak samples
		b.estimateBPM()
	}

	// Calculate beat phase (0.0 to 1.0) based on current estimated BPM
	beatIntervalMs := float64(60000.0 / math.Max(60.0, math.Min(180.0, float64(b.estimatedBPM))))
	elapsedMs := float64(now.Sub(b.lastBeatTick).Milliseconds())

	if elapsedMs >= beatIntervalMs {
		b.lastBeatTick = now
		b.beatCount++
		if b.beatCount%4 == 0 {
			b.barCount++
		}
		elapsedMs = 0
	}

	b.beatPhase = float32(elapsedMs / beatIntervalMs)
}

func (b *BPMEngine) estimateBPM() {
	if len(b.peakTimes) < 4 {
		return
	}

	// Calculate deltas between consecutive peaks
	deltas := make([]float64, 0, len(b.peakTimes)-1)
	for i := 1; i < len(b.peakTimes); i++ {
		deltaMs := float64(b.peakTimes[i].Sub(b.peakTimes[i-1]).Milliseconds())
		if deltaMs >= 250 && deltaMs <= 1200 { // 50 BPM to 240 BPM
			deltas = append(deltas, deltaMs)
		}
	}

	if len(deltas) < 3 {
		return
	}

	// Calculate candidate BPMs
	bpms := make([]float64, len(deltas))
	for i, d := range deltas {
		bpm := 60000.0 / d
		// Normalize 2x harmonics (e.g. 160 BPM vs 80 BPM) into 70-150 BPM range
		for bpm > 155.0 {
			bpm /= 2.0
		}
		for bpm < 75.0 {
			bpm *= 2.0
		}
		bpms[i] = bpm
	}

	sort.Float64s(bpms)
	medianBPM := bpms[len(bpms)/2]

	// Smoothly interpolate estimated BPM
	if b.estimatedBPM == 0 {
		b.estimatedBPM = float32(medianBPM)
	} else {
		b.estimatedBPM = b.estimatedBPM*0.7 + float32(medianBPM)*0.3
	}
}

type BPMState struct {
	BPM       float32
	BeatPhase float32
	BeatCount uint64
	BarCount  uint64
	IsBarBeat bool
}

func (b *BPMEngine) GetState() BPMState {
	b.mu.Lock()
	defer b.mu.Unlock()

	return BPMState{
		BPM:       b.estimatedBPM,
		BeatPhase: b.beatPhase,
		BeatCount: b.beatCount,
		BarCount:  b.barCount,
		IsBarBeat: (b.beatCount % 4) == 0,
	}
}
