package gfx

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type ShaderWatcher struct {
	path          string
	lastModTime   time.Time
	mu            sync.Mutex
	currentShader *ebiten.Shader
	pendingShader *ebiten.Shader
	hasPending    bool
	stopChan      chan struct{}
	OnReload      func()
}

func NewShaderWatcher(shaderPath string, fallbackSrc []byte) (*ShaderWatcher, error) {
	sw := &ShaderWatcher{
		path:     shaderPath,
		stopChan: make(chan struct{}),
	}

	var src []byte
	var modTime time.Time

	info, err := os.Stat(shaderPath)
	if err == nil {
		modTime = info.ModTime()
		data, readErr := os.ReadFile(shaderPath)
		if readErr == nil {
			src = data
		}
	}

	if len(src) == 0 {
		src = fallbackSrc
	}

	shader, err := ebiten.NewShader(src)
	if err != nil {
		return nil, fmt.Errorf("failed initial shader compilation: %w", err)
	}

	sw.currentShader = shader
	sw.lastModTime = modTime
	return sw, nil
}

func (sw *ShaderWatcher) Start() {
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-sw.stopChan:
				return
			case <-ticker.C:
				sw.checkAndReload()
			}
		}
	}()
}

func (sw *ShaderWatcher) checkAndReload() {
	info, err := os.Stat(sw.path)
	if err != nil {
		return
	}

	if !info.ModTime().After(sw.lastModTime) {
		return
	}

	data, err := os.ReadFile(sw.path)
	if err != nil {
		return
	}

	sw.lastModTime = info.ModTime()

	newShader, compileErr := ebiten.NewShader(data)
	if compileErr != nil {
		log.Printf("⚠️  [Shader Hot-Reload Error] %v", compileErr)
		return
	}

	log.Printf("🔥 [Shader Hot-Reload] Successfully reloaded %s at %s", sw.path, time.Now().Format("15:04:05"))

	sw.mu.Lock()
	sw.pendingShader = newShader
	sw.hasPending = true
	sw.mu.Unlock()

	if sw.OnReload != nil {
		sw.OnReload()
	}
}

// GetShader returns the active shader and applies any newly compiled hot-reloaded shader thread-safely
func (sw *ShaderWatcher) GetShader() *ebiten.Shader {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.hasPending {
		sw.currentShader = sw.pendingShader
		sw.pendingShader = nil
		sw.hasPending = false
	}
	return sw.currentShader
}

func (sw *ShaderWatcher) Stop() {
	close(sw.stopChan)
}
