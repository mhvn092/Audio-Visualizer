# Phase 3: The director (3 weeks)

## Goal

Visuals scheduled like a lighting designer's cue list: sections, a drop forecast, phrase-quantized cuts, a reaction
budget, latency compensation, and Song Memory for timecode-exact replays.

## Prerequisites

Phase 0 (beat tracker, record/replay). Phase 2 is recommended (so cues drive a real world) but the director can be
built against the legacy renderer by mapping cues to existing uniforms.

## Tasks

### Task 1: Section classifier and drop forecast (`internal/ear/sections.go`)

Inputs per bar: kick presence, bass energy (40–120 Hz), loudness, centroid, flatness, snare density, novelty (distance
between consecutive bars' mean mel vectors).

Rules to start (then tune on the test set; an ML model can replace them later):

- **Build**: rising centroid and flatness over ≥ 4 bars (risers are noisy and brighten), snare density increasing
  (quarters → eighths → sixteenths), kick often removed in the last 1–2 bars.
- **Drop**: kick + bass return at full energy on a phrase boundary after a Build or Break; big novelty spike.
- **Break**: kick absent ≥ 2 bars, loudness down, harmonic content up.
- **Intro/Outro**: position in track (from media timeline when available) + low density.
- **Drop forecast** `DropIn[1,2,4 bars]`: high when in Build for ≥ 4 bars, riser slope positive, and the next phrase
  boundary is within the horizon. A 1-bar "gap" (kick + bass out) is a strong signal for a drop at the next downbeat.
- Phrase boundaries: every 8 bars from the first confident downbeat, re-synced when novelty spikes on a downbeat.

### Task 2: Show grammar (hot-reloaded data files in `shows/`)

```yaml
# shows/grammar.yaml
reaction_budget:          # which musical element drives which channel
  kick: key_light
  snare: strobe           # rate-limited, see safety
  hat: lasers
  vocal: hero_emissive
  bass: haze_density
intensity_levels:         # how many channels are live per section
  intro: [key_light]
  build: [key_light, strobe, hero_emissive]
  drop: [key_light, strobe, lasers, hero_emissive, haze_density]
  break: [hero_emissive]
shots:
  monumental_wide: { rig: orbit, radius: 17, height: 2.6, fov: 34, sweep_deg: 48 }
  seam_closeup:    { rig: crane, from: [1.5, 3.0, 3.6], to: [0.9, 5.4, 2.7], fov: 30, look: hero_seam, env_scale: 0.35 }
  crowd_pov:       { rig: handheld, pos: [2.2, 0.55, 13.5], look: [0, 6.2, 0], fov: 46, shake: 0.04 }
  overhead:        { rig: dolly, from: [0.6, 25, 7.5], to: [-0.6, 21, 6], fov: 42 }
  telephoto_side:  { rig: dolly, from: [16, 2, 2], to: [13.5, 2.8, 5], fov: 22 }
  reveal_pushin:   { rig: dolly, from: [0, 0.9, 36], to: [0, 1.8, 20], fov: 28 }
  blackout:        { rig: hold, exposure: 0.1, rim_only: true }
playbook:                 # per section: shot order and cue rules
  build:  { shots: [crowd_pov, telephoto_side, seam_closeup], last_half_bar: blackout, beams: converge, color: desaturate }
  drop:   { first_cut: monumental_wide, flash_on_beat1: true, cut_every_bars: 4, shots: [monumental_wide, seam_closeup, overhead, crowd_pov] }
  break:  { shots: [telephoto_side, seam_closeup], speed: 0.5, dof: shallow, hero: dissolve }
```

The exact numbers above come from the demo (`demo/colossus-demo.html`, functions `state()` and `camera()`), which is
the reference implementation of this playbook.

### Task 3: Scheduler (`internal/director/scheduler.go`)

- Tick at 240 Hz. Keep a queue of cues with musical timestamps (bar, beat).
- Cuts only on phrase boundaries, except: the drop cut on beat 1, and forced cuts when confidence collapses.
- The last half bar before a forecast drop: blackout (rim light only). This "breath" is what makes the drop hit.
- Look ahead one phrase: ask the renderer to `Prewarm` the next world/shot, and the Forge for anything missing.
- Low tempo confidence (< 0.5): **free-flow mode**: no hard cuts, slow camera, reaction via smooth envelopes only.

### Task 4: Camera rigs

Dolly, crane, orbit, handheld (layered low-frequency noise, ≤ 5 cm), hold. Moves eased (smoothstep), start on a
downbeat, end on the phrase boundary. The renderer interpolates between director ticks.

### Task 5: Latency calibration

- A calibration screen: a click in the audio and a white flash on screen at the same scheduled time; the user adjusts
  a slider until they coincide (or uses a phone mic app / microphone auto-detection later).
- Store `audio_output_latency_ms` and `display_latency_ms` in `config.json`. Default: WASAPI reported latency + one
  frame + 15 ms.
- The director publishes frames for `now + displayLatency - audioLatency` in music time.

### Task 6: Song Memory (`internal/vault/songmemory.go`)

- Key: normalized `artist|title|duration_s` (lowercase, punctuation stripped).
- Store after each full play: beats, downbeats, sections, drops, energy curve, tempo map (SQLite via
  `modernc.org/sqlite`, pure Go).
- On replay: use the media-session timeline position for coarse alignment, then cross-correlate the live onset
  envelope with the stored one over ±2 s to refine to < 10 ms. Then `Clock.FromMemory = true`: the director knows every
  upcoming drop exactly.
- After a first play, the Forge may run offline structure analysis (All-In-One) on a buffer of the captured audio to
  improve the memory (only in memory, never saved as audio).

### Task 7: Safety limiter

A single `strobe` output stage that enforces ≤ 3 flashes per second (target 2) and a maximum luminance swing,
regardless of what any cue asks for. User switch to disable strobe; default off when the OS reduced-motion setting is on.

## Acceptance criteria

- [ ] On the 20 four-on-the-floor test tracks, first play: ≥ 80% of drops get their cut within ±1 frame of beat 1;
      replay (Song Memory): 100%.
- [ ] Drops are flagged ≥ 1 bar ahead in ≥ 80% of cases on first play.
- [ ] No cut mid-phrase except drop cuts (automated check over replay logs).
- [ ] Measured visual-to-audio offset after calibration: |offset| < 10 ms (high-speed phone video of screen + speaker).
- [ ] Strobe limiter verified: never > 3 flashes in any 1 s window (automated check over a replay).

## Results

(Fill in.)
