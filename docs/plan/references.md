# References

## About Anyma's production

- Y.M.Cinema, "The Making of Anyma's The End of Genesys at the Las Vegas Sphere": https://ymcinema.com/2025/01/30/the-making-of-anymas-the-end-of-genesys-at-the-las-vegas-sphere-a-groundbreaking-fusion-of-music-and-technology/ (C4D/Houdini, custom pipelines, cloud render farms, 16K, 100+ people)
- "How Anyma's live shows use Unreal Engine": https://virtualproduction.services/how-anymas-live-shows-use-unreal-engine-to-create-a-cybernetic-opera/ (UE5, timecode via disguise)
- Pollstar, "Anyma at Sphere": https://news.pollstar.com/2025/01/01/anyma-at-sphere-the-future-is-here/
- Rolling Stone, "Anyma will become first electronic act to headline the Sphere": https://www.rollingstone.com/music/music-news/anyma-will-become-first-electronic-act-headline-sphere-1235060677/
- Collater.al on Alessio De Vecchi (visual co-creative director): https://www.collater.al/en/anyma-afterlife-genesys-sphere-las-vegas-new-year-alessio-de-vecchi-visual-art/

## Rendering (Phase 1–2)

- go-webgpu/webgpu, zero-cgo bindings to wgpu-native: https://github.com/go-webgpu/webgpu
- gogpu/wgpu, pure-Go WebGPU (fallback): https://github.com/gogpu/wgpu
- AMD FSR 2 (MIT), temporal upscaling: https://gpuopen.com/fidelityfx-superresolution-2/
- Bloom: Jimenez, "Next Generation Post Processing in Call of Duty: Advanced Warfare" (SIGGRAPH 2014)
- Area lights: Heitz et al., "Real-Time Polygonal-Light Shading with Linearly Transformed Cosines" (2016)
- Volumetrics: Wronski, "Volumetric Fog" (SIGGRAPH 2014); Hillaire, "Physically Based and Unified Volumetric Rendering in Frostbite" (2015)
- AgX tonemapping: the Blender 4.x and three.js implementations
- glTF loader for Go: https://github.com/qmuntal/gltf
- Gaussian splats in WebGPU: https://github.com/KeKsBoTer/web-splat, https://blog.playcanvas.com/new-in-supersplat-webgpu-and-streaming-bring-huge-performance-wins/
- Assets: Poly Haven (CC0 HDRIs/textures); MetaHuman licence change (usable outside Unreal since 2025): https://www.cgchannel.com/2025/06/you-can-now-sell-metahumans-or-use-them-in-unity-or-godot/

## Audio (Phase 0, 3, 6)

- winrt-go media control (GSMTC): https://pkg.go.dev/github.com/saltosystems/winrt-go/windows/media/control
- Process loopback capture (Windows build 20348+): https://learn.microsoft.com/en-us/windows/win32/api/audioclientactivationparams/ns-audioclientactivationparams-audioclient_process_loopback_params
- BeatNet (real-time beat/downbeat): https://github.com/mjhydri/BeatNet ; BEAST: https://arxiv.org/abs/2312.17156
- All-In-One music structure analyzer: https://github.com/mir-aidj/all-in-one
- HS-TasNet (real-time stems): https://arxiv.org/abs/2402.17701 ; StemgenRT: https://github.com/sweetspotsoundsystem/stemgen-rt
- Pure-Go SQLite: https://pkg.go.dev/modernc.org/sqlite

## ML runtime and generation (Phase 4, 6)

- Windows ML (system ONNX Runtime with NPU/GPU providers): https://blogs.windows.com/windowsdeveloper/2025/09/23/windows-ml-is-generally-available-empowering-developers-to-scale-local-ai-across-windows-devices/
- ONNX Runtime from Go without cgo: https://github.com/shota3506/onnxruntime-purego
- Image-to-3D (TRELLIS 2 MIT; Hunyuan3D has regional licence limits): https://www.3daistudio.com/blog/trellis-2-vs-hunyuan-3d-differences-explained
- World Labs Marble splat export: https://docs.worldlabs.ai/marble/export/gaussian-splat/index
- StreamDiffusion: https://github.com/cumulo-autumn/StreamDiffusion ; StreamDiffusionV2: https://arxiv.org/abs/2511.07399 ; SANA-Streaming: https://www.alphaxiv.org/overview/2605.30409
- Anthropic Go SDK (show writer): https://github.com/anthropics/anthropic-sdk-go

## Room output (Phase 5)

- Hue Entertainment API: https://iotech.blog/posts/philips-hue-entertainment-api/
- WLED DDP: https://kno.wled.ge/interfaces/ddp/
