package gfx

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"visualizer/audio"
)

type ImageFetcher struct {
	client *http.Client
}

func NewImageFetcher() *ImageFetcher {
	return &ImageFetcher{
		client: &http.Client{Timeout: 6 * time.Second},
	}
}

func (f *ImageFetcher) FetchTrackArtwork(meta audio.TrackMetadata, onDownloaded func(path string)) {
	go func() {
		if meta.FullTrack == "" {
			return
		}

		safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(meta.FullTrack, "_")
		filePath := filepath.Join("assets", fmt.Sprintf("track_%s.jpg", safeName))

		// Check if already generated / downloaded
		if _, err := os.Stat(filePath); err == nil {
			if onDownloaded != nil {
				onDownloaded(filePath)
			}
			return
		}

		// Try downloading online thematic image from Picsum
		imgURL := fmt.Sprintf("https://picsum.photos/seed/%d/1280/720", meta.Seed)
		resp, err := f.client.Get(imgURL)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			out, err := os.Create(filePath)
			if err == nil {
				_, err = io.Copy(out, resp.Body)
				out.Close()
				if err == nil {
					if onDownloaded != nil {
						onDownloaded(filePath)
					}
					return
				}
			}
		}

		// Fallback: Generate custom procedural artwork matching track essence & hue
		proceduralImg := generateProceduralArtwork(1280, 720, meta)
		out, err := os.Create(filePath)
		if err == nil {
			defer out.Close()
			jpeg.Encode(out, proceduralImg, &jpeg.Options{Quality: 92})
			if onDownloaded != nil {
				onDownloaded(filePath)
			}
		}
	}()
}

// Generate procedural 2D artwork matching track essence & hue
func generateProceduralArtwork(w, h int, meta audio.TrackMetadata) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	hue := float64(meta.HueOffset)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx := float64(x)/float64(w) - 0.5
			ny := float64(y)/float64(h) - 0.5

			dist := math.Hypot(nx, ny)
			angle := math.Atan2(ny, nx)

			val1 := math.Sin(angle*7.0 + dist*16.0 + float64(meta.Seed%13))
			val2 := math.Cos(dist*22.0 - angle*5.0)

			r := uint8(math.Min(255, math.Max(0, (val1+1.0)*130*math.Abs(math.Sin(hue*6.28)))))
			g := uint8(math.Min(255, math.Max(0, (val2+1.0)*110*math.Abs(math.Cos(hue*6.28)))))
			b := uint8(math.Min(255, math.Max(0, (1.0-dist)*240)))

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}
