# Phase 1: Renderer spike and migration to WebGPU (2–3 weeks)

## Goal

Replace Ebiten + Kage with a WebGPU renderer driven from Go, reach visual parity with today's three SDF acts, and put
the HDR pipeline, a real bloom chain and a frame-budget governor in place.

## Why

Ebiten is a 2D engine. In v2.9.9 images are 8-bit RGBA only, a shader gets at most 4 input images, and there is no
depth buffer, compute, instancing, storage buffers, multiple render targets or 3D textures. The target look (Phase 2)
needs all of those.

## Decision

| Option | Verdict |
|---|---|
| Ebiten + Kage | Keep only until the new renderer reaches parity |
| **Go + WebGPU (wgpu-native via zero-cgo bindings)** | **Recommended.** Primary: `github.com/go-webgpu/webgpu` (wraps wgpu-native v29 through `goffi`, no C compiler). Fallback: `github.com/gogpu/wgpu` (pure Go, or `-tags rust` for wgpu-native). |
| Unreal Engine 5 as the runtime | No: second codebase in C++, huge runtime. Use UE5/Blender **offline** to make content (Phase 2/4). |
| TouchDesigner / Notch | No: closed runtimes, not shippable. Fine for look-dev sketches. |

Dependency caution: `goffi` and `ebitengine/purego` both define `fakecgo` symbols. While Ebiten and the new renderer
coexist, or when adding `onnxruntime-purego` later, you may hit duplicate-symbol linker errors. Mitigations: keep the
legacy renderer in a separate build tag (`-tags legacy`), or use the `pureffi` wrapper the goffi authors recommend.

## Prerequisites

Phase 0 done (clean `Features`/`Clock` snapshots, no races).

## Tasks

### Task 1: One-week spike (go/no-go)

Build `cmd/spike` (standalone) on your Windows GPU with `go-webgpu/webgpu`:

1. Window + surface + swapchain (FIFO present, max 1 frame in flight). Use the binding's surface helpers or a minimal
   Win32 window via `golang.org/x/sys/windows`.
2. An RGBA16F offscreen target; draw a lit mesh with a depth buffer.
3. A compute pass updating 1,000,000 particles in a storage buffer; draw them with instancing/indirect draw.
4. The bloom chain from task 4 and the finish pass from task 5.
5. GPU timestamp queries per pass, printed once per second.
6. WGSL hot reload: recompile pipelines on file change **asynchronously** and swap on success.

Go/no-go: if 1440p holds 120 fps with all of that on the target GPU and there are no binding crashes over 30 minutes,
proceed with `go-webgpu`. Otherwise try `gogpu/wgpu`, then cgo bindings (`cogentcore/webgpu`). Write the decision and
numbers in Results.

### Task 2: Renderer interface and strangler migration

- Introduce `internal/render.Renderer` (see `01-architecture.md`). Wrap the current Ebiten code as
  `internal/render/ebiten` implementing it. `main.go` talks only to the interface.
- Add `internal/render/wgpu` implementing the same interface. Select with a flag: `-renderer=wgpu|ebiten`.
- The app keeps working with `-renderer=ebiten` until parity is confirmed, then Ebiten is removed.

### Task 3: Frame graph

A small frame graph: named passes with declared inputs/outputs, resources allocated once per resolution, per-pass GPU
timestamps. Initial passes: `sdf` (half res) → `composite` → `bloom` → `finish` → swapchain.

### Task 4: HDR + bloom chain

- All scene targets RGBA16F.
- Bloom: 6–8 mip downsample chain with a 13-tap filter and Karis average on the first downsample (the Call of Duty:
  Advanced Warfare method, Jimenez 2014), then tent-filter upsample with additive blend. Threshold with a soft knee.
  Energy-conserving: `final = mix(scene, bloom, strength)` with strength ~0.04–0.1, not an additive blow-out.

### Task 5: Finish pass

AgX tonemap (see Blender/three.js implementations), exposure, 3D LUT (33³, RGBA16F 3D texture) per world, subtle
chromatic aberration (≤ 0.5% at the corners), luma-weighted film grain changing at 24 Hz, vignette, and
**blue-noise dither** (a 64×64 blue-noise texture, offset per frame) before the 8-bit write. Banding in dark gradients
is the most visible amateur tell; this pass must remove it.

### Task 6: Port the three SDF acts to WGSL (fixes D6, D7)

- Evaluate **only the active act** (branch on a uniform). Transitions happen in screen space (crossfade, glitch cut,
  flash), never by blending distance fields.
- Render at ½ resolution; upscale with a depth-aware bilateral filter plus temporal accumulation (reproject last frame).
- Replace the sin-hash `vnoise` with a tiling 3D noise texture (or a 2D atlas of slices); precompute once.
- Add bounding-sphere early exit per ray; use relaxed sphere tracing (step factor 1.2 with fallback on overshoot).
- Drop the album-art "relief" from the distance field; apply artwork as a triplanar-projected material instead.

### Task 7: Budget governor

- Target frame time per quality tier (e.g. 8.3 ms at 120 Hz). Read GPU timestamps each frame; EMA.
- If the EMA exceeds the budget for 0.5 s, reduce render scale by 10% (min 50%); if under 70% of budget for 3 s,
  increase by 5%. Temporal upscaling (FSR 2, MIT licence, or a simpler TAAU) reconstructs the output.
- Quality tiers: Low (iGPU, 1080p60), Medium, High (1440p120), Ultra (4K60). Tier chosen on first run by a 5 s benchmark.

### Task 8: Frame pacing

FIFO present, one frame in flight, no per-frame allocations (preallocate uniform buffers; write with
`queue.WriteBuffer`), pipelines created asynchronously and prewarmed before use. Measure p99 frame time.

## Frame budget (1440p at 120 Hz on an RTX 4070-class GPU; estimates to verify)

| Pass | ms |
|---|---|
| Compute prelude (audio textures, particles, VAT) | 0.6 |
| Depth prepass + opaque meshes | 1.4 |
| Shadows (2–3 cached spots) | 0.6 |
| SDF raymarch at ½ res | 1.2 |
| Froxel haze + shafts + lasers | 1.0 |
| Particles (1–4 M) | 0.8 |
| Temporal upscale | 0.5 |
| DOF + motion blur | 0.6 |
| Bloom + lens | 0.4 |
| Finish (AgX, LUT, grain, dither) | 0.2 |
| **Total** | **7.3 of 8.3** |

Gaussian-splat worlds (Phase 4) cost about 2 ms for 1–2 M splats and replace the set geometry in scenes that use them.
Phases 1 only needs the SDF, bloom and finish rows; the others arrive in Phase 2.

## Acceptance criteria

- [ ] Spike decision recorded with numbers.
- [ ] `-renderer=wgpu` shows the three acts at visual parity (side-by-side screenshots in Results).
- [ ] 1440p: p99 frame time ≤ 8.3 ms on the target GPU; 1080p60 on an integrated GPU at Low tier.
- [ ] No visible banding on a 0–5% gray ramp test pattern (`-testpattern=ramp`).
- [ ] WGSL hot reload works without a frame hitch > 2 ms.
- [ ] Ebiten removed from the default build.

## Pitfalls

- Shader compile stalls: always compile off the render thread and swap when ready.
- sRGB: render linear into RGBA16F; only the swapchain view is sRGB (or encode manually in the finish pass).
- Keep the governor from oscillating: hysteresis and minimum hold times.

## Results

(Fill in.)
