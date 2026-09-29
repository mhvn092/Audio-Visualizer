# 00: Diagnosis of the current app

Audited at commit `6843cf9` (September 2026). Line numbers refer to that commit; if they drift, search for the quoted
code.

## What the app is today

- `main.go`: an Ebiten game loop. `Update()` reads audio features, smooths them, picks an "act" (Monolith / Organism /
  Constellation) from the long-term bass/mid/treble ratio, and switches artwork on strong onsets. `Draw()` passes ~25
  uniforms to one full-screen Kage shader, then a post-process shader.
- `audio/capture.go`: WASAPI loopback capture (via `go-wca`), downmix to mono, feed the analyzer.
- `audio/analysis.go`: 512-point FFT, three bands (bass 20–180 Hz, mid 180–1800 Hz, treble 1.8–16 kHz), auto-gain,
  bass-only onset.
- `audio/bpm.go`: BPM estimate from onset intervals, plus a beat phase and bar counter.
- `audio/metadata.go` + `scripts/get_media.ps1`: track title/artist by running PowerShell every 4 s.
- `gfx/fetcher.go`: album art from iTunes, or procedural fallback art.
- `gfx/artanalysis.go`: k-means palette, brightness, warmth, complexity from the cover.
- `gfx/library.go`: loaded textures, resized to screen size.
- `gfx/postprocess.go` + `shaders/postprocess.kage`: chromatic aberration, "bloom", ACES tonemap, vignette.
- `gfx/watcher.go`: shader hot-reload.
- `shaders/visualizer.kage`: raymarched SDF scene; blends three SDF "acts" and embosses album art.

## Five root causes (why it looks amateur)

1. **Content is invented live.** One full-screen shader builds the whole world from math every frame. Professional
   shows pre-produce content and use real-time only for playback, reaction and control.
2. **Time is just loudness.** No beat grid, bars, phrases, sections or drop forecast. The beat phase is not locked to
   the music (see defect D1).
3. **Light is flat.** 8-bit buffers, no area lights or reflections, no haze, bloom radius of 1.5 px. Stage visuals are
   mostly light: black space, haze, a few very bright sources and chrome that reflects them.
4. **Everything reacts to everything.** Every band drives every parameter all the time, so nothing reads as intentional.
5. **The engine has a ceiling.** Ebiten v2.9.9 (verified in its source): `NewImageOptions` has no pixel-format option,
   so images are 8-bit RGBA only; a shader gets at most 4 source images (`ShaderSrcImageCount = 4`); there is no depth
   buffer, compute, instancing, storage buffers, multiple render targets or 3D textures.

## Verified defects

| ID | Where | What happens | Effect | Fixed in |
|---|---|---|---|---|
| D1 | `audio/bpm.go:59–71` | `lastBeatTick` is a timer started at app launch; detected onsets never move it. | Anything "on the beat" lands at a random offset from the music. | Phase 0, task 4 |
| D2 | `audio/capture.go:124` (`flags == 0`) | Packets flagged silent/discontinuous are skipped; loopback sends nothing while idle. | After a pause, features freeze at the last loud value; visuals keep pumping. | Phase 0, task 2 |
| D3 | `audio/capture.go:73`, `main.go:104` | `c.analyzer` and `game.currentMeta` are replaced from background goroutines while the render loop reads them. | Data races (run `go run -race .` to see them). | Phase 0, task 1 |
| D4 | `audio/analysis.go` | Recursive 512-point FFT allocating per window. At 48 kHz each bin is 94 Hz: about 2 bins for bass. Linear magnitudes; a single bass-only onset. | Kick and bass blur; snare/hats trigger nothing. | Phase 0, task 3 |
| D5 | `main.go:186` | Onset sampled at 60 Hz from a value that decays on every FFT hop, fixed 0.75 threshold. | Missed beats, jumpy BPM. | Phase 0, task 4 |
| D6 | `shaders/visualizer.kage:204–329` | Each scene sample evaluates all three acts plus a texture fetch; up to 80 march + 6 normal + 5 AO + 12 shadow ≈ 103 samples/pixel at full res. | GPU spent on morphing blobs; nothing left for light. | Phase 1 (port at half res, evaluate only the active act) |
| D7 | `shaders/visualizer.kage:225` | `return d - artRelief` breaks the SDF distance bound. | Noisy surfaces, extra march steps. | Phase 1 |
| D8 | `gfx/library.go:154` | Square cover stretched to window aspect. | Distorted artwork. | Phase 0, task 6 |
| D9 | `shaders/postprocess.kage:46` | "Bloom" = 8 taps at 1.5 px on an 8-bit buffer. | No glow; highlights clip. | Phase 1 |
| D10 | `audio/metadata.go:61` | `powershell` launched every 4 s. | CPU spikes; up to 4 s late; no position or pause state. | Phase 0, task 5 |
| D11 | `main.go:262`, `gfx/postprocess.go:56` | New `map[string]any` uniforms every frame. | GC churn, micro-stutter. | Phase 0, task 6 |
| D12 | `shaders/pulse.kage` | Never loaded anywhere. | Dead code. | Phase 0, task 6 (delete or wire up) |

Note: the capture code *does* use the device's real sample rate (`capture.go:71–73`, `NewAnalyzer(512, sampleRate)`);
the problem is the FFT size, not the rate.

## What to keep

- Go as the application language, and the "zero C toolchain" build.
- WASAPI loopback capture (improved in Phase 0).
- The artwork fetcher (becomes a Forge fallback) and palette analysis (move to OKLab in Phase 4).
- Hot-reload as a principle (shaders now; show files and WGSL later).
- The three SDF acts: they become one "abstract" world in the new renderer (Phase 1).
