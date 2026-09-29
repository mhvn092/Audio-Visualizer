# Art direction: the visual bible

What "Anyma-level" means in concrete, checkable terms. Use this to judge every frame, and as the rule set the
director and quality gate implement.

## 1. References to study

Anyma "Genesys" / "Genesys II" live shows, "The End of Genesys" at the Las Vegas Sphere (Dec 2024 – Jan 2025), and the
Afterlife label visuals. Recurring motifs: a humanoid android character (Eva), monumental scale against tiny humans,
chrome and glass materials, halos and rings, cathedral-like architecture, figures "breaking out" of the screen, slow
cinematic camera, hard cuts and white flashes on drops, near-monochrome palettes with one accent. Build a reference
board (`docs/plan/refs/`, stills you are allowed to use for internal comparison only) before Phase 2.

## 2. Palette

- Mostly **black**. At least 60% of the frame below 10% luminance in normal shots.
- **White** for light sources, chrome highlights and flashes.
- **One accent at a time** per section: ice white/blue, laser red, or warm amber. Never a rainbow.
- Accent source: the album cover palette (OKLab, most chromatic cluster), unless the show script overrides it.

## 3. Lighting recipe

1. Darkness first; add light only where the eye should go.
2. Chrome needs **light cards**: thin, very bright strips around the hero, rendered into reflections and as area lights.
3. **Haze** everywhere, but thin: beams and lasers must be visible, and the blacks must stay black.
4. **Rim light** behind the hero separates it from the background; in a blackout, rim light is the only light.
5. Bloom is a soft glow around sources, not a veil over the image.

## 4. Scale and composition

- Tiny human figures at the base (the demo uses 110) make anything monumental.
- Low camera angles for the hero; symmetry for "monumental wide"; telephoto (22–30 mm FOV-equivalent narrow) for
  compression; wide (40–46° FOV) for the crowd POV.
- Negative space: the hero rarely fills more than a third of the frame, except in close-ups.

## 5. Choreography playbook (what the director does per section)

| Section | Camera | Light | Hero | Special |
|---|---|---|---|---|
| Intro | One slow push-in from far and low | God ray from above fades in; key light slowly up | Dark, unlit seams | Dust motes, dense haze |
| Build | Crowd POV → telephoto side → seam close-up, getting closer | Beams converge on the hero; color drains toward white; strobe tracks the snare roll (rate-limited) | Seams flicker | Shards gather inward; **last half bar: blackout, rim light only** |
| Drop | Hard cut to monumental wide on beat 1; cut every 4 bars (wide, close-up, overhead, crowd) | White flash on beat 1; key light pulses on kicks; lasers on hats | Awakening: seams/eyes ignite over 1 beat | Crowd bobs on kicks; halo flare |
| Breakdown | Slow, close, shallow depth of field | Warm amber, 2 soft beams | Emissive follows the melody/vocal | Disintegration: hero/shards drift apart |
| Build 2 | Push-in | As build, faster | Gather | Blackout half bar |
| Drop 2 | As drop, faster orbit | Accent switches (e.g. red halo) | **Re-form on beat 1** | Bigger than drop 1 |
| Outro | Pull back into haze | Everything fades | Fades | Loop or hold |

## 6. Reaction budget

One musical element → one visual channel, at a time:

| Element | Channel |
|---|---|
| Kick | Key light intensity pulse |
| Snare | Strobe (rate-limited) |
| Hats | Lasers and particle sparkle |
| Melody / vocal | Hero emissive (seams, eyes) |
| Bass | Haze density, camera weight |

The intensity level (0–5) per section decides how many channels are live. Everything else holds still. The demo's
"Reactive (today)" toggle shows what breaking this rule looks like.

## 7. Safety (non-negotiable)

- Never more than **3 flashes in any 1 s window** (target 2); enforced by one limiter stage at the output.
- A global "strobe off" switch; default off when the OS reduced-motion setting is enabled.
- Warn users that the show contains flashing lights.
- Room lights (Phase 5) go through the same limiter.

## 8. Amateur tells to avoid (checklist for reviews)

- [ ] Gray veil over the image (too much bloom or haze)
- [ ] Banding in dark gradients (missing dither)
- [ ] Rainbow hue cycling
- [ ] Everything pulsing on every beat
- [ ] Constant orbit camera with no cuts
- [ ] Aliasing/shimmer on thin lines (lasers, rings)
- [ ] Stutter (uneven frame pacing), worse than a lower frame rate
- [ ] Chrome that reads as gray plastic (no light cards)
- [ ] Visual hits late relative to the audio (missing latency compensation)
