package main

import (
	_ "embed"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"visualizer/audio"
	"visualizer/gfx"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed shaders/visualizer.kage
var visualizerShaderSrc []byte

//go:embed shaders/postprocess.kage
var postprocessShaderSrc []byte

type Game struct {
	shaderWatcher *gfx.ShaderWatcher
	postPipeline  *gfx.PostPipeline
	audioCap      *audio.Capturer
	bpmEngine     *audio.BPMEngine
	metaReader    *audio.MetadataReader
	imgFetcher    *gfx.ImageFetcher
	texLib        *gfx.TextureLibrary
	startTime     time.Time
	tickCount     int
	beatCooldown  int
	currentMeta   audio.TrackMetadata
	paused        bool

	pendingImgMu    sync.Mutex
	pendingImgPaths []string

	reloadNoticeUntil time.Time

	// Smoothed audio values
	smoothBass   float32
	smoothMid    float32
	smoothTreble float32

	// Dynamic act selection driven by audio energy profile
	// 0.0 = Monolith (bass-heavy), 1.0 = Organism (mid/melodic), 2.0 = Constellation (treble/busy)
	actBlend float32

	// Running energy accumulators for act selection (longer time window)
	energyBassAvg   float32
	energyMidAvg    float32
	energyTrebleAvg float32

	// Cumulative audio energy — evolves continuously so shapes at min 5 ≠ min 30
	evolution float32
}

func NewGame() (*Game, error) {
	watcher, err := gfx.NewShaderWatcher("shaders/visualizer.kage", visualizerShaderSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize shader watcher: %w", err)
	}

	postPipe, err := gfx.NewPostPipeline(postprocessShaderSrc)
	if err != nil {
		log.Printf("Warning: post pipeline init: %v", err)
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
	bpm := audio.NewBPMEngine()

	game := &Game{
		shaderWatcher:   watcher,
		postPipeline:    postPipe,
		audioCap:        cap,
		bpmEngine:       bpm,
		startTime:       time.Now(),
		texLib:          lib,
		imgFetcher:      fetcher,
		pendingImgPaths: make([]string, 0),
		currentMeta:     audio.TrackMetadata{Speed: 1.0, HueOffset: 0.6, GeoMode: 0},
		actBlend:        0.0,
	}

	watcher.OnReload = func() {
		game.reloadNoticeUntil = time.Now().Add(2500 * time.Millisecond)
	}
	watcher.Start()

	metaReader := audio.NewMetadataReader(func(meta audio.TrackMetadata) {
		game.currentMeta = meta
		game.updateWindowTitle()

		if meta.FullTrack != "" {
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

func (g *Game) updateWindowTitle() {
	bpmState := g.bpmEngine.GetState()
	actNames := []string{"Monolith", "Organism", "Constellation"}
	actIdx := int(g.actBlend) % 3

	titleStr := fmt.Sprintf("Audio-Visualizer (%.0f BPM — %s)", bpmState.BPM, actNames[actIdx])
	if g.currentMeta.FullTrack != "" {
		titleStr = fmt.Sprintf("▶ %s (%.0f BPM — %s)", g.currentMeta.FullTrack, bpmState.BPM, actNames[actIdx])
	}
	if time.Now().Before(g.reloadNoticeUntil) {
		titleStr += " [🔥 SHADER HOT-RELOADED]"
	}
	ebiten.SetWindowTitle(titleStr)
}

func lerp(current, target, speed float32) float32 {
	return current + (target-current)*speed
}

func (g *Game) Update() error {
	if g.tickCount%30 == 0 || !g.reloadNoticeUntil.IsZero() {
		g.updateWindowTitle()
	}

	// Process downloaded artwork on main thread
	g.pendingImgMu.Lock()
	if len(g.pendingImgPaths) > 0 {
		pathsToProcess := g.pendingImgPaths
		g.pendingImgPaths = make([]string, 0)
		g.pendingImgMu.Unlock()

		for _, path := range pathsToProcess {
			if eimg, feats, err := gfx.LoadSingleImageWithAnalysis(path); err == nil {
				g.texLib.AddTextureWithFeatures(eimg, feats, path)
				log.Printf("Embedded artwork for track: %s (Brightness: %.2f, Warmth: %.2f, Complexity: %.2f)", path, feats.Brightness, feats.Warmth, feats.Complexity)
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

		g.bpmEngine.ProcessSample(feat.Onset, feat.Bass)

		// Smooth audio values
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

		// ── Dynamic Act Selection based on audio energy profile ──
		// Running exponential average over ~3-5 seconds (slow decay)
		decay := float32(0.005) // Very slow — takes several seconds to shift acts
		g.energyBassAvg = g.energyBassAvg*(1.0-decay) + g.smoothBass*decay
		g.energyMidAvg = g.energyMidAvg*(1.0-decay) + g.smoothMid*decay
		g.energyTrebleAvg = g.energyTrebleAvg*(1.0-decay) + g.smoothTreble*decay

		// Determine dominant energy band
		total := g.energyBassAvg + g.energyMidAvg + g.energyTrebleAvg + 0.001
		bassRatio := g.energyBassAvg / total
		midRatio := g.energyMidAvg / total
		trebleRatio := g.energyTrebleAvg / total

		// Target act: weighted centroid
		// Bass-dominant → 0 (Monolith), Mid-dominant → 1 (Organism), Treble-dominant → 2 (Constellation)
		targetAct := float32(0.0)*bassRatio + float32(1.0)*midRatio + float32(2.0)*trebleRatio

		// Artwork complexity bias: simple covers nudge toward Monolith, complex covers nudge toward Constellation
		artFeats := g.texLib.GetCurrentFeatures()
		artBias := (artFeats.Complexity - 0.4) * 0.4
		targetAct = float32(math.Max(0, math.Min(2.0, float64(targetAct+artBias))))

		// Clamp and smooth transition to target
		g.actBlend = lerp(g.actBlend, targetAct, 0.008) // Very slow transition

		// Evolution: cumulative energy that grows continuously over the set
		g.evolution += (g.smoothBass + g.smoothMid + g.smoothTreble) * 0.0003

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

	bpmState := g.bpmEngine.GetState()
	currentPhoto := g.texLib.GetCurrentResized(w, h)
	artFeats := g.texLib.GetCurrentFeatures()

	uniforms := map[string]any{
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
		"BPM":            bpmState.BPM,
		"BeatPhase":      bpmState.BeatPhase,
		"BarCount":       float32(bpmState.BarCount),
		"ActBlend":       g.actBlend,
		"Evolution":      g.evolution,
		"ArtColor1":      []float32{artFeats.DominantColors[0][0], artFeats.DominantColors[0][1], artFeats.DominantColors[0][2]},
		"ArtColor2":      []float32{artFeats.DominantColors[1][0], artFeats.DominantColors[1][1], artFeats.DominantColors[1][2]},
		"ArtColor3":      []float32{artFeats.DominantColors[2][0], artFeats.DominantColors[2][1], artFeats.DominantColors[2][2]},
		"ArtBrightness":  artFeats.Brightness,
		"ArtWarmth":      artFeats.Warmth,
		"ArtComplexity":  artFeats.Complexity,
		"ArtContrast":    artFeats.Contrast,
	}

	g.postPipeline.Render(screen, func(target *ebiten.Image) {
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = currentPhoto
		op.Images[1] = currentPhoto
		op.Uniforms = uniforms

		currentShader := g.shaderWatcher.GetShader()
		target.DrawRectShader(w, h, currentShader, op)
	}, uniforms)
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
	defer game.shaderWatcher.Stop()
	defer game.postPipeline.Stop()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
