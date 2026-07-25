# Audio-reactive visualizer — Rust + wgpu implementation plan

## Goal

A native Windows app that captures whatever's playing system-wide (any player, no plugin), analyzes it in real time, and drives a GPU shader pipeline that warps and blends a library of pre-generated images into music-reactive visuals — at true real-time frame rate, no neural net inference in the loop.

## Why this stack

`wgpu` gives you a modern, safe, cross-platform GPU API (Vulkan/DX12/Metal under the hood) without hand-rolling Vulkan yourself. `winit` handles the window and event loop. Everything else is small, focused crates rather than a framework — you own the whole render loop, which is the point of going "raw" instead of openFrameworks or an engine.

## Dependencies

Patch versions below are illustrative — crate versions here move fast, so run `cargo add <name>` for each and let it grab whatever's current; check migration notes if it lands a major version ahead of what's shown.

```toml
[package]
name = "reactive-visualizer"
version = "0.1.0"
edition = "2021"

[dependencies]
wgpu = "29"
winit = "0.30"
pollster = "0.3"
bytemuck = { version = "1", features = ["derive"] }
image = "0.25"
rustfft = "6"

[target.'cfg(windows)'.dependencies]
wasapi = "*"   # HEnquist/wasapi-rs — safe WASAPI bindings with loopback capture built in
```

One deliberate choice here: `cpal` is the more commonly-recommended cross-platform audio crate, but its WASAPI loopback support has had a rocky history (added, removed, reported as unreliable across several open issues). Since this project is Windows-first and loopback is the one thing you actually need, going straight to the `wasapi` crate — which lists loopback capture as a supported, first-class feature — is the more reliable path. You'd only reach for `cpal` if you later want cross-platform (macOS/Linux) capture too.

## Project layout

```
src/
  main.rs              # entry point, owns the winit event loop
  app.rs               # ApplicationHandler impl — ties audio + render together
  audio/
    mod.rs
    capture.rs         # WASAPI loopback capture, runs on its own thread
    analysis.rs        # windowing, FFT, band energies, onset detection
  gfx/
    mod.rs
    device.rs          # wgpu instance/adapter/device/queue/surface setup
    pipeline.rs         # render pipeline + bind group layouts
    feedback.rs          # ping-pong texture pair for the feedback loop
    library.rs             # loads a folder of images into GPU textures
  shaders/
    composite.wgsl
```

## Threading model

Audio capture and analysis run on their own thread, decoupled entirely from the render loop, and hand off state through a plain mutex:

```rust
#[derive(Clone, Copy, Default)]
pub struct AudioFeatures {
    pub bass: f32,
    pub mid: f32,
    pub treble: f32,
    pub onset: f32,   // 0..1, spikes on a detected beat, decays otherwise
}

pub type SharedAudio = std::sync::Arc<std::sync::Mutex<AudioFeatures>>;
```

The audio thread overwrites this after every analysis hop (roughly 90×/sec at a 512-sample hop). The render thread reads it once per frame (60–144×/sec) before building the shader's uniform buffer. On a four-float struct at these rates, lock contention is genuinely not a concern — don't reach for a lock-free ring buffer or triple-buffer until you've actually measured a problem. It's tempting to over-engineer this part; resist it.

## Milestones

Build in this order — each one is runnable and demoable on its own, which matters for staying motivated on a project this size:

0. **Scaffold** — winit window + a wgpu clear-to-color loop, no audio yet. Confirms the graphics half boots.
1. **Capture** — `wasapi` loopback thread, print RMS level to the console. No rendering tie-in yet. Confirms the audio half works in isolation.
2. **Analysis** — FFT + three-band energy split + onset detection, feeding the `AudioFeatures` struct above.
3. **First reaction** — wire bass energy into a single uniform (background brightness pulsing, say). This is the "hello world" that proves the whole pipeline end to end.
4. **Image library** — load a folder of images into GPU textures, hard-cut between two of them on a timer.
5. **The look** — add the ping-pong feedback pair and the warp/blend shader below. This is the milestone where it starts to feel like the thing you're actually after.
6. **Reactive switching** — swap source images on onset or energy-level changes instead of a fixed timer.
7. **Polish** — fullscreen / second-monitor output, hotkeys (pause, skip, reload folder), a config file for the image path.

## The composite shader (WGSL, not GLSL)

Since wgpu speaks WGSL rather than GLSL, here's the same feedback + warp + blend idea translated properly:

```wgsl
struct Uniforms {
    time: f32,
    bass: f32,
    onset: f32,
    _pad: f32,
};

@group(0) @binding(0) var<uniform> u: Uniforms;
@group(0) @binding(1) var feedback_tex: texture_2d<f32>;
@group(0) @binding(2) var source_tex: texture_2d<f32>;
@group(0) @binding(3) var samp: sampler;

fn hash(p: vec2<f32>) -> f32 {
    return fract(sin(dot(p, vec2<f32>(12.9898, 78.233))) * 43758.5453);
}

@fragment
fn fs_main(@location(0) uv: vec2<f32>) -> @location(0) vec4<f32> {
    let n = vec2<f32>(hash(uv * 3.0 + u.time), hash(uv * 3.0 - u.time)) - 0.5;
    let warped = uv + u.bass * 0.02 * n;
    let trail = textureSample(feedback_tex, samp, warped).rgb;
    let next  = textureSample(source_tex, samp, uv).rgb;
    let blend = smoothstep(0.0, 1.0, u.onset);
    return vec4<f32>(mix(trail * 0.96, next, blend), 1.0);
}
```

Each frame: render this into whichever ping-pong texture isn't the current `feedback_tex`, present that result, then swap which texture counts as "current" for next frame. Both textures need `TEXTURE_BINDING | RENDER_ATTACHMENT` usage since each one alternates between being sampled from and rendered into.

## wgpu / winit specifics worth knowing before you start

- Device and adapter creation is async (`request_adapter`, `request_device`). You don't need a full async runtime for this — `pollster::block_on` at startup is the standard, simple pattern used throughout the wgpu ecosystem.
- winit 0.30+ moved to the `ApplicationHandler` trait (`resumed`, `window_event`, `about_to_wait`) instead of the older closure-based `event_loop.run(|event, ...| ...)`. Plenty of older wgpu tutorials online still show the closure style — if you're following one, expect to adapt the event-loop boilerplate.
- Don't hardcode the swapchain's pixel format — query it (`surface.get_capabilities(&adapter).formats[0]`), since it varies by backend (Vulkan/DX12/Metal).
- `PresentMode::Fifo` caps you to the monitor's refresh rate (vsync on); `Mailbox` renders as fast as possible and drops frames. For a visualizer, Fifo is almost always the right choice — there's no reason to render faster than the screen can show.

## Audio analysis sketch

```rust
// called once per hop (e.g. every 512 samples) with a mono, windowed buffer
fn analyze(samples: &[f32], fft: &dyn rustfft::Fft<f32>) -> AudioFeatures {
    // apply a Hann window, run the FFT, take magnitudes
    // bucket magnitudes into bass (<250Hz) / mid (250Hz–2kHz) / treble (>2kHz)
    // track a rolling average per band; onset = how far current bass energy
    // spikes above its own recent rolling average, clamped to 0..1
}
```

This is the one part worth tuning by ear once it's running — the onset threshold and decay rate matter more for "feels right" than any exact formula does.

## What's deliberately not on this list

- A video codec/decoder — you're compositing still images, not playing video files, so there's nothing to decode.
- A scripting layer for scenes — a folder of images plus a couple of enums for "which shader mode is active" will get you further than you'd expect before you actually need real configurability.

## Next step

Milestone 0 — the winit window and wgpu boot sequence — is the natural first file to actually write. Say the word and we can start on `main.rs` directly.
