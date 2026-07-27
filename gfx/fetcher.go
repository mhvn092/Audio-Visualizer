package gfx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"visualizer/audio"
)

type ImageFetcher struct {
	client *http.Client
}

type iTunesSearchResult struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		CollectionName string `json:"collectionName"`
		ArtistName     string `json:"artistName"`
		TrackName      string `json:"trackName"`
		ArtworkURL100  string `json:"artworkUrl100"`
	} `json:"results"`
}

func NewImageFetcher() *ImageFetcher {
	return &ImageFetcher{
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

// sanitizeFilename turns a string into a clean filesystem safe name
func sanitizeFilename(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	cleaned := reg.ReplaceAllString(s, "_")
	// Trim duplicate underscores
	regUnderscores := regexp.MustCompile(`_+`)
	return strings.Trim(regUnderscores.ReplaceAllString(cleaned, "_"), "_")
}

func (f *ImageFetcher) FetchTrackArtwork(meta audio.TrackMetadata, onDownloaded func(path string)) {
	go func() {
		if meta.FullTrack == "" && meta.Title == "" {
			return
		}

		safeTrack := sanitizeFilename(meta.FullTrack)
		if safeTrack == "" {
			safeTrack = sanitizeFilename(meta.Title)
		}

		// 1. Check if an exact track file already exists in assets/
		trackFile := filepath.Join("assets", fmt.Sprintf("album_%s.jpg", safeTrack))
		if _, err := os.Stat(trackFile); err == nil {
			log.Printf("📂 Using existing cached track artwork: %s", trackFile)
			if onDownloaded != nil {
				onDownloaded(trackFile)
			}
			return
		}

		// 2. Scan assets/ folder for any existing artwork matching track/artist/album keywords
		if existingPath := findExistingAsset(meta); existingPath != "" {
			log.Printf("📂 Found existing album artwork in assets: %s", existingPath)
			if onDownloaded != nil {
				onDownloaded(existingPath)
			}
			return
		}

		// 3. Search iTunes for official album cover art
		queries := []string{}
		if meta.Artist != "" && meta.Artist != "Unknown Artist" && meta.Artist != "Media Player" && meta.Artist != "WMP Legacy" {
			queries = append(queries, meta.Artist+" "+meta.Title)
			queries = append(queries, meta.Artist)
		}
		if meta.Title != "" {
			queries = append(queries, meta.Title)
		}
		if meta.FullTrack != "" {
			queries = append(queries, meta.FullTrack)
		}

		for _, q := range queries {
			albumName, coverURL := f.searchITunes(q, "album")
			if coverURL == "" {
				albumName, coverURL = f.searchITunes(q, "song")
			}

			if coverURL != "" {
				// Deduplicate by Album Name + Artist Name
				albumKey := sanitizeFilename(meta.Artist + "_" + albumName)
				if albumKey == "" {
					albumKey = safeTrack
				}
				albumFilePath := filepath.Join("assets", fmt.Sprintf("album_%s.jpg", albumKey))

				// Check if this album cover has ALREADY been downloaded previously
				if _, err := os.Stat(albumFilePath); err == nil {
					log.Printf("⚡ Reusing previously downloaded album cover ['%s']: %s", albumName, albumFilePath)
					if onDownloaded != nil {
						onDownloaded(albumFilePath)
					}
					return
				}

				// Download HD 1000x1000 Album Cover
				if err := f.downloadImage(coverURL, albumFilePath); err == nil {
					log.Printf("💿 Successfully downloaded official Album Cover ['%s'] for '%s': %s", albumName, meta.FullTrack, albumFilePath)
					if onDownloaded != nil {
						onDownloaded(albumFilePath)
					}
					return
				}
			}
		}

		// 4. Fallback: Generate custom procedural 2D artwork matching track seed & hue (NO stock wallpapers)
		fallbackFile := filepath.Join("assets", fmt.Sprintf("album_%s.jpg", safeTrack))
		proceduralImg := generateProceduralArtwork(1280, 720, meta)
		out, err := os.Create(fallbackFile)
		if err == nil {
			defer out.Close()
			jpeg.Encode(out, proceduralImg, &jpeg.Options{Quality: 92})
			log.Printf("🎨 Generated procedural album artwork for track: %s", fallbackFile)
			if onDownloaded != nil {
				onDownloaded(fallbackFile)
			}
		}
	}()
}

// findExistingAsset checks if an asset already exists in assets/ matching track or artist keywords
func findExistingAsset(meta audio.TrackMetadata) string {
	entries, err := os.ReadDir("assets")
	if err != nil {
		return ""
	}

	searchKeyTrack := strings.ToLower(sanitizeFilename(meta.Title))
	searchKeyArtist := strings.ToLower(sanitizeFilename(meta.Artist))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		nameLower := strings.ToLower(entry.Name())
		ext := filepath.Ext(nameLower)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		// If filename contains track title or artist name, match it
		if searchKeyTrack != "" && strings.Contains(nameLower, searchKeyTrack) {
			return filepath.Join("assets", entry.Name())
		}
		if searchKeyArtist != "" && searchKeyArtist != "unknown_artist" && searchKeyArtist != "wmp_legacy" && strings.Contains(nameLower, searchKeyArtist) {
			return filepath.Join("assets", entry.Name())
		}
	}
	return ""
}

func (f *ImageFetcher) searchITunes(query string, entity string) (string, string) {
	searchURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=%s&limit=1", url.QueryEscape(query), entity)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := f.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", ""
	}
	defer resp.Body.Close()

	var searchRes iTunesSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err == nil && searchRes.ResultCount > 0 {
		artURL := searchRes.Results[0].ArtworkURL100
		albumName := searchRes.Results[0].CollectionName
		if albumName == "" {
			albumName = searchRes.Results[0].TrackName
		}
		if artURL != "" {
			// Upgrade to 1000x1000 HD resolution
			artURL = strings.Replace(artURL, "100x100bb", "1000x1000bb", 1)
			return albumName, artURL
		}
	}
	return "", ""
}

func (f *ImageFetcher) downloadImage(imageURL string, destPath string) error {
	if imageURL == "" || destPath == "" {
		return fmt.Errorf("empty params")
	}

	httpReq, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func errIfEmpty(url, dest string) error {
	if url == "" || dest == "" {
		return fmt.Errorf("empty params")
	}
	return nil
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
