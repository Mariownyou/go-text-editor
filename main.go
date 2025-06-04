package main

import (
	"strings"
	"unicode/utf8"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	winWidth  = 800
	winHeight = 600
	fontSize  = 24
)

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow("Text Editor UTF-8", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		winWidth, winHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	font, err := ttf.OpenFont("FiraCode-Regular.ttf", fontSize)
	if err != nil {
		panic(err)
	}
	defer font.Close()

	var textBuffer string
	var cursorPos int // Rune offset, not byte

	sdl.StartTextInput()
	defer sdl.StopTextInput()

	for running := true; running; {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.TextInputEvent:
				input := e.GetText()
				textBuffer = insertAtRune(textBuffer, cursorPos, input)
				cursorPos += utf8.RuneCountInString(input)
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						if cursorPos > 0 {
							textBuffer = deleteAtRune(textBuffer, cursorPos-1)
							cursorPos--
						}
					case sdl.K_LEFT:
						if cursorPos > 0 {
							cursorPos--
						}
					case sdl.K_RIGHT:
						if cursorPos < utf8.RuneCountInString(textBuffer) {
							cursorPos++
						}
					case sdl.K_RETURN:
						textBuffer = insertAtRune(textBuffer, cursorPos, "\n")
						cursorPos++
					}
				}
			}
		}

		// Clear screen
		surface.FillRect(nil, 0)

		lines := strings.Split(textBuffer, "\n")
		y := int32(10)
		runeCounter := 0
		var cursorX, cursorY int32

		for _, line := range lines {
			lineRunes := utf8.RuneCountInString(line)

			// Render line
			if len(line) > 0 {
				textSurface, err := font.RenderUTF8Blended(line, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				if err == nil {
					defer textSurface.Free()
					dst := sdl.Rect{X: 10, Y: y, W: textSurface.W, H: textSurface.H}
					textSurface.Blit(nil, surface, &dst)
				}
			}

			// Determine cursor position on this line
			if cursorPos >= runeCounter && cursorPos <= runeCounter+lineRunes {
				substring := getFirstNRunes(line, cursorPos-runeCounter)
				w, _, _ := font.SizeUTF8(substring)
				cursorX = 10 + int32(w)
				cursorY = y
			}

			runeCounter += lineRunes + 1 // +1 for newline
			y += fontSize
		}

		// Draw cursor (white bar)
		surface.FillRect(&sdl.Rect{X: cursorX, Y: cursorY, W: 2, H: int32(fontSize)}, sdl.MapRGB(surface.Format, 255, 255, 255))

		window.UpdateSurface()
		sdl.Delay(16)
	}
}

// Utility: Insert UTF-8 string at rune index
func insertAtRune(s string, index int, insert string) string {
	runes := []rune(s)
	return string(runes[:index]) + insert + string(runes[index:])
}

// Utility: Delete rune at index
func deleteAtRune(s string, index int) string {
	runes := []rune(s)
	if index >= len(runes) {
		return s
	}
	return string(runes[:index]) + string(runes[index+1:])
}

// Utility: Get first N runes from string
func getFirstNRunes(s string, n int) string {
	runes := []rune(s)
	if n > len(runes) {
		n = len(runes)
	}
	return string(runes[:n])
}
