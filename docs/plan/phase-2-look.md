# Phase 2: The look (3–4 weeks)

## Goal

Make frames that look like a stage show: a monumental chrome hero in haze, crisp reflections, lasers, particles,
cinematic lenses and grading. Deliver the first complete world, **"Chrome Colossus"**. Read `art-direction.md` before
starting; it defines what "good" means here.

## Prerequisites

Phase 1 done (WebGPU renderer with HDR, bloom, finish pass, governor).

## Tasks

### Task 1: Mesh pipeline

- glTF 2.0 loader (`github.com/qmuntal/gltf`), with meshopt-compressed buffers and KTX2/BasisU textures
  (transcode to BC7 on desktop).
- PBR material: base color, metal/roughness, normal, emissive, clearcoat. GGX specular, Lambert diffuse.
- Instancing for repeated objects (crowd, shards), GPU frustum culling in a compute pass, indirect draws.
- Depth prepass + forward shading (a clustered/forward+ light list is only needed past ~16 lights).

### Task 2: Chrome needs something to reflect (the most important task)

- Image-based lighting: a prefiltered environment cubemap (GGX mip chain) + split-sum BRDF LUT. Source HDRIs from
  Poly Haven (CC0) and from the Forge later.
- **Light cards**: large emissive strips (like a car commercial's softboxes) placed around the hero. Render them into
  the environment (for reflections) *and* evaluate them as analytic rectangular area lights with LTC (Heitz et al.
  2016) for correct highlights on metal. Mostly black environment + a few thin, very bright strips = the look.
- The demo shows the effect: a dark environment with thin strips makes chrome read as chrome; a bright uniform
  environment makes it read as gray plastic.

### Task 3: Haze (froxel volumetrics)

- A 160×90×128 froxel grid (view-aligned), compute pass: density (global + noise + height falloff), in-scattering from
  spot lights and lasers with their shadow maps, Henyey-Greenstein phase (g ≈ 0.6).
- Temporal reprojection with 5–10% new-frame blend; jitter the sample depth with blue noise.
- Integrate front to back; apply in the composite pass.
- Lasers: thin line lights; add in-scattering analytically along the beam, plus a thin emissive core with bloom.

### Task 4: Shadows

Spot light shadow maps (2–3 lights), cached: static geometry once, dynamic objects redrawn only when they move. PCF or
PCSS for soft edges.

### Task 5: Particles

- 1–4 M particles in storage buffers. Compute update: curl noise, attraction to a target (mesh surface samples or an
  SDF volume), beat impulses from the director.
- Render as soft sprites (additive or premultiplied), motion-stretched along velocity.
- "Disintegration": sample the hero mesh surface into particle targets (precomputed); dissolve = interpolate particles
  from surface to a noise field; re-form = spring back on the downbeat.

### Task 6: Lens and camera

- Physical camera: focal length (mm), sensor 36×24 mm, aperture (f-stop), focus distance.
- Bokeh DOF at half resolution (gather-based), per-object motion blur from a velocity buffer, anamorphic streaks on
  the brightest pixels (horizontal blur of the bloom threshold image), halation (reddish glow around highlights).
- TAA or FSR 2 (it also antialiases). No jaggies anywhere.

### Task 7: Vertex animation textures (VAT)

- Offline, in Houdini (SideFX Labs VAT) or Blender, bake: a shatter of the monolith, cloth wings unfolding, a slow
  "breathing" deformation of the hero. Store positions/normals in RGBA16F textures.
- Runtime: a vertex shader samples the frame; the director scrubs the frame (e.g. shatter progress tied to a cue).

### Task 8: World "Chrome Colossus"

Contents (placeholders acceptable at first, then a real hero asset):

- Hero: a colossal chrome android bust (or monolith for the first cut) with emissive seams/eyes; poses: idle, awaken
  (eyes ignite), head turn left/right. Sources in `references.md` (licensed models, MetaHuman, commissioned).
- Halo arcs behind the hero, a floor ring, a dark glossy stage floor.
- 100+ tiny human silhouettes at the base for scale.
- 6 moving-head beams on a truss, a laser fan behind the hero, a god-ray top light.
- Per-world LUT: black, white and one accent (ice white in Drop 1, laser red in Drop 2, amber in breakdowns).
- A world is defined by a data file `shows/worlds/chrome_colossus.yaml` listing assets, light rig, camera rigs and
  looks, so new worlds need no code.

### Task 9: Look-dev harness

- `-lookdev` mode: free camera, sliders (via a small ImGui-like overlay or a web control page on localhost) for every
  light, look and haze parameter; save presets into the world file.
- Screenshot sequence export at 4K for reviews.

## Acceptance criteria

- [ ] Chrome Colossus renders at 1440p ≤ 8.3 ms p99 on the target GPU, all passes on.
- [ ] Blind A/B: 5 people compare 10 stills from the app with 10 reference stills from pro shows; the app is not
      reliably picked as "amateur" (≤ 60% correct identification).
- [ ] No banding, no aliasing shimmer on the laser fan or halo arcs in a 30 s recording.
- [ ] All look parameters editable live and saved to the world file.

## Pitfalls

- Too much haze and bloom turns everything gray (the demo went through this). Keep true black in most of the frame.
- Chrome without light cards looks like gray plastic.
- Beams must fade out near the camera, or close-ups get a milky veil.

## Results

(Fill in.)
