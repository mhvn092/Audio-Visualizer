# Colossus Engine Plan: start here

This folder is the complete plan for turning this project (a Go audio visualizer, see the root `README.md`) into a
stage-grade, "Anyma-level" visual engine. It is written so that someone with **no prior context** can pick it up and
start working. Read this file first, then the phase file you are working on.

> Live companion page with a working demo of the target look and choreography:
> `docs/plan/demo/colossus-demo.html` (open it in a desktop browser, press **Play with sound**).

---

## 1. The goal in one paragraph

Anyma's shows (Afterlife, "Genesys", "The End of Genesys" at the Las Vegas Sphere) look expensive for two reasons:
**the content is made long before the show** (Cinema 4D, Houdini and Unreal Engine 5, rendered at up to 16K on cloud
farms, by 100+ people over about four years), and **every visual event lands on musical time** (the show runs to
timecode through a show-control system called disguise). We cannot out-render that team live. We copy their *method*
and win where a fixed show cannot: **every song you play gets its own directed show**, generated and refined on your own PC.

## 2. The four rules (every design decision must follow these)

1. **Pre-produce anything expensive.** Simulations, lighting, geometry, textures and whole worlds are made in advance
   and stored in an asset bank (the "Vault"). At runtime the engine only plays, relights, composites and directs them.
2. **Schedule on musical time; never just react to loudness.** A musical clock (beat, bar, phrase, section, drop
   forecast) triggers every cue. Cues fire early by the measured audio/video latency so they land with the sound.
   Replays use a stored song timeline ("Song Memory"), like a timecoded show.
3. **Generate ahead, never inside the frame.** A background "Forge" makes new material for each track against
   deadlines ("needed by drop 2 in 95 s"), passes it through a quality gate, and swaps it in on a phrase boundary.
4. **Spend the frame on light.** HDR buffers, area-light reflections on chrome, haze, real bloom, grading, and a budget
   governor that keeps frame pacing smooth.

## 3. Glossary

| Term | Meaning |
|---|---|
| **EAR** | Audio intelligence subsystem: capture, features, beat/phrase/section tracking, drop forecast. |
| **Musical clock** | The EAR's output: beat index, beat phase (0..1), bar, phrase, section, confidence, forecasts. |
| **DIRECTOR** | Show brain: turns the musical clock into shots (camera), cues (lights, lasers, strobe, hero behaviour) and looks. |
| **RENDERER** | The GPU pipeline. Today Ebiten + Kage shaders; target WebGPU (wgpu-native) from Go. |
| **FORGE** | Background job system that generates assets (palettes, depth dioramas, skies, 3D models, show scripts). |
| **VAULT** | On-disk, content-addressed store of assets, generated content and Song Memory. |
| **Song Memory** | Per-track stored timeline (beats, downbeats, sections, energy) used for perfect sync on replays. |
| **Cue** | A scheduled visual event on musical time (e.g. "white flash on beat 1 of the drop"). |
| **Shot** | A camera rig + move lasting until the next cut (e.g. "Monumental wide", "Seam close-up"). |
| **World** | A set (hero, environment, lights, palette, LUT) the director can cut to. |
| **Reaction budget** | Rule that each musical element drives exactly one visual channel at a time. |
| **Phrase** | Group of 8/16/32 bars; in electronic music, sections change on phrase boundaries. |
| **Drop** | The high-energy section after a build; the most important moment to hit precisely. |
| **HDR** | Rendering with 16-bit float buffers so highlights can exceed 1.0 before tonemapping. |
| **VAT** | Vertex Animation Texture: a baked simulation stored in textures and played back on the GPU for free. |

## 4. Target architecture (summary; details in `01-architecture.md`)

```
 AUDIO IN (WASAPI loopback + media session)
      │ audio
      ▼
     EAR ──clock──► DIRECTOR ──cues──► RENDERER ──frames──► SCREEN
      ▲                │                  ▲   │
      │ song memory    │ jobs+deadlines   │   └─light cues─► ROOM (DMX / Hue / WLED)
      │                ▼                  │ streamed assets
      │              FORGE ──assets──► VAULT
      └──────────────────────────────────┘
```

Four clocks: the audio thread (hard real-time, never blocks), the director tick (240 Hz, musical time), the render loop
(display rate), and forge workers (background, deadline-driven).

## 5. Phases (do them in order)

| # | File | Goal | Rough size |
|---|---|---|---|
| 0 | [`phase-0-foundations.md`](phase-0-foundations.md) | Fix verified bugs, rebuild audio analysis, add record/replay + metrics HUD | 1 week |
| 1 | [`phase-1-renderer.md`](phase-1-renderer.md) | WebGPU spike, then migrate the renderer; HDR, bloom, governor | 2–3 weeks |
| 2 | [`phase-2-look.md`](phase-2-look.md) | The look: meshes, PBR, light cards, haze, lasers, particles, grading; first world "Chrome Colossus" | 3–4 weeks |
| 3 | [`phase-3-director.md`](phase-3-director.md) | Musical clock, sections, drop forecast, show grammar, shots, latency calibration, Song Memory | 3 weeks |
| 4 | [`phase-4-forge.md`](phase-4-forge.md) | Async generation: job system, Vault, cover dioramas, skies, Claude show writer, quality gate | 4+ weeks |
| 5 | [`phase-5-beyond-screen.md`](phase-5-beyond-screen.md) | DMX / Hue / WLED output, box mode with head tracking, dual screen, Spout/NDI | 2 weeks |
| 6 | [`phase-6-frontier.md`](phase-6-frontier.md) | Optional: real-time diffusion pass, ML stems and beat tracking | ongoing |

Durations are rough, for one developer working with AI assistance. **The app must keep running after every phase**:
new subsystems sit behind interfaces and replace old ones only when they reach parity.

Supporting documents:

- [`00-diagnosis.md`](00-diagnosis.md): why the current app looks amateur, with every verified defect and its file/line.
- [`01-architecture.md`](01-architecture.md): subsystems, threads, data types, Go interfaces, package layout.
- [`art-direction.md`](art-direction.md): the visual bible: palette, lighting recipe, choreography playbook, shot list, reaction budget, safety rules.
- [`references.md`](references.md): sources and libraries, with what each is used for.
- [`demo/colossus-demo.html`](demo/colossus-demo.html): the browser prototype (three.js) of the target look and director.

## 6. How to work on this plan

1. Pick the lowest-numbered phase that is not done (see the status checklist below).
2. Read that phase file top to bottom. Each has: **Goal**, **Why**, **Prerequisites**, **Tasks** (numbered, with files
   to touch and code sketches), **Acceptance criteria** (measurable), **Pitfalls**.
3. Work task by task on a branch. Each task should be a separate, reviewable commit that leaves the app runnable.
4. A phase is done only when every acceptance criterion is met and written down (numbers, screenshots) in the
   phase file's "Results" section at the bottom.
5. Tick the checklist below in the same commit that completes the phase.

Development environment: Windows 10/11 (the app captures Windows audio), Go 1.25+, a GPU with D3D12 or Vulkan.
Run with `go run .` from the repo root. Shaders and (later) show files hot-reload from disk.

## 7. Status checklist

- [ ] Phase 0: Foundations
- [ ] Phase 1: Renderer spike and migration
- [ ] Phase 2: The look
- [ ] Phase 3: The director
- [ ] Phase 4: The forge
- [ ] Phase 5: Beyond the screen
- [ ] Phase 6: Frontier (optional)

## 8. Non-negotiables

- **Photosensitivity safety:** never more than 3 full-screen flashes per second (the demo caps at 2); a global "strobe
  off" switch; respect the OS reduced-motion setting by defaulting strobe off. See `art-direction.md` §7.
- **No allocations in per-frame or audio-thread hot paths.**
- **Nothing expensive on the frame's critical path.** If it takes more than ~1 ms, it goes to the Forge.
- **Every generator is optional**, cached, and has a local fallback. The app must run offline with no API keys.
- **Record the licence of every third-party or generated asset** in the Vault.
