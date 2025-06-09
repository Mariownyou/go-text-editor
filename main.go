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

var cursorManager = NewCursorManager()

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

	// set cursor to selection
	sdl.SetCursor(sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_IBEAM))

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
					primary := cursorManager.GetPrimary()
					x, y := e.X, e.Y
					x, y = GetRealMousePos(x, y, window, renderer)
					y += scrollOffsetY // Adjust for scroll offset
					row, col := GetRowColFromClick(x, y, bufferText, atlas, renderer)
					primary.Row, primary.Col = row, col
				}
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					primary := cursorManager.GetPrimary()
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						oldRow := strings.Split(bufferText, "\n")[primary.Row]
						ol := len([]rune(oldRow))
						if ol+1 == primary.Col {
							bufferText = deleteFromEndOfLine(bufferText, primary.Row)
							if ol == 0 {
								primary.Row--
								primary.Col = len([]rune(strings.Split(bufferText, "\n")[primary.Row])) + 1
							} else {
								primary.Col--
							}
						} else {
							bufferText = deleteAtCursor(bufferText, primary.Row, primary.Col)
							if primary.Col > 0 {
								primary.Col--
							} else if primary.Row > 0 {
								primary.Row--
								primary.Col = len([]rune(strings.Split(bufferText, "\n")[primary.Row])) - ol
							}
						}

					case sdl.K_UP:
						if primary.Row > 0 {
							r := strings.Split(bufferText, "\n")[primary.Row-1]
							ol := len([]rune(r))
							primary.Row--
							if ol == 0 {
								primary.Col = 1
							} else {
								primary.Col = 0
							}
						}
					case sdl.K_DOWN:
						if primary.Row < len(strings.Split(bufferText, "\n"))-1 {
							r := strings.Split(bufferText, "\n")[primary.Row+1]
							ol := len([]rune(r))
							primary.Row++
							if ol == 0 {
								primary.Col = 1
							} else {
								primary.Col = 0
							}
						}
					case sdl.K_LEFT:
						if primary.Col > 0 {
							r := strings.Split(bufferText, "\n")[primary.Row]
							ol := len([]rune(r))
							if ol+1 == primary.Col {
								primary.Col--
							}
							primary.Col--
						}
					case sdl.K_RIGHT:
						if primary.Col <= len([]rune(strings.Split(bufferText, "\n")[curRow]))-1 {
							r := strings.Split(bufferText, "\n")[primary.Row]
							ol := len([]rune(r))
							if ol-1 == primary.Col {
								primary.Col++
							}
							primary.Col++
						}
					case sdl.K_RETURN:
						bufferText = insertAtCursor(bufferText, "\n", primary.Row, primary.Col)
						r := strings.Split(bufferText, "\n")[primary.Row+1]
						ol := len([]rune(r))
						primary.Row++
						primary.Col = 0
						if ol == 0 {
							primary.Col = 1
						}
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
					primary := cursorManager.GetPrimary()
					if cursorManager.HasSelection() {
						bufferText = DeleteSelectedText(bufferText, cursorManager)
						cursorManager.ClearAllSelections()
					}
					bufferText = insertAtCursor(bufferText, input, primary.Row, primary.Col)
					primary.Col += len([]rune(input))
				}

			}
		}

		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.Clear()

		RenderTextWithSelection(renderer, atlas, bufferText, cursorManager)

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

func RenderTextWithSelection(renderer *sdl.Renderer, atlas *GlyphAtlas, text string, cm *CursorManager) {
	y := int32(10) - scrollOffsetY
	primary := cm.GetPrimary()

	for row, line := range strings.Split(text, "\n") {
		x := int32(10)
		runes := []rune(line)

		chars := 0

		for i := 0; i < len(runes); i++ {
			s := string(runes[i])
			chars++

			// Handle ligatures
			if i < len(runes)-1 {
				pair := string(runes[i]) + string(runes[i+1])
				if contains(ligatures, pair) {
					s = pair
					i++ // Skip next rune
				}
			}

			// Check if this character is selected
			isSelected := IsCharacterSelected(row, i, primary.Selection)

			// Render selection background
			if isSelected {
				tx := atlas.GetTexture(s, renderer)
				if tx != nil {
					_, _, w, h, _ := tx.Query()
					selectionRect := sdl.Rect{X: x, Y: y, W: w, H: h}
					renderer.SetDrawColor(173, 216, 230, 128) // Light blue selection
					renderer.FillRect(&selectionRect)
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

			cm.SetRenderPos(row, i, x, y)
			renderer.Copy(tx, nil, &sdl.Rect{X: int32(x), Y: int32(y), W: w, H: h})

			// Render cursor(s)

			x += w
		}

		cm.SetRenderPos(row, chars+1, x, y)

		y += int32(atlas.Size)
	}

	RenderCursors(renderer, atlas, cm)
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

func deleteFromEndOfLine(text string, row int) string {
	lines := strings.Split(text, "\n")

	if row < 0 || row >= len(lines) {
		return text // Invalid row
	}

	line := lines[row]
	runes := []rune(line)

	if len(runes) == 0 {
		// delete newline
		if row > 0 {
			lines[row-1] += lines[row]
			lines = append(lines[:row], lines[row+1:]...) // Remove current line
		} else {
			return text
		}
		return strings.Join(lines, "\n")
	}

	lines[row] = string(runes[:len(runes)-1]) // Delete last character of the line
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
				return row, col
			}

			curX += w
		}

		if curX < x && curY <= y &&
			curY+int32(atlas.Size) > y {
			fmt.Println("x:", x, "curX:", curX, "y:", y, "curY:", curY, "col:", col+1)
			return row, len(runes) + 1 // Return the end of the line
		}

		curY += int32(atlas.Size)
		curX = int32(10) // Reset X for the next line
	}

	return row, col // Return -1, -1 if no valid position found
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
