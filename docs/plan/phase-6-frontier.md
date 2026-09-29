# Phase 6: Frontier (optional, ongoing)

Each experiment ships behind a switch with a non-ML fallback. None of this may run on the render GPU during normal use.

## Experiments

1. **Real-time diffusion "dream" pass.** Feed the engine's depth/normal render to a streaming diffusion model with a
   depth ControlNet and let it repaint breakdown sequences. Reference numbers: StreamDiffusion reaches about 55–60 fps
   at 512² on an RTX 4090 (SD-turbo, one step); SANA-Streaming does 1280×704 at 24 fps on one RTX 5090;
   StreamDiffusionV2 targets video with temporal consistency. All of them saturate a GPU, so run on a second GPU, a
   second PC over NDI, or a cloud service. Blend its output with the engine frame; use it only in breakdowns.
2. **ML stems.** Real-time source separation (HS-TasNet class, ~3 ms delay; StemgenRT, ~6 ms) to get drums/bass/
   vocals/other envelopes, so the reaction budget can bind to true stems. Run on NPU/CPU via Windows ML.
3. **ML beat and downbeat tracking.** Causal trackers (BeatNet, BEAST: < 50 ms latency) as an alternative to the
   Phase 0 tracker; keep whichever scores better on the test set.
4. **Gaussian-splat worlds** in the renderer (GPU radix sort + instanced quads; 1–2 M splats in ~2 ms), audio-reactive
   splat displacement and dissolve.

## Acceptance

Each experiment: a switch, a fallback, a measured cost, and an A/B comparison against the non-ML path recorded in
Results.

## Results

(Fill in.)
