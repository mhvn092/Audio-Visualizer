package audio

import (
	"hash/fnv"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type TrackMetadata struct {
	Title     string
	Artist    string
	FullTrack string
	Seed      uint32
	HueOffset float32
	Speed     float32
	GeoMode   int
}

type MetadataReader struct {
	mu            sync.RWMutex
	currentTrack  TrackMetadata
	onTrackChange func(track TrackMetadata)
	stopChan      chan struct{}
}

func NewMetadataReader(onTrackChange func(track TrackMetadata)) *MetadataReader {
	return &MetadataReader{
		onTrackChange: onTrackChange,
		stopChan:      make(chan struct{}),
		currentTrack:  TrackMetadata{Speed: 1.0},
	}
}

func (mr *MetadataReader) GetCurrent() TrackMetadata {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return mr.currentTrack
}

func (mr *MetadataReader) Start() {
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		mr.checkTrack()

		for {
			select {
			case <-mr.stopChan:
				return
			case <-ticker.C:
				mr.checkTrack()
			}
		}
	}()
}

func (mr *MetadataReader) checkTrack() {
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", "scripts/get_media.ps1")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return
	}

	parts := strings.SplitN(raw, "|", 2)
	title := parts[0]
	artist := ""
	if len(parts) > 1 {
		artist = parts[1]
	}
	fullTrack := title + " - " + artist

	mr.mu.RLock()
	lastTrack := mr.currentTrack.FullTrack
	mr.mu.RUnlock()

	if fullTrack == lastTrack {
		return
	}

	h := fnv.New32a()
	h.Write([]byte(fullTrack))
	seed := h.Sum32()

	hue := float32(seed%360) / 360.0
	speed := 0.8 + float32((seed>>4)%40)/100.0
	geoMode := int((seed >> 8) % 4)

	meta := TrackMetadata{
		Title:     title,
		Artist:    artist,
		FullTrack: fullTrack,
		Seed:      seed,
		HueOffset: hue,
		Speed:     speed,
		GeoMode:   geoMode,
	}

	mr.mu.Lock()
	mr.currentTrack = meta
	mr.mu.Unlock()

	if mr.onTrackChange != nil {
		mr.onTrackChange(meta)
	}
}

func (mr *MetadataReader) Stop() {
	close(mr.stopChan)
}
