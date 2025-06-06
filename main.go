package main

import (
	"fmt"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	fontPath = "FiraCode-Regular.ttf"
	fontSize = 16
)

var (
	zoom      = 2.0
	ligatures = []string{
		"->", "=>", "<-", "<=", "==", "!=", "&&", "||", "++", "--",
	}
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

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow("HiDPI Text Viewer", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, 800, 600, sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// set white background
	r, _ := window.GetRenderer()
	r.SetDrawColor(255, 255, 255, 255)

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	atlas := NewGlyphAtlas(renderer, fontPath, int(float64(fontSize)*zoom))
	defer atlas.Destroy()

	sampleText := "Hello, High-DPI World!"
	cursorPos := len(sampleText) // rune-based position

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN && e.Button == sdl.BUTTON_LEFT {
					cursorPos = getCursorPosAtClick(e.X, e.Y, sampleText, atlas, renderer, 10, 10)
				}
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						if cursorPos > 0 {
							sampleText = deleteAtRune(sampleText, cursorPos-1)
							cursorPos--
						}
					case sdl.K_UP:
						cursorPos = moveCursorLineUp(cursorPos, sampleText)
					case sdl.K_DOWN:
						cursorPos = moveCursorLineDown(cursorPos, sampleText)
					case sdl.K_LEFT:
						if cursorPos > 0 {
							cursorPos--
						}
					case sdl.K_RIGHT:
						if cursorPos < len(sampleText) {
							cursorPos++
						}
					case sdl.K_RETURN:
						sampleText = insertAtRune(sampleText, cursorPos, "\n")
						cursorPos++ // Move cursor after the newline
					case sdl.K_TAB:
						sampleText = insertAtRune(sampleText, cursorPos, "    ")
						cursorPos += 4
					}
				}
				if e.Keysym.Sym == sdl.K_ESCAPE && e.State == sdl.PRESSED {
					running = false
				} else if e.Keysym.Sym == sdl.K_EQUALS && e.Keysym.Mod&uint16(sdl.KMOD_CTRL) != 0 && e.State == sdl.PRESSED {
					zoom += 0.5
					atlas.Destroy()
					atlas = NewGlyphAtlas(renderer, fontPath, int(float64(fontSize)*zoom))
				} else if e.Keysym.Sym == sdl.K_MINUS && e.Keysym.Mod&uint16(sdl.KMOD_CTRL) != 0 && e.State == sdl.PRESSED {
					zoom -= 0.5
					if zoom < 0.5 {
						zoom = 0.5
					}
					atlas.Destroy()
					atlas = NewGlyphAtlas(renderer, fontPath, int(float64(fontSize)*zoom))
				}
			case *sdl.TextInputEvent:
				input := e.GetText()
				if input != "" {
					sampleText = insertAtRune(sampleText, cursorPos, input)
					cursorPos += len(input)
				}
			}
		}

		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.Clear()

		y := int32(10)
		var cursorX, cursorY = int32(10), int32(10)
		charCount := 0

		for _, line := range strings.Split(sampleText, "\n") {

			x := int32(10)

			for i := 0; i < len(line); i++ {
				charCount++
				s := string(line[i])
				if i < len(line)-1 && contains(ligatures, line[i:i+2]) {
					s = line[i : i+2]
					i++ // Skip the next character since it's part of the ligature
				}

				tx := atlas.GetTexture(s, renderer)
				if tx == nil {
					continue
				}
				_, _, w, h, _ := tx.Query()
				renderer.Copy(tx, nil, &sdl.Rect{X: int32(x), Y: int32(y), W: w, H: h})
				x += w

				if charCount == cursorPos {
					cursorX = x
					cursorY = y
				}
			}

			y += int32(atlas.Size)
			charCount += 1 // Account for the newline character

			if charCount == cursorPos {
				cursorX = 10
				cursorY = y
			}
		}

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.FillRect(&sdl.Rect{X: cursorX, Y: cursorY, W: 4, H: int32(atlas.Size)})

		renderer.Present()
		sdl.Delay(8)
	}
}

func insertAtRune(s string, pos int, insert string) string {
	runes := []rune(s)
	if pos < 0 || pos > len(runes) {
		return s
	}
	return string(runes[:pos]) + insert + string(runes[pos:])
}

func deleteAtRune(s string, pos int) string {
	runes := []rune(s)
	if pos < 0 || pos >= len(runes) {
		return s
	}
	return string(runes[:pos]) + string(runes[pos+1:])
}

func moveCursorLineUp(cursorPos int, text string) int {
	lines := strings.Split(text, "\n")
	lineIndex := 0
	charCount := 0

	// Find the line index of the current cursor position
	for i, line := range lines {
		if charCount+len(line) >= cursorPos {
			lineIndex = i
			break
		}
		charCount += len(line) + 1 // +1 for newline character
	}

	// Move up one line if possible
	if lineIndex > 0 {
		return charCount - len(lines[lineIndex-1]) - 1 // -1 for newline character
	}
	return cursorPos // Already at the top line
}

func moveCursorLineDown(cursorPos int, text string) int {
	lines := strings.Split(text, "\n")
	lineIndex := 0
	charCount := 0

	// Find the line index of the current cursor position
	for i, line := range lines {
		if charCount+len(line) >= cursorPos {
			lineIndex = i
			break
		}
		charCount += len(line) + 1 // +1 for newline character
	}

	// Move down one line if possible
	if lineIndex < len(lines)-1 {
		return charCount + len(lines[lineIndex]) + 1 // +1 for newline character
	}
	return cursorPos // Already at the bottom line
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func getCursorPosAtClick(mouseX, mouseY int32, text string, atlas *GlyphAtlas, renderer *sdl.Renderer, startX, startY int32) int {
	lines := strings.Split(text, "\n")
	cursorY := startY
	cursorIndex := 0

	for _, line := range lines {
		cursorX := startX
		runes := []rune(line)

		for i, r := range runes {
			char := string(r)
			tx := atlas.GetTexture(char, renderer)
			if tx == nil {
				continue
			}
			_, _, w, _, _ := tx.Query()
			w = int32(float64(w) / zoom)
			y := int32(float64(atlas.Size) / zoom)

			// Hit test for this glyph rectangle
			if mouseX >= cursorX && mouseX < cursorX+w &&
				mouseY >= cursorY && mouseY < cursorY+y {
				fmt.Println("Clicked on character:", char, "at index:", cursorIndex+i+1)
				fmt.Println(mouseX, mouseY, cursorX, cursorY, w, atlas.Size)
				return cursorIndex + i + 1
			}

			cursorX += int32(w)
		}

		// Check if the click was on the newline character
		if mouseX >= cursorX && mouseY >= cursorY && mouseY < cursorY+int32(float64(atlas.Size)/zoom) {
			fmt.Println("Clicked on newline at index:", cursorIndex+len(runes)+1)
			return cursorIndex + len(runes) // +1 for the newline character
		}

		cursorIndex += len(runes) + 1 // +1 for '\n'
		cursorY += int32(float64(atlas.Size) / zoom)
	}

	return len([]rune(text)) // Click was past the end
}
