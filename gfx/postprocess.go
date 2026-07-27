package gfx

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type PostPipeline struct {
	offscreenBuffer *ebiten.Image
	shaderWatcher   *ShaderWatcher
	enabled         bool
}

func NewPostPipeline(fallbackSrc []byte) (*PostPipeline, error) {
	watcher, err := NewShaderWatcher("shaders/postprocess.kage", fallbackSrc)
	if err != nil {
		log.Printf("Warning: post-process shader watcher fallback: %v", err)
	}
	if watcher != nil {
		watcher.Start()
	}

	return &PostPipeline{
		shaderWatcher: watcher,
		enabled:       true,
	}, nil
}

func (pp *PostPipeline) Render(screen *ebiten.Image, renderScene func(target *ebiten.Image), uniforms map[string]any) {
	if !pp.enabled || pp.shaderWatcher == nil {
		renderScene(screen)
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Ensure offscreen buffer matches screen size
	if pp.offscreenBuffer == nil || pp.offscreenBuffer.Bounds().Dx() != w || pp.offscreenBuffer.Bounds().Dy() != h {
		pp.offscreenBuffer = ebiten.NewImage(w, h)
	}

	// Pass 1: Render 3D Scene + Lasers into offscreen buffer
	pp.offscreenBuffer.Clear()
	renderScene(pp.offscreenBuffer)

	// Pass 2: Apply Post-Processing Shader (Chromatic Aberration + Bloom) to final screen
	shader := pp.shaderWatcher.GetShader()
	if shader == nil {
		screen.DrawImage(pp.offscreenBuffer, nil)
		return
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = pp.offscreenBuffer
	op.Uniforms = map[string]any{
		"ScreenSize":       []float32{float32(w), float32(h)},
		"Bass":             uniforms["Bass"],
		"Mid":              uniforms["Mid"],
		"Treble":           uniforms["Treble"],
		"Onset":            uniforms["Onset"],
		"BloomIntensity":   float32(1.2),
		"AberrationAmount": float32(1.0),
	}

	screen.DrawRectShader(w, h, shader, op)
}

func (pp *PostPipeline) Stop() {
	if pp.shaderWatcher != nil {
		pp.shaderWatcher.Stop()
	}
}
