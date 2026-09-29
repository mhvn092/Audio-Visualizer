# Phase 5: Beyond the screen (2 weeks)

## Goal

Make the show fill the room, and add performer tooling.

## Prerequisites

Phase 3 (director `Frame` with light cues and latency model).

## Tasks

### Task 1: Room lights (`internal/room`)

All outputs read the same director `Frame` as the renderer, stamped for display time, sent at 40–60 Hz.

- **WLED over DDP** (UDP port 4048; about 2 ms on local Wi-Fi). Map: accent color washes, strobe (same rate limiter),
  kick pulses on the key-light channel.
- **Philips Hue Entertainment API**: DTLS-PSK UDP stream to the bridge (bridge forwards to lights at ~25 Hz; stream at
  50–60 Hz). One-time pairing flow in settings. Hue is slower: use it for color washes, not strobes.
- **DMX via Art-Net or sACN (E1.31)**: fixture profiles in `shows/fixtures/*.yaml` (channel maps for pars, moving heads,
  strobes). Moving heads follow the virtual beam rig in the world.
- Per-output latency offsets in config (lights have their own delays).

### Task 2: Box mode (fourth-wall illusion)

- Off-axis (asymmetric frustum) projection: the screen becomes a window into a box behind it.
- Optional head tracking with a webcam: a face-landmark model (ONNX, e.g. a MediaPipe face landmarker export) gives the
  eye position; update the projection each frame with smoothing. Without a webcam, use a fixed "sweet spot".
- A "box" world: a room behind the screen plane whose hero reaches toward or through the frame on the drop.

### Task 3: Dual-screen and outputs

- Control window (on the laptop screen): transport, world/shot overrides, intensity fader, strobe kill, live HUD.
- Fullscreen output window on the second display.
- Spout (Windows) and/or NDI output for OBS and VJ software.
- MIDI/OSC input for manual cues (next world, blackout, flash, hold).

## Acceptance criteria

- [ ] Room lights match the screen within one frame after calibration (phone video check).
- [ ] Box mode holds 120 fps with head tracking on; tracking loss falls back smoothly.
- [ ] Control and output windows run on separate displays; Spout/NDI visible in OBS.

## Results

(Fill in.)
