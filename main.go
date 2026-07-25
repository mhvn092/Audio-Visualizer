package main

import (
	_ "embed"
	"fmt"
	"log"
	"sync"
	"time"

	"visualizer/audio"
	"visualizer/gfx"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed shaders/visualizer.kage
var visualizerShaderSrc []byte

type Game struct {
	shader       *ebiten.Shader
	startTime    time.Time
	audioCap     *audio.Capturer
	metaReader   *audio.MetadataReader
	imgFetcher   *gfx.ImageFetcher
	texLib       *gfx.TextureLibrary
	tickCount    int
	beatCooldown int
	currentMeta  audio.TrackMetadata
	paused       bool

	// Thread-safe queue for newly downloaded/generated artwork paths
	pendingImgMu    sync.Mutex
	pendingImgPaths []string

	// Smoothed audio values for organic shader movement
	smoothBass   float32
	smoothMid    float32
	smoothTreble float32
}

func NewGame() (*Game, error) {
	s, err := ebiten.NewShader(visualizerShaderSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to compile shader: %w", err)
	}

	cap := audio.NewCapturer()
	if err := cap.Start(); err != nil {
		log.Printf("Warning: audio capture failed: %v", err)
	}

	lib, err := gfx.NewTextureLibrary()
	if err != nil {
		return nil, fmt.Errorf("failed to load textures: %w", err)
	}

	fetcher := gfx.NewImageFetcher()

	game := &Game{
		shader:          s,
		startTime:       time.Now(),
		audioCap:        cap,
		texLib:          lib,
		imgFetcher:      fetcher,
		pendingImgPaths: make([]string, 0),
		currentMeta:     audio.TrackMetadata{Speed: 1.0, HueOffset: 0.6, GeoMode: 0},
	}

	metaReader := audio.NewMetadataReader(func(meta audio.TrackMetadata) {
		game.currentMeta = meta
		if meta.FullTrack != "" {
			ebiten.SetWindowTitle(fmt.Sprintf("▶ %s", meta.FullTrack))

			// Trigger automatic background artwork fetching based on track metadata
			game.imgFetcher.FetchTrackArtwork(meta, func(imgPath string) {
				game.pendingImgMu.Lock()
				game.pendingImgPaths = append(game.pendingImgPaths, imgPath)
				game.pendingImgMu.Unlock()
			})
		}
	})
	metaReader.Start()
	game.metaReader = metaReader

	return game, nil
}

// lerp smoothly interpolates between current and target
func lerp(current, target, speed float32) float32 {
	return current + (target-current)*speed
}

func (g *Game) Update() error {
	// Process any newly downloaded artwork on the main Ebitengine thread
	g.pendingImgMu.Lock()
	if len(g.pendingImgPaths) > 0 {
		pathsToProcess := g.pendingImgPaths
		g.pendingImgPaths = make([]string, 0)
		g.pendingImgMu.Unlock()

		for _, path := range pathsToProcess {
			if eimg, err := gfx.LoadSingleImage(path); err == nil {
				g.texLib.AddTextureAndActivate(eimg, path)
				log.Printf("Embedded artwork for track: %s", path)
			}
		}
	} else {
		g.pendingImgMu.Unlock()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.texLib.SwitchOnBeat()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if len(g.texLib.Textures) > 0 {
			g.texLib.Current = (g.texLib.Current - 1 + len(g.texLib.Textures)) % len(g.texLib.Textures)
			g.texLib.Next = (g.texLib.Current + 1) % len(g.texLib.Textures)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	if g.paused {
		return nil
	}

	g.tickCount++

	if g.audioCap != nil {
		feat := g.audioCap.GetFeatures()

		// Smooth interpolation — this is what makes the visuals feel organic
		// instead of jittery. Attack fast (0.15), decay slow (0.04)
		if feat.Bass > g.smoothBass {
			g.smoothBass = lerp(g.smoothBass, feat.Bass, 0.15)
		} else {
			g.smoothBass = lerp(g.smoothBass, feat.Bass, 0.04)
		}
		if feat.Mid > g.smoothMid {
			g.smoothMid = lerp(g.smoothMid, feat.Mid, 0.12)
		} else {
			g.smoothMid = lerp(g.smoothMid, feat.Mid, 0.03)
		}
		if feat.Treble > g.smoothTreble {
			g.smoothTreble = lerp(g.smoothTreble, feat.Treble, 0.12)
		} else {
			g.smoothTreble = lerp(g.smoothTreble, feat.Treble, 0.03)
		}

		if g.beatCooldown > 0 {
			g.beatCooldown--
		} else if feat.Onset > 0.78 {
			g.texLib.SwitchOnBeat()
			g.beatCooldown = 50
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	elapsed := float32(time.Since(g.startTime).Seconds())
	speed := g.currentMeta.Speed
	if speed < 0.1 {
		speed = 1.0
	}
	t := elapsed * speed

	var feat audio.AudioFeatures
	if g.audioCap != nil {
		feat = g.audioCap.GetFeatures()
	}

	currentPhoto := g.texLib.GetCurrentResized(w, h)

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = currentPhoto
	op.Images[1] = currentPhoto
	op.Uniforms = map[string]any{
		"Time":           t,
		"ScreenSize":     []float32{float32(w), float32(h)},
		"Bass":           feat.Bass,
		"Mid":            feat.Mid,
		"Treble":         feat.Treble,
		"Onset":          feat.Onset,
		"HueOffset":      g.currentMeta.HueOffset,
		"GeoMode":        float32(g.currentMeta.GeoMode),
		"SmoothedBass":   g.smoothBass,
		"SmoothedMid":    g.smoothMid,
		"SmoothedTreble": g.smoothTreble,
	}

	screen.DrawRectShader(w, h, g.shader, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Audio-Visualizer (3D Audio-Reactive GPU Engine)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}
	defer game.audioCap.Stop()
	defer game.metaReader.Stop()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
