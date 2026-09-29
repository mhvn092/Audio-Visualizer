# Phase 4: The forge (4+ weeks)

## Goal

New material for every track, made ahead of need: a background job system that generates assets with deadlines, a
quality gate, and a content-addressed Vault, so each song gets its own show without ever hurting the frame rate.

## Rule

Nothing expensive runs on the frame's critical path. The Forge computes it ahead, caches it forever, and the director
swaps it in on a phrase boundary once the renderer reports it loaded.

## Prerequisites

Phase 3 (the director can issue requests with deadlines from the musical forecast). Phase 2 (renderer can load meshes,
textures, environment maps).

## Tasks

### Task 1: Vault (`internal/vault`)

- Content-addressed store under `assets/vault/`: key = SHA-256 of (generator name + version + inputs + parameters).
- Each entry: the files plus `meta.json` { kind, created, generator, inputs, licence, quality score, size }.
- An index in SQLite (same DB as Song Memory). LRU eviction above a size limit (default 20 GB), never evicting
  "pinned" packs.
- Asset formats: glTF + meshopt, KTX2/BasisU textures, RGBA16F VAT textures, SPZ Gaussian splats, `.cube` LUTs.

### Task 2: Job system (`internal/forge`)

- Priority queue ordered by deadline, then priority. Identical keys are deduplicated.
- Worker pools: `cpu` (N-1 cores), `ml` (1–2 slots; NPU/CPU via Windows ML or ONNX Runtime), `gpu` (1 slot, only
  runs when the director reports intensity ≤ 2, i.e. not during drops), `cloud` (HTTP, concurrency-limited).
- A job that will miss its deadline is cancelled and the director uses the library fallback. The frame never waits.
- Progress and results are reported on a channel the director reads.

### Task 3: Generators, by latency tier

| Ready in | Generator | Runs on | Output | Used |
|---|---|---|---|---|
| < 1 s | Cover palette in OKLab (k-means, sorted by chroma/lightness; replaces RGB k-means in `gfx/artanalysis.go`) | CPU | palette + LUT tint | from the first bar |
| < 1 s | Seeded variations: world parameters, SDF-form mutations from a small grammar, title as 3D type (SDF font) | CPU | parameter sets, meshes | from the first bar |
| 1–10 s | Cover depth (Depth Anything V2/3 small, ONNX) + segmentation → layered 2.5D diorama mesh | ML | glTF | by the first build |
| 1–10 s | Mood tags from the cover and metadata | ML/cloud | tags | show script input |
| 1–10 s | **Claude show writer** (task 4) | cloud | show script JSON | by the first build |
| 10 s–4 min | Image-to-3D sculpture from the cover (TRELLIS 2: 20 s–4 min, MIT licence) | GPU (quiet sections) or cloud | glTF | second drop, or next play |
| 10 s–4 min | 360° sky / HDRI from a text prompt (fast diffusion model, equirectangular) | GPU/cloud | HDR env map | reflections + backdrop |
| minutes–overnight | Gaussian-splat worlds (e.g. World Labs Marble API: 0.5–2 M splats as SPZ/PLY) | cloud | SPZ | content packs |
| offline | Hero passes rendered in Blender/Unreal; sims baked to VAT | artist machine | packs | content packs |

ONNX from Go without cgo: `github.com/shota3506/onnxruntime-purego`, or Windows ML (system ONNX Runtime with
NPU/GPU execution providers). If linker conflicts appear with the renderer's FFI library, run ML in a small sidecar
process (`cmd/forge-ml`) and talk over a local socket.

### Task 4: Claude show writer (optional; the app must work without it)

- On track change, send: title, artist, album, tempo, key/mode, energy-curve preview (from Song Memory if known), cover
  palette and mood tags, and the list of available worlds/shots/looks from `shows/grammar.yaml`.
- Ask for a JSON show script **chosen only from that vocabulary**: world per section, shot order, palette/accent,
  hero behaviour, and prompts for the sky/sculpture generators. Validate against a JSON schema; reject anything else.
- Implementation: official Go SDK `github.com/anthropics/anthropic-sdk-go`, model `claude-opus-5-5` at low effort,
  structured outputs (`output_config.format` with the JSON schema), prompt caching on the static vocabulary/system
  prompt. Read the SDK docs before writing the code; do not guess API names.
- API key from the environment (`ANTHROPIC_API_KEY`), never committed. Timeout 20 s; on failure or no key, the
  rule-based playbook runs.
- It never writes code or shaders.

### Task 5: Quality gate

- Every visual asset is rendered offscreen to a 512×288 thumbnail in its world.
- Score with: an aesthetic model (ONNX; e.g. a CLIP-based aesthetic predictor) + rule checks from `art-direction.md`:
  ≥ 60% of pixels below 10% luminance (keep it dark), a highlight range, max 1 saturated accent hue, no banding.
- Below threshold: rejected (kept in the Vault with its score for debugging, never shown).

### Task 6: Scheduling around the music

- Deadlines come from the forecast: e.g. "sculpture needed by drop 2" = predicted drop-2 time − 10 s load margin.
- GPU jobs pause during drops (director intensity ≥ 3) and resume in breakdowns.
- Idle-time pre-generation: when no music plays, pre-generate for the most recent 20 tracks in Song Memory.

## Acceptance criteria

- [ ] A never-seen track gets its first generated asset (palette/diorama) on screen by the first drop.
- [ ] With all generators running, no frame exceeds the budget because of generation (compare p99 with Forge on/off).
- [ ] Offline with no API keys: the app runs and uses library fallbacks.
- [ ] Repeat plays reuse cached results (no regeneration; verified in logs).
- [ ] Every asset has a licence recorded.

## Results

(Fill in.)
