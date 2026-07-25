package audio

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

type Capturer struct {
	analyzer *Analyzer
	stopChan chan struct{}
}

func NewCapturer() *Capturer {
	return &Capturer{
		analyzer: NewAnalyzer(512, 44100.0),
		stopChan: make(chan struct{}),
	}
}

func (c *Capturer) GetFeatures() AudioFeatures {
	if c.analyzer != nil {
		return c.analyzer.GetFeatures()
	}
	return AudioFeatures{}
}

func (c *Capturer) Start() error {
	errChan := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
			errChan <- fmt.Errorf("CoInitializeEx failed: %w", err)
			return
		}
		defer ole.CoUninitialize()

		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			errChan <- fmt.Errorf("CoCreateInstance failed: %w", err)
			return
		}
		defer mmde.Release()

		var mmd *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
			errChan <- fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
			return
		}
		defer mmd.Release()

		var ac *wca.IAudioClient
		if err := mmd.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &ac); err != nil {
			errChan <- fmt.Errorf("Activate IAudioClient failed: %w", err)
			return
		}
		defer ac.Release()

		var wfx *wca.WAVEFORMATEX
		if err := ac.GetMixFormat(&wfx); err != nil {
			errChan <- fmt.Errorf("GetMixFormat failed: %w", err)
			return
		}

		sampleRate := float64(wfx.NSamplesPerSec)
		c.analyzer = NewAnalyzer(512, sampleRate)

		var defaultPeriod, minimumPeriod wca.REFERENCE_TIME
		if err := ac.GetDevicePeriod(&defaultPeriod, &minimumPeriod); err != nil {
			defaultPeriod = 100000
		}

		if err := ac.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, wca.AUDCLNT_STREAMFLAGS_LOOPBACK, defaultPeriod, 0, wfx, nil); err != nil {
			errChan <- fmt.Errorf("Initialize WASAPI loopback failed: %w", err)
			return
		}

		var acc *wca.IAudioCaptureClient
		if err := ac.GetService(wca.IID_IAudioCaptureClient, &acc); err != nil {
			errChan <- fmt.Errorf("GetService IAudioCaptureClient failed: %w", err)
			return
		}
		defer acc.Release()

		if err := ac.Start(); err != nil {
			errChan <- fmt.Errorf("Start audio client failed: %w", err)
			return
		}
		defer ac.Stop()

		errChan <- nil // Signal successful audio capture start

		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChan:
				return
			case <-ticker.C:
				var packetLength uint32
				if err := acc.GetNextPacketSize(&packetLength); err != nil || packetLength == 0 {
					continue
				}

				for packetLength > 0 {
					var data *byte
					var framesAvailable uint32
					var flags uint32
					var devicePosition uint64
					var qpcPosition uint64

					if err := acc.GetBuffer(&data, &framesAvailable, &flags, &devicePosition, &qpcPosition); err != nil {
						break
					}

					if framesAvailable > 0 && flags == 0 && data != nil {
						channels := int(wfx.NChannels)
						bitsPerSample := int(wfx.WBitsPerSample)
						totalSamples := int(framesAvailable) * channels

						monoSamples := make([]float32, int(framesAvailable))

						if bitsPerSample == 32 {
							rawSamples := (*[1 << 26]float32)(unsafe.Pointer(data))[:totalSamples:totalSamples]
							for i := 0; i < int(framesAvailable); i++ {
								var sum float32
								for ch := 0; ch < channels; ch++ {
									sum += rawSamples[i*channels+ch]
								}
								monoSamples[i] = sum / float32(channels)
							}
						} else if bitsPerSample == 16 {
							rawSamples := (*[1 << 26]int16)(unsafe.Pointer(data))[:totalSamples:totalSamples]
							for i := 0; i < int(framesAvailable); i++ {
								var sum float32
								for ch := 0; ch < channels; ch++ {
									sum += float32(rawSamples[i*channels+ch]) / 32768.0
								}
								monoSamples[i] = sum / float32(channels)
							}
						}

						c.analyzer.ProcessSamples(monoSamples)
					}

					acc.ReleaseBuffer(framesAvailable)
					if err := acc.GetNextPacketSize(&packetLength); err != nil {
						break
					}
				}
			}
		}
	}()

	return <-errChan
}

func (c *Capturer) Stop() {
	close(c.stopChan)
}
