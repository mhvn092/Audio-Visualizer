# 🔮 Audio-Visualizer

A modern, high-performance, real-time 3D audio visualizer built in pure **Go** using **Ebitengine** (`ebiten/v2`) and GPU fragment shaders written in **Kage**.

> ⚡ **AI-Assisted Coding**: Built through AI-assisted coding

This application captures live Windows audio loopback, analyzes frequency bands in real-time, extracts live track metadata (ID3 tags / WinRT GSMTC / WMP Legacy), and renders organic morphing 3D raymarched Signed Distance Fields (SDFs) merged with audio-warped visual backdrops.

---

## ✨ Features

- 🎧 **Zero-Latency WASAPI Loopback Capture**: Captures native system audio directly from Windows Audio Session API with zero CGO overhead.
- 📊 **Real-time 3-Band Frequency Analysis**: Split audio stream into Bass, Mid, and Treble with onset energy beat detection and exponential smoothing (fast attack, slow decay).
- 🎵 **Multi-Source Metadata Reader**:
  - Windows WinRT System Media Transport Controls (Spotify, YouTube on Chrome/Edge, modern players).
  - Windows Media Player Legacy XML (`lastplayed.wpl`) with Windows Shell COM ID3 tag extraction (Title, Artist).
  - Native Window Title fallback for legacy media players.
- 🎨 **Dynamic Artwork Embedding**:
  - Automatically fetches or generates track-tailored 2D artwork upon track changes.
  - Dynamically embeds artwork as an audio-reactive visual backdrop and refracts it onto 3D metallic surfaces.
- 🔥 **Live Shader Hot-Reloading**:
  - Edit `.kage` shaders on disk while the app is running and watch visuals update live without restarting!
  - Graceful error handling keeps the previous valid shader active if a syntax error occurs while typing.
- 🌀 **High-Art 3D GPU Raymarching**:
  - Real-time 3D Signed Distance Fields (SDFs) with organic `smin()` shape morphing (Torus, Monolith, Octahedron, Orbiting Satellites).
  - Volumetric aura glow, two-point lighting, fresnel rim lighting, and subtle film grain / CRT scanline aesthetics.

---

## 🛠️ Requirements

- **Go**: Version 1.25+ installed on Windows.
- **Graphics Hardware**: Any GPU supporting DirectX 11/12 or Vulkan (integrated Intel/AMD graphics or dedicated NVIDIA/AMD GPUs).
- **Zero C++ Dependencies**: Compiles natively in pure Go without requiring Visual Studio C++ compilers or MSVC SDKs.

---

## 🚀 Quick Start

1. **Clone the repository**:
   ```powershell
   git clone https://github.com/mhvn092/Audio-Visualizer.git
   cd Audio-Visualizer
   ```

2. **Run the application**:
   ```powershell
   go run .
   ```

3. **Play Music**: Start playing music on Spotify, YouTube, Windows Media Player, or VLC. The visualizer will detect the audio and track metadata automatically!

---

## ⌨️ Controls

| Key | Action |
| :--- | :--- |
| **`F11`** | Toggle Fullscreen Mode |
| **`Space`** / **`Right Arrow`** | Trigger Manual Beat Artwork Switch |
| **`Left Arrow`** | Switch to Previous Artwork |
| **`P`** | Pause / Resume Visuals |

---

## 🗺️ Roadmap to "Crazy" Concert-Level Visualizer

Here is the master plan for evolving this project into an industry-grade, mind-blowing visual engine:

### 🌌 Phase 1: Volumetric Light Fields & Particle Systems
- [x] **Volumetric Laser Rays**: Add raymarched atmospheric light cones and scanning laser grids driven by treble frequencies.
- [x] **Audio-Reactive GPU Particle Field**: 3D floating embers and sparkling dust motes orbiting the 3D geometry.
- [x] **Post-Processing Pipeline**: Bloom glow, Chromatic Aberration lens distortion, and contrast vignette passes.

### 🧠 Phase 2: Live AI Stem Separation & Neural Shaders
- [ ] **Real-time Stem Separation**: Integrate ONNX Runtime to split audio stream into 4 stems (*Vocals, Drums, Bass, Instruments*).
- [ ] **Multi-Layer Reactivity**: Map Vocals to inner core geometry, Drums to camera shake / bass pulse, and Instruments to background warp fields.

### ⚡ Phase 3: Developer Experience & Live Performance Tools
- [x] **Live Kage Shader Hot-Reloading**: Automatically reload `.kage` fragment shaders when edited on disk without restarting the application.
- [x] **BPM & Tap-Tempo Sync**: Automatic BPM estimation for quantized camera transitions and light flashes.
- [ ] **MIDI & DMX Lighting Output**: Export DMX lighting control signals (Art-Net / SACN) to drive real stage lights in sync with the visualizer.

### 🌐 Phase 4: WebGL & Cross-Platform Deployment
- [ ] **Wasm / WebGL2 Build Target**: Compile to WebAssembly to allow running directly in modern web browsers.
- [ ] **Multi-Monitor Projection Mode**: Dual-window output for VJing (Control Panel on Monitor 1, Fullscreen Visuals on Monitor 2).

---

## 📜 License

Distributed under the MIT License. Feel free to modify and build upon it!
