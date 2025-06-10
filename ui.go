package main

// #cgo CFLAGS: -x objective-c -fmodules
// #cgo LDFLAGS: -framework Cocoa
// void SetNSWindowTitleBarColor(void* nswindowPtr);
import "C"

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	uiHeight = 10
)

var (
	uiBackgroundColor   = []uint8{247, 247, 247, 1}
	tabsBackgroundColor = []uint8{222, 222, 222, 1}
)

func setColor(renderer *sdl.Renderer, color []uint8) {
	renderer.SetDrawColor(color[0], color[1], color[2], color[3])
}

func SetTitleBarColor(window *sdl.Window) {
	info, _ := window.GetWMInfo()
	wi := info.GetCocoaInfo()
	if wi != nil {
		C.SetNSWindowTitleBarColor(wi.Window)
	}
}

func DrawFPS(renderer *sdl.Renderer, atlas *GlyphAtlas, fps int) {
	fpsText := fmt.Sprintf("FPS: %d", fps)
	fpsTexture := atlas.GetTexture(fpsText, renderer)
	if fpsTexture != nil {
		_, _, w, h, _ := fpsTexture.Query()
		rw, _, _ := renderer.GetOutputSize()
		fpsX := rw - w - 10 // 10 pixels from the right edge
		// white background for FPS text
		// rect := sdl.Rect{X: fpsX - 5, Y: 0, W: w + 10, H: h + 10}
		// renderer.SetDrawColor(255, 255, 255, 200)
		// renderer.FillRect(&rect)
		renderer.Copy(fpsTexture, nil, &sdl.Rect{X: fpsX, Y: 10, W: w, H: h})
	}
}

func DrawTabs(renderer *sdl.Renderer, atlas *GlyphAtlas, tabs []string) {
	tabX := int32(10)
	tabY := int32(10)

	w := int32(0)
	h := int32(0)

	for _, tab := range tabs {
		tabTexture := atlas.GetTexture(tab, renderer)
		if tabTexture == nil {
			continue
		}

		_, _, w, h, _ = tabTexture.Query()

		rect := sdl.Rect{X: tabX, Y: tabY - 10, W: w + 10, H: h + 10}
		setColor(renderer, uiBackgroundColor)
		renderer.FillRect(&rect)
		renderer.Copy(tabTexture, nil, &sdl.Rect{X: tabX, Y: tabY, W: w, H: h})
		tabX += w + 10 // Move to the right for the next tab
	}

	rw, _, _ := renderer.GetOutputSize()
	setColor(renderer, tabsBackgroundColor)
	renderer.FillRect(&sdl.Rect{X: tabX - 10, Y: 0, W: rw, H: h + 10})
}
