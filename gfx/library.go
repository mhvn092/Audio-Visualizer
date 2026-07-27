package gfx

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type TextureLibrary struct {
	mu          sync.Mutex
	Textures    []*ebiten.Image
	Features    []ArtFeatures
	Names       []string
	Current     int
	Next        int
	scaledCache map[string]*ebiten.Image
}

func NewTextureLibrary() (*TextureLibrary, error) {
	lib := &TextureLibrary{
		Textures:    make([]*ebiten.Image, 0),
		Features:    make([]ArtFeatures, 0),
		Names:       make([]string, 0),
		scaledCache: make(map[string]*ebiten.Image),
	}

	os.MkdirAll("assets", 0755)

	// Scan assets directory for existing user artwork (png, jpg, jpeg)
	entries, err := os.ReadDir("assets")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				fullPath := filepath.Join("assets", entry.Name())
				img, feats, err := loadEbitenImageAndAnalyze(fullPath)
				if err == nil {
					lib.Textures = append(lib.Textures, img)
					lib.Features = append(lib.Features, feats)
					lib.Names = append(lib.Names, entry.Name())
				}
			}
		}
	}

	// If no images are found in assets/, create an in-memory neutral placeholder texture
	if len(lib.Textures) == 0 {
		placeholder := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				placeholder.Set(x, y, color.RGBA{R: 20, G: 20, B: 30, A: 255})
			}
		}
		eImg := ebiten.NewImageFromImage(placeholder)
		lib.Textures = append(lib.Textures, eImg)
		lib.Features = append(lib.Features, DefaultArtFeatures())
		lib.Names = append(lib.Names, "default_placeholder")
	}

	lib.Current = 0
	if len(lib.Textures) > 1 {
		lib.Next = 1
	}

	return lib, nil
}

func LoadSingleImage(path string) (*ebiten.Image, error) {
	img, _, err := loadEbitenImageAndAnalyze(path)
	return img, err
}

func LoadSingleImageWithAnalysis(path string) (*ebiten.Image, ArtFeatures, error) {
	return loadEbitenImageAndAnalyze(path)
}

func (tl *TextureLibrary) AddTextureAndActivate(img *ebiten.Image, name string) {
	tl.AddTextureWithFeatures(img, DefaultArtFeatures(), name)
}

func (tl *TextureLibrary) AddTextureWithFeatures(img *ebiten.Image, feats ArtFeatures, name string) {
	if img == nil {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.Textures = append(tl.Textures, img)
	tl.Features = append(tl.Features, feats)
	tl.Names = append(tl.Names, filepath.Base(name))

	// Immediately activate the newly added texture
	newIdx := len(tl.Textures) - 1
	tl.Current = newIdx
	tl.Next = (newIdx + 1) % len(tl.Textures)

	// Clear cache to prevent stale scaled textures
	tl.scaledCache = make(map[string]*ebiten.Image)
}

func (tl *TextureLibrary) GetCurrentFeatures() ArtFeatures {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.Features) == 0 || tl.Current < 0 || tl.Current >= len(tl.Features) {
		return DefaultArtFeatures()
	}
	return tl.Features[tl.Current]
}

func (tl *TextureLibrary) SwitchOnBeat() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.Textures) <= 1 {
		return
	}
	tl.Current = (tl.Current + 1) % len(tl.Textures)
	tl.Next = (tl.Current + 1) % len(tl.Textures)
}

func (tl *TextureLibrary) GetResizedTexture(idx int, targetW, targetH int) *ebiten.Image {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.Textures) == 0 || idx < 0 || idx >= len(tl.Textures) {
		return nil
	}

	src := tl.Textures[idx]
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == targetW && sh == targetH {
		return src
	}

	key := fmt.Sprintf("%d_%d_%d", idx, targetW, targetH)
	if cached, ok := tl.scaledCache[key]; ok {
		return cached
	}

	dst := ebiten.NewImage(targetW, targetH)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(targetW)/float64(sw), float64(targetH)/float64(sh))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(src, op)

	tl.scaledCache[key] = dst
	return dst
}

func (tl *TextureLibrary) GetCurrentResized(targetW, targetH int) *ebiten.Image {
	return tl.GetResizedTexture(tl.Current, targetW, targetH)
}

func (tl *TextureLibrary) GetNextResized(targetW, targetH int) *ebiten.Image {
	return tl.GetResizedTexture(tl.Next, targetW, targetH)
}

func loadEbitenImageAndAnalyze(path string) (*ebiten.Image, ArtFeatures, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, DefaultArtFeatures(), err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, DefaultArtFeatures(), err
	}

	feats := AnalyzeArtwork(img)
	eImg := ebiten.NewImageFromImage(img)
	return eImg, feats, nil
}
