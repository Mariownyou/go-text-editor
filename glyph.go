package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type GlyphAtlas struct {
	Textures map[string]*sdl.Texture
	Font     *ttf.Font
	Size     int
}

func NewGlyphAtlas(renderer *sdl.Renderer, fontPath string, size int) *GlyphAtlas {
	font, err := ttf.OpenFont(fontPath, size)
	if err != nil {
		panic(err)
	}

	atlas := &GlyphAtlas{
		Textures: make(map[string]*sdl.Texture),
		Font:     font,
		Size:     size,
	}

	return atlas
}

func (a *GlyphAtlas) GetTexture(s string, renderer *sdl.Renderer) *sdl.Texture {
	if s == "\t" {
		s = "    " // Replace tab with spaces
	}

	if tex, ok := a.Textures[s]; ok {
		return tex
	}

	surf, err := a.Font.RenderUTF8Blended(s, sdl.Color{0, 0, 0, 255})
	if err != nil {
		return nil
	}
	defer surf.Free()

	tx, err := renderer.CreateTextureFromSurface(surf)
	if err != nil {
		return nil
	}

	a.Textures[s] = tx
	return tx
}

func (a *GlyphAtlas) Destroy() {
	for _, tex := range a.Textures {
		tex.Destroy()
	}
	a.Font.Close()
}
