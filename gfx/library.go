package gfx

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type TextureLibrary struct {
	mu          sync.Mutex
	Textures    []*ebiten.Image
	Names       []string
	Current     int
	Next        int
	scaledCache map[string]*ebiten.Image
}

func NewTextureLibrary() (*TextureLibrary, error) {
	lib := &TextureLibrary{
		Textures:    make([]*ebiten.Image, 0),
		Names:       make([]string, 0),
		scaledCache: make(map[string]*ebiten.Image),
	}

	os.MkdirAll("assets", 0755)

	// Scan assets directory for png, jpg, and jpeg files
	entries, err := os.ReadDir("assets")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				fullPath := filepath.Join("assets", entry.Name())
				img, err := loadEbitenImage(fullPath)
				if err == nil {
					lib.Textures = append(lib.Textures, img)
					lib.Names = append(lib.Names, entry.Name())
				}
			}
		}
	}

	// If no images found, generate default procedural art
	if len(lib.Textures) == 0 {
		gen1 := generateCyberGrid(960, 540)
		gen2 := generateCosmicNebula(960, 540)
		saveImage("assets/cyber_grid.png", gen1)
		saveImage("assets/cosmic_nebula.png", gen2)

		if img1, err := loadEbitenImage("assets/cyber_grid.png"); err == nil {
			lib.Textures = append(lib.Textures, img1)
			lib.Names = append(lib.Names, "cyber_grid.png")
		}
		if img2, err := loadEbitenImage("assets/cosmic_nebula.png"); err == nil {
			lib.Textures = append(lib.Textures, img2)
			lib.Names = append(lib.Names, "cosmic_nebula.png")
		}
	}

	if len(lib.Textures) == 0 {
		return nil, fmt.Errorf("no valid images could be loaded from assets/")
	}

	lib.Current = 0
	if len(lib.Textures) > 1 {
		lib.Next = 1
	}

	return lib, nil
}

func LoadSingleImage(path string) (*ebiten.Image, error) {
	return loadEbitenImage(path)
}

func (tl *TextureLibrary) AddTextureAndActivate(img *ebiten.Image, name string) {
	if img == nil {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.Textures = append(tl.Textures, img)
	tl.Names = append(tl.Names, filepath.Base(name))

	// Immediately activate the newly added texture
	newIdx := len(tl.Textures) - 1
	tl.Current = newIdx
	tl.Next = (newIdx + 1) % len(tl.Textures)

	// Clear cache to prevent stale scaled textures
	tl.scaledCache = make(map[string]*ebiten.Image)
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

func loadEbitenImage(path string) (*ebiten.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func saveImage(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}

func generateCyberGrid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx := float64(x) / float64(w)
			ny := float64(y) / float64(h)

			grid := math.Mod(nx*24.0, 1.0) < 0.07 || math.Mod(ny*18.0, 1.0) < 0.07
			dist := math.Hypot(nx-0.5, ny-0.5)

			r := uint8(math.Min(255, (1.0-dist)*180+50))
			g := uint8(0)
			b := uint8(math.Min(255, dist*220+80))

			if grid {
				r = 255
				g = 100
				b = 255
			}
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func generateCosmicNebula(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx := float64(x)/float64(w) - 0.5
			ny := float64(y)/float64(h) - 0.5

			angle := math.Atan2(ny, nx)
			radius := math.Hypot(nx, ny)

			val := math.Sin(angle*5.0 + radius*12.0)
			r := uint8(math.Min(255, (val+1.0)*100+20))
			g := uint8(math.Min(255, (math.Cos(radius*15.0)+1.0)*120))
			b := uint8(math.Min(255, (1.0-radius)*255))

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}
