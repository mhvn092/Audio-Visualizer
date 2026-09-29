# 01: Target architecture

This is the end state. Each phase builds part of it; the interfaces below are introduced in Phase 0 and 1 so the old
code can be swapped out piece by piece.

## Subsystems and threads

| Subsystem | Runs on | Rate | Must never |
|---|---|---|---|
| EAR (capture + analysis) | Dedicated OS-locked goroutine (`runtime.LockOSThread`) | Every audio packet (~5–10 ms) | Block, allocate, take a mutex the render thread holds |
| DIRECTOR | Its own goroutine with a ticker | 240 Hz | Touch GPU objects |
| RENDERER | Main thread (window/GPU owner) | Display rate (60–144 Hz) | Wait on disk, network or ML |
| FORGE | Worker pool goroutines + optional ML sidecar / NPU / cloud | Background | Use the render GPU during drops |
| VAULT | Library used by all (disk + in-memory index) | On demand | Be read synchronously from the render loop (stream instead) |
| ROOM | Goroutine sending UDP (DMX/WLED/Hue) | 40–60 Hz | Drift from the frame the viewer sees (latency-compensated) |

Communication between threads uses **lock-free single-writer snapshots** (triple buffer or `atomic.Pointer[T]` to an
immutable struct). No shared maps, no mutexes on hot paths.

## Proposed package layout

```
main.go                  // wiring only: create subsystems, run the loop
internal/ear/            // capture.go, stft.go, features.go, tempo.go, sections.go, memory.go, media_windows.go
internal/clock/          // MusicalClock types shared by ear/director/renderer
internal/director/       // grammar.go (show file types), scheduler.go, shots.go, cues.go, playbook.go
internal/render/         // Renderer interface; render/ebiten (legacy), render/wgpu (new)
internal/render/wgpu/    // device.go, framegraph.go, passes/*.go, shaders/*.wgsl
internal/forge/          // jobs.go, scheduler.go, gate.go, generators/*.go
internal/vault/          // store.go (content-addressed), assets.go, songmemory.go (SQLite)
internal/room/           // dmx.go (Art-Net/sACN), wled.go (DDP), hue.go (Entertainment API)
internal/replay/         // record/replay of feature streams for tests
shows/                   // hot-reloaded show grammar files (YAML/JSON)
assets/                  // user + downloaded assets (git-ignored), vault lives in assets/vault/
```

Moving existing code: `audio/` → `internal/ear/`, `gfx/` → split into `internal/render/ebiten` and `internal/forge`.
Do the moves in Phase 0 as mechanical commits with no behaviour change.

## Core data types (Go sketches; refine while implementing)

```go
// internal/clock
type Features struct {
    T          time.Duration // audio timestamp (QPC-derived) of the analysis frame
    Bands      [64]float32   // log-mel magnitudes, normalized 0..1
    Kick, Snare, Hat, Vocal float32 // per-element onset strength 0..1 (spectral flux, adaptive threshold)
    Loudness   float32       // short-term loudness, normalized
    Centroid   float32       // brightness 0..1
    Flatness   float32       // noisiness 0..1 (risers are noisy)
    Chroma     [12]float32
    Silent     bool
}

type Clock struct {
    T            time.Duration // audio time this snapshot describes
    BPM          float32
    Beat         int64   // beats since track start (or since lock)
    BeatPhase    float32 // 0..1 within the current beat, phase-locked to onsets
    Bar, BeatInBar int   // 4/4 assumed; BeatInBar 0..3
    Phrase       int     // index of 8-bar phrase
    BarInPhrase  int
    Section      SectionKind // Intro, Build, Drop, Break, Outro, Unknown
    SectionConf  float32
    DropIn       [3]float32  // probability of a drop within 1, 2, 4 bars
    TempoConf    float32     // below ~0.5, director switches to free-flow mode
    FromMemory   bool        // true when aligned to Song Memory (replay)
}

// internal/director
type Frame struct { // everything the renderer needs for one frame; immutable once published
    MusicTime   float64 // in beats, latency-compensated for display time
    Shot        ShotState  // camera position/target/fov/aperture, rig id
    World       WorldID
    Lights      [8]LightCue
    Lasers      LaserCue
    Strobe      float32 // already rate-limited (≤ 3 flashes/s; target 2)
    Hero        HeroCue // emissive level, pose blend, dissolve 0..1
    Particles   ParticleCue
    Look        LookCue // exposure, bloom, LUT id, grain, haze density
}
```

## Interfaces

```go
// internal/render
type Renderer interface {
    Init(win Window) error
    Load(ctx context.Context, a vault.AssetRef) error   // async upload; never blocks a frame
    Prewarm(w director.WorldID) error                   // compile pipelines / stream assets ahead of a cut
    Draw(f *director.Frame) (Stats, error)              // one frame
    Resize(w, h int)
    Close()
}

type Stats struct { GPUms map[string]float32; CPUms float32; RenderScale float32 }

// internal/ear
type Source interface { // live capture or replay file
    Start(out chan<- []float32, ts chan<- time.Duration) error
    Stop()
}

// internal/forge
type Job struct {
    Key      vault.Key      // hash of inputs; identical jobs are deduplicated
    Kind     string         // "palette", "diorama", "sky", "sculpture", "showscript", ...
    Deadline time.Time      // from the musical forecast; 0 = whenever
    Priority int
    Run      func(ctx context.Context) (vault.AssetRef, error)
}
```

## Data flow for one frame

1. EAR publishes `Features` and `Clock` snapshots (atomic pointer swap).
2. DIRECTOR, every 1/240 s, reads the latest `Clock`, advances its schedule, and publishes a `Frame` for time
   `now + displayLatency` (so what is shown matches what is heard).
3. RENDERER reads the latest `Frame`, interpolates camera between director ticks, draws, and publishes GPU timings.
4. ROOM reads the same `Frame` and sends light values stamped for the same display time.
5. FORGE runs jobs requested by the DIRECTOR (via a channel) and writes results into the VAULT; the DIRECTOR swaps
   them in on the next phrase boundary after the RENDERER reports them loaded.

## Latency model

`audible time = capture time + device output latency` (WASAPI loopback captures what is being sent to the device, so
the sound reaches the speakers slightly after we see it). `visible time = render time + swapchain latency + display lag`.
The director schedules cue `c` at music time `m` to be *drawn* at wall time
`wall(m) - (displayLatency - audioOutputLatency)`. Both latencies are measured once by the calibration screen in
Phase 3 and stored in the config file.
