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
	fontSize  = 20
)

var font *ttf.Font

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "2") // Best anti-aliasing for scaling

	window, err := sdl.CreateWindow("Text Editor UTF-8", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		winWidth, winHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	font, err = ttf.OpenFont("FiraCode-Regular.ttf", fontSize)
	if err != nil {
		panic(err)
	}
	defer font.Close()

	var textBuffer string
	var cursorPos int // rune-based position

	sdl.StartTextInput()
	defer sdl.StopTextInput()

	for running := true; running; {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN && e.Button == sdl.BUTTON_LEFT {
					cursorPos = getCharIndexOnClick(e.X, e.Y, textBuffer)
				}
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

		// Clear background
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()

		// Draw text
		lines := strings.Split(textBuffer, "\n")
		y := int32(10)
		runeCounter := 0
		var cursorX, cursorY int32

		for _, line := range lines {
			lineRunes := utf8.RuneCountInString(line)

			if len(line) > 0 {
				textSurface, err := font.RenderUTF8Blended(line, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				if err == nil {
					defer textSurface.Free()
					textTexture, err := renderer.CreateTextureFromSurface(textSurface)
					if err == nil {
						defer textTexture.Destroy()
						dst := sdl.Rect{X: 10, Y: y, W: textSurface.W, H: textSurface.H}
						renderer.Copy(textTexture, nil, &dst)
					}
				}
			}

			// Cursor position logic
			if cursorPos >= runeCounter && cursorPos <= runeCounter+lineRunes {
				substr := getFirstNRunes(line, cursorPos-runeCounter)
				w, _, _ := font.SizeUTF8(substr)
				cursorX = 10 + int32(w)
				cursorY = y
			}

			runeCounter += lineRunes + 1 // \n counts
			y += int32(fontSize)
		}

		// Draw cursor
		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.FillRect(&sdl.Rect{X: cursorX, Y: cursorY, W: 2, H: int32(fontSize)})

		renderer.Present()
		sdl.Delay(16)
	}
}

func getCharIndexOnClick(mouseX, mouseY int32, text string) int {
	runes := []rune(text)
	cursorX, cursorY := int32(10), int32(10)

	for i, r := range runes {
		w, _, _ := font.SizeUTF8(string(r))
		if mouseX >= cursorX && mouseX <= cursorX+int32(w) && mouseY >= cursorY && mouseY <= cursorY+int32(fontSize) {
			return i
		}
		cursorX += int32(w)
		if r == '\n' {
			cursorX = 10
			cursorY += int32(fontSize)
		}
	}
	return len(runes)
}

func insertAtRune(s string, index int, insert string) string {
	runes := []rune(s)
	return string(runes[:index]) + insert + string(runes[index:])
}

func deleteAtRune(s string, index int) string {
	runes := []rune(s)
	if index >= len(runes) {
		return s
	}
	return string(runes[:index]) + string(runes[index+1:])
}

func getFirstNRunes(s string, n int) string {
	runes := []rune(s)
	if n > len(runes) {
		n = len(runes)
	}
	return string(runes[:n])
}
