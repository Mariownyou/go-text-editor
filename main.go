package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	fontPath    = "FiraCode-Regular.ttf"
	fontSize    = 16
	scrollSpeed = 10
)

var (
	zoom                = 2.0
	scrollOffsetY int32 = 0
	curCol              = 0
	curRow              = 0
	ligatures           = []string{
		"->", "=>", "<-", "<=", "==", "!=", "&&", "||", "++", "--",
	}

	rw int32
)

var (
	frameCount    = 0
	fps           = 0
	lastFPSUpdate = sdl.GetTicks64()
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

func main() {
	bufferText := "Hello, High-DPI World!\nasdadasda"

	if len(os.Args) > 1 {
		filePath := os.Args[1]
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}
		bufferText = string(data)
	}

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

	rw, _, _ = renderer.GetOutputSize()

	atlas := NewGlyphAtlas(renderer, fontPath, int(float64(fontSize)*zoom))
	defer atlas.Destroy()

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseWheelEvent:
				scrollAmount := int32(e.Y) * int32(float64(atlas.Size)/2) * 3
				scrollOffsetY -= scrollAmount
				fmt.Println("Scroll offset:", scrollOffsetY, "Scroll amount:", scrollAmount)
				if scrollOffsetY < 0 {
					scrollOffsetY = 0
				}
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN && e.Button == sdl.BUTTON_LEFT {
					x, y := e.X, e.Y
					x, y = GetRealMousePos(x, y, window, renderer)
					y += scrollOffsetY // Adjust for scroll offset
					curRow, curCol = GetRowColFromClick(x, y, bufferText, atlas, renderer)
					// fmt.Println("Cursor position set to row:", curRow, "col:", curCol)
				}
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						oldRow := strings.Split(bufferText, "\n")[curRow]
						bufferText = deleteAtCursor(bufferText, curRow, curCol)
						if curCol > 0 {
							curCol--
						} else if curRow > 0 {
							curRow--
							curCol = len(strings.Split(bufferText, "\n")[curRow]) - len(oldRow)
						}
					case sdl.K_UP:
						if curRow > 0 {
							curRow--
						}
					case sdl.K_DOWN:
						if curRow < len(strings.Split(bufferText, "\n"))-1 {
							curRow++
						}
					case sdl.K_LEFT:
						if curCol > 0 {
							curCol--
						}
					case sdl.K_RIGHT:
						if curCol <= len(strings.Split(bufferText, "\n")[curRow])-1 {
							curCol++
						}
					case sdl.K_RETURN:
						bufferText = insertAtCursor(bufferText, "\n", curRow, curCol)
						curRow++
						curCol = 0
					case sdl.K_TAB:
						bufferText = insertAtCursor(bufferText, "    ", curRow, curCol) // Insert 4 spaces for tab
						curCol += 4
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
					bufferText = insertAtCursor(bufferText, input, curRow, curCol)
					curCol += len([]rune(input))
				}
			}
		}

		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.Clear()

		y := int32(10) - scrollOffsetY
		var cursorX, cursorY = int32(10), int32(10)

		for row, line := range strings.Split(bufferText, "\n") {

			x := int32(10)

			runes := []rune(line)
			for i := 0; i < len(runes); i++ {
				s := string(runes[i])

				if i < len(runes)-1 {
					pair := string(runes[i]) + string(runes[i+1])
					if contains(ligatures, pair) {
						s = pair
						i++ // Skip next rune
					}
				}

				tx := atlas.GetTexture(s, renderer)
				if tx == nil {
					continue
				}
				_, _, w, h, _ := tx.Query()

				if x+w > rw-50 {
					x = 10 // Reset to start of line if it exceeds window width
					y += int32(atlas.Size)
				}

				renderer.Copy(tx, nil, &sdl.Rect{X: int32(x), Y: int32(y), W: w, H: h})

				x += w

				if i+1 == curCol && row == curRow {
					cursorX = x
					cursorY = y
				}
			}

			y += int32(atlas.Size)
		}

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.FillRect(&sdl.Rect{X: cursorX, Y: cursorY, W: 4, H: int32(atlas.Size)})

		frameCount++
		currentTime := sdl.GetTicks64()
		if currentTime-lastFPSUpdate >= 1000 {
			fps = frameCount
			frameCount = 0
			lastFPSUpdate = currentTime
		}

		fpsText := fmt.Sprintf("FPS: %d", fps)
		fpsTexture := atlas.GetTexture(fpsText, renderer)
		if fpsTexture != nil {
			_, _, w, h, _ := fpsTexture.Query()
			rw, _, _ := renderer.GetOutputSize()
			fpsX := rw - w - 10 // 10 pixels from the right edge
			// white background for FPS text
			rect := sdl.Rect{X: fpsX - 5, Y: 5, W: w + 10, H: h + 5}
			renderer.SetDrawColor(255, 255, 255, 200)
			renderer.FillRect(&rect)
			renderer.Copy(fpsTexture, nil, &sdl.Rect{X: fpsX, Y: 10, W: w, H: h})
		}

		renderer.Present()
		sdl.Delay(4)
	}
}

func insertAtCursor(text string, input string, row, col int) string {
	lines := strings.Split(string(text), "\n")

	if row < 0 || row >= len(lines) {
		return string(text) // Invalid row
	}

	line := lines[row]
	runes := []rune(line)

	if col < 0 || col > len(runes) {
		col = len(runes) // Adjust to end of line if col is out of bounds
	}

	lines[row] = string(runes[:col]) + string(input) + string(runes[col:])
	return strings.Join(lines, "\n")
}

func deleteAtCursor(text string, row, col int) string {
	lines := strings.Split(text, "\n")

	if row < 0 || row >= len(lines) {
		return text // Invalid row
	}

	line := lines[row]
	runes := []rune(line)

	if col < 0 || col >= len(runes) {
		return text // Invalid column
	}

	if col == 0 {
		if row > 0 {
			lines[row-1] += string(runes) // Merge with previous line
			lines = append(lines[:row], lines[row+1:]...)
		}
	} else {
		lines[row] = string(runes[:col-1]) + string(runes[col:]) // Delete character at col
	}

	return strings.Join(lines, "\n")
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func GetRowColFromClick(x, y int32, sampleText string, atlas *GlyphAtlas, renderer *sdl.Renderer) (int, int) {

	lines := strings.Split(sampleText, "\n")

	curX := int32(10)
	curY := int32(10) // Starting Y position for the first line

	row, col := 0, 0

	for i, line := range lines {

		row = i

		runes := []rune(line)

		for j, ch := range runes {
			col = j
			tx := atlas.GetTexture(string(ch), renderer)
			if tx == nil {
				continue
			}

			_, _, w, _, _ := tx.Query()

			if curX+w > rw-50 {
				curX = int32(10) // Reset X for the next line
				curY += int32(atlas.Size)
			}

			if curX <= x && curX+w > x &&
				curY <= y && curY+int32(atlas.Size) > y {
				fmt.Println("x:", x, "curX:", curX, "y:", y, "curY:", curY, "w:", w, "col:", col, "string(ch):", string(ch))
				return row, col + 1
			}

			curX += w
		}

		curY += int32(atlas.Size)
		curX = int32(10) // Reset X for the next line
	}

	return row, col + 1 // Return -1, -1 if no valid position found
}

func GetRealMousePos(x, y int32, window *sdl.Window, renderer *sdl.Renderer) (int32, int32) {
	winW, winH := window.GetSize()
	renderW, renderH, _ := renderer.GetOutputSize()

	scaleX := float32(renderW) / float32(winW)
	scaleY := float32(renderH) / float32(winH)

	realX := int32(float32(x) * scaleX)
	realY := int32(float32(y) * scaleY)

	return realX, realY
}
