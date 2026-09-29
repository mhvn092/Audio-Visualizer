# Phase 0: Foundations (about 1 week)

## Goal

Fix the verified defects, rebuild audio analysis on solid ground, and add the tooling (record/replay, metrics HUD,
test set) that every later phase depends on. No visual redesign yet.

## Why first

Every later phase schedules visuals on musical time. If the beat phase is wrong (D1), features freeze on pause (D2) or
there are data races (D3), nothing built on top can be tight. Record/replay makes audio work testable without playing
music by hand.

## Prerequisites

- Windows 10/11, Go 1.25+, the repo building with `go run .`.
- A folder of 30 test tracks (see task 8) you are allowed to use locally. Do not commit audio files.

## Tasks

### Task 1: Remove data races (D3)

- `audio/capture.go`: construct the analyzer once with the real sample rate **before** starting the goroutine, or
  store it in an `atomic.Pointer[Analyzer]`. `GetFeatures()` must read through the atomic pointer.
- `main.go`: the metadata callback currently assigns `game.currentMeta` from another goroutine. Replace with an
  `atomic.Pointer[audio.TrackMetadata]` (or a channel drained in `Update()`).
- Verify with `go run -race .` for 5 minutes of playback + track changes: zero race reports.

### Task 2: Silence and discontinuity handling (D2)

- In the capture loop, handle `flags`:
  - `AUDCLNT_BUFFERFLAGS_SILENT` (0x2): feed `framesAvailable` zeros to the analyzer.
  - `AUDCLNT_BUFFERFLAGS_DATA_DISCONTINUITY` (0x1): still process the data; reset the analyzer's onset history.
- Loopback delivers **no packets** when nothing is playing. Track the time of the last packet; if more than 50 ms pass
  without one, synthesize zeros so features decay to silence.
- Record the QPC position (`qpcPosition` from `GetBuffer`, in 100 ns units) with each packet; pass it to the
  analyzer so features carry an audio timestamp (needed for latency compensation later).
- Remove the per-packet `make([]float32, ...)`: use a preallocated ring buffer.
- Acceptance: pause music → all features reach 0 within 1 s and `Silent == true`.

### Task 3: New analysis core (D4)

Create `internal/ear/stft.go` and `internal/ear/features.go` (move `audio/` into `internal/ear/` first as a pure move
commit). Implement:

1. **STFT**: 2048-sample Hann window, 256-sample hop (5.3 ms at 48 kHz). Use an iterative radix-2 FFT with precomputed
   twiddles and bit-reversal table, operating on preallocated buffers (zero allocations per hop). `gonum.org/v1/gonum/dsp/fourier`
   is acceptable if it benchmarks under 50 µs per 2048-point transform; otherwise hand-write it.
2. **Mel bands**: 64 triangular mel filters 30 Hz – 16 kHz, magnitudes → `log1p(k * mag)`, then per-band running
   normalization (slow-decaying max, like the existing AGC but per band).
3. **Spectral flux per element** (half-wave-rectified difference between consecutive frames, summed over a band range):
   - Kick: 40–120 Hz; Snare: 150–300 Hz **plus** 2–6 kHz noise; Hat: 7–15 kHz; Vocal/lead: 300 Hz–3 kHz with low flatness.
   - Adaptive threshold per element: `median(last 1 s) + k * MAD(last 1 s)`, with `k ≈ 1.5`; onset strength = excess over
     threshold, clamped to 0..1; minimum spacing 80 ms.
4. **Loudness** (short-term, 400 ms window, simple K-weighting approximation), **spectral centroid**, **flatness**,
   **chroma** (12 bins, summed over octaves).
5. Publish `clock.Features` (see `01-architecture.md`) through an atomic pointer.
6. Keep the old three-band values (`Bass/Mid/Treble/Onset`) as derived fields so the existing shader keeps working.
- Benchmark: `go test -bench .` in `internal/ear`: full analysis of one hop under 150 µs, 0 allocs/op.

### Task 4: A real beat tracker (D1, D5)

Replace `audio/bpm.go` with `internal/ear/tempo.go`:

1. **Onset envelope**: the kick+snare flux sum at the hop rate (~187 Hz).
2. **Tempo**: every 0.5 s, autocorrelate the last 8 s of onset envelope; score lags for 70–180 BPM with a comb of
   harmonics (1×, 2×, 4× lag) and a log-Gaussian prior centred at 124 BPM; pick the best; hysteresis so the tempo only
   changes when a new candidate wins by >10% for 2 s.
3. **Phase lock (PLL)**: keep `beatPhase`. On each detected kick/snare onset near a predicted beat (within ±25% of a
   beat), compute the phase error and correct: `phase += 0.2 * err`, `period += 0.02 * err`. Phase is advanced from
   **audio timestamps**, not `time.Now()` in `Update()`.
4. **Downbeat**: accumulate kick strength per beat-in-bar position (mod 4) with decay; the strongest position is beat 1.
   Also reward positions where the chroma changes (chord changes land on downbeats).
5. **Confidence**: 0..1 from autocorrelation peak sharpness and PLL error variance.
6. Remove the 60 Hz sampling in `main.go:186`; the tracker runs inside the audio goroutine on every hop.
- Acceptance: on the test set (task 8), p90 absolute beat-phase error < 20 ms once locked; lock within 4 s of music start.
  Measure against reference beats (task 8) with the replay harness.

### Task 5: Native media session instead of PowerShell (D10)

- Use `github.com/saltosystems/winrt-go/windows/media/control` (pure Go WinRT bindings) to get
  `GlobalSystemMediaTransportControlsSessionManager`.
- Subscribe to `CurrentSessionChanged`, `MediaPropertiesChanged`, `PlaybackInfoChanged`, `TimelinePropertiesChanged`.
- Expose: title, artist, album, **playback status** (playing/paused), **timeline position + last-updated time**
  (used by Song Memory in Phase 3), and **thumbnail** (read the stream into an image; use it before iTunes).
- Keep `scripts/get_media.ps1` only as a fallback if WinRT activation fails, polled at most every 10 s.
- Acceptance: track change reflected in < 300 ms; pause state visible; zero PowerShell processes during normal use.

### Task 6: Small correctness and hygiene fixes (D8, D11, D12)

- `gfx/library.go:154`: letterbox/cover-fit the art into the target instead of stretching (keep aspect; crop to fill).
- Uniforms: build the `map[string]any` once and update values in place (Ebiten accepts reused maps); same in
  `gfx/postprocess.go`. Goal: 0 allocations in `Draw()` (check with `testing.AllocsPerRun` on a draw helper or pprof).
- `shaders/pulse.kage`: delete it (it is unused), or wire it as an optional feedback pass. Default: delete.

### Task 7: Record/replay harness

- `internal/replay`: a `Source` implementation that reads a WAV file (use `github.com/go-audio/wav` or a small WAV
  reader) and feeds it through the exact same analysis at real-time or as-fast-as-possible speed.
- A recorder that writes the `Features` + `Clock` stream to a compact binary file (`.frec`) with timestamps.
- CLI: `go run ./cmd/analyze track.wav` prints BPM, lock time, and writes `track.frec` + a CSV of beats.
- The app accepts `-replay file.wav` to run visuals from a file instead of loopback (deterministic demos and tests).

### Task 8: Test set and metrics HUD

- Test set: 30 tracks (at least 20 four-on-the-floor electronic, 5 hip-hop/pop, 5 ambient/non-4/4). For each, a
  reference beat file (`.beats`, one timestamp per line). Create references with an offline tracker (e.g. madmom or
  All-In-One, see `references.md`) and spot-check by ear. Store only the reference files in `testdata/beats/`, never audio.
- `go test ./internal/ear -run TestBeatAccuracy -tracks=<folder>`: runs every track through replay, reports p50/p90
  phase error, lock time, BPM error, and fails over the thresholds.
- Metrics HUD (toggle with `F3`): FPS, frame time p50/p99, audio-to-analysis latency, BPM + confidence, beat phase dot
  that flashes on each beat, current section (placeholder until Phase 3).

## Acceptance criteria (all must pass)

- [ ] `go run -race .` for 5 minutes: no race reports.
- [ ] Pause playback: features at rest (all ≤ 0.02) within 1 s.
- [ ] Beat-phase error p90 < 20 ms on the 4/4 part of the test set; lock within 4 s.
- [ ] Analysis: < 150 µs per hop, 0 allocs/op (benchmark output pasted in Results).
- [ ] Track change reflected in < 300 ms, no PowerShell spawned.
- [ ] `Draw()` allocates 0 bytes per frame in steady state.
- [ ] A replayed `.wav` produces the same `.frec` twice (deterministic).

## Pitfalls

- WASAPI loopback timestamps: `qpcPosition` is in 100 ns units of the performance counter; convert once.
- Do not run the tracker in Ebiten's `Update()`; it runs at 60 TPS and will alias beats.
- Hysteresis on tempo is essential; without it, half/double-time flips make the visuals stutter.
- winrt-go event handlers run on other threads: publish through atomics/channels.

## Results

(Fill in when done: benchmark numbers, test-set table, notes.)
