package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	fontPath        = "FiraCode-Regular.ttf"
	fontSize        = 14
	scrollSpeed     = 100
	scrollLerpSpeed = 0.1 // smaller = slower
)

var (
	zoom                = 2.0
	scrollOffsetY int32 = 0

	targetScrollOffsetY float32 = 0
	actualScrollOffsetY float32 = 0

	ligatures = []string{
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
var isDragging = false

func saveBufferToFile(bufferText string, filePath string) error {
	if filePath == "" {
		filePath = "buffer.txt"
	}
	data := []byte(bufferText)
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save buffer to file: %w", err)
	}
	return nil
}

func main() {
	bufferText := ""
	filePath := ""

	buffer := NewBuffer()

	if len(os.Args) > 1 {
		filePath = os.Args[1]
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}
		bufferText = string(data)
	} else {
		// read from buffer.txt if no file is provided
		data, err := os.ReadFile("buffer.txt")
		if err != nil {
			fmt.Println("Error reading buffer.txt:", err)
			bufferText = "Hello, High-DPI World!\nasdadasda"
		} else {
			bufferText = string(data)
		}
	}

	buffer.SetContent(bufferText)

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow(
		"HiDPI Text Viewer",
		sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, 800, 600,
		sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE,
	)

	SetTitleBarColor(window)

	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// set cursor to selection
	sdl.SetCursor(sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_IBEAM))

	// set white background
	r, _ := window.GetRenderer()
	setColor(r, uiBackgroundColor)

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
				if filePath == "" {
					saveBufferToFile(buffer.Content, "")
				}
				running = false
			case *sdl.WindowEvent:
				if e.Event == sdl.WINDOWEVENT_RESIZED || e.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
					// Update renderer size
					rw, _, _ = renderer.GetOutputSize()
				}
			case *sdl.MouseWheelEvent:
				scrollAmount := float32(e.Y) * scrollSpeed // adjust multiplier to taste
				targetScrollOffsetY -= scrollAmount
				if targetScrollOffsetY < 0 {
					targetScrollOffsetY = 0
				}
			case *sdl.MouseButtonEvent:
				if e.Button != sdl.BUTTON_LEFT {
					continue // Only handle left mouse button events
				}

				x, y := e.X, e.Y
				x, y = GetRealMousePos(x, y, window, renderer)
				y += scrollOffsetY // Adjust for scroll offset
				row, col := GetRowColFromClick(x, y, buffer.Content, atlas, renderer)
				primary := cursorManager.GetPrimary()

				if e.Type == sdl.MOUSEBUTTONDOWN {
					if !primary.Selection.Active && !isDragging ||
						(primary.Selection.Active && (primary.Selection.StartRow != row || primary.Selection.StartCol != col)) {
						isDragging = true
						// Start new selection from current cursor position
						primary.Selection.StartRow = row
						primary.Selection.StartCol = col
					}
					primary.Row, primary.Col = row, col
				} else if e.Type == sdl.MOUSEBUTTONUP {
					// If we dragged, finalize the selection
					if !isDragging {
						cursorManager.ClearAllSelections()
						primary.Selection.Active = false
					}
					isDragging = false

					if primary.Selection.Active && (primary.Selection.StartRow != row || primary.Selection.StartCol != col) {
						primary.Selection.EndRow = row
						primary.Selection.EndCol = col
					}
				}
			case *sdl.MouseMotionEvent:
				if e.State == sdl.PRESSED {
					x, y := e.X, e.Y
					x, y = GetRealMousePos(x, y, window, renderer)
					y += scrollOffsetY
					row, col := GetRowColFromClick(x, y, buffer.Content, atlas, renderer)

					primary := cursorManager.GetPrimary()
					primary.Selection.Active = true
					if primary.Selection.StartRow != row || primary.Selection.StartCol != col {
						primary.Selection.Active = true
						primary.Selection.EndRow = row
						primary.Selection.EndCol = col
						primary.Row = row
						primary.Col = col
					}
				}
			case *sdl.KeyboardEvent:
				primary := cursorManager.GetPrimary()
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_BACKSPACE:
						if cursorManager.HasSelection() {
							buffer.SetContent(DeleteSelectedText(buffer.Content, cursorManager))
							cursorManager.ClearAllSelections()
							continue
						}
						or := strings.Split(buffer.Content, "\n")[primary.Row]
						ol := len([]rune(or))
						buffer.SetContent(deleteAtCursor(buffer.Content, primary.Row, primary.Col))
						if primary.Col > 0 || primary.Row == 0 {
							primary.Col--
						} else {
							nr := strings.Split(buffer.Content, "\n")[primary.Row-1]
							nl := len([]rune(nr))
							primary.Row--
							primary.Col = nl - ol
						}
					case sdl.K_UP:
						if primary.Row > 0 {
							primary.Row--
							primary.Col = 0 // Reset column to 0 when moving up
						}
					case sdl.K_DOWN:
						if primary.Row < len(strings.Split(buffer.Content, "\n"))-1 {
							primary.Row++
							primary.Col = 0 // Reset column to 0 when moving down
						}
					case sdl.K_LEFT:
						if primary.Col > 0 {
							primary.Col--
						}
					case sdl.K_RIGHT:
						if primary.Col <= len([]rune(strings.Split(buffer.Content, "\n")[primary.Row]))-1 {
							primary.Col++
						}
					case sdl.K_RETURN:
						buffer.SetContent(insertAtCursor(buffer.Content, "\n", primary.Row, primary.Col))
						primary.Row++
						primary.Col = 0
					case sdl.K_TAB:
						buffer.SetContent(insertAtCursor(buffer.Content, "    ", primary.Row, primary.Col)) // Insert 4 spaces for tab
						primary.Col += 4
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
				} else if e.Keysym.Sym == sdl.K_v && e.Keysym.Mod&uint16(sdl.KMOD_GUI) != 0 && e.State == sdl.PRESSED {
					fmt.Println("Paste from clipboard")
					clipboardText, err := sdl.GetClipboardText()
					if err != nil {
						fmt.Println("Error getting clipboard text:", err)
						continue
					}
					if clipboardText != "" {
						if cursorManager.HasSelection() {
							buffer.SetContent(DeleteSelectedText(buffer.Content, cursorManager))
							cursorManager.ClearAllSelections()
						}
						old := strings.Split(buffer.Content, "\n")
						ol := len([]rune(old[primary.Row]))
						buffer.SetContent(insertAtCursor(buffer.Content, clipboardText, primary.Row, primary.Col))
						lines := strings.Split(clipboardText, "\n")
						ll := len(lines)
						primary.Col = ol + len([]rune(lines[ll-1]))
						primary.Row += ll - 1 // Move to the last line of the pasted text
					}
				} else if e.Keysym.Sym == sdl.K_a && e.Keysym.Mod&uint16(sdl.KMOD_GUI) != 0 && e.State == sdl.PRESSED {
					fmt.Println("Select all")
					cursorManager.ClearAllSelections()
					primary.Selection.Active = true
					primary.Selection.StartRow = 0
					primary.Selection.StartCol = 0
					primary.Selection.EndRow = len(strings.Split(buffer.Content, "\n")) - 1
					primary.Selection.EndCol = len([]rune(strings.Split(buffer.Content, "\n")[primary.Selection.EndRow]))
					primary.Row = primary.Selection.EndRow
					primary.Col = primary.Selection.EndCol
				} else if e.Keysym.Sym == sdl.K_z && e.Keysym.Mod&uint16(sdl.KMOD_GUI) != 0 && e.State == sdl.PRESSED {
					fmt.Println("Undo last change")
					if buffer.Undo() {
						// Update cursor position after undo
						primary := cursorManager.GetPrimary()
						lines := strings.Split(buffer.Content, "\n")
						if primary.Row >= len(lines) {
							primary.Row = len(lines) - 1
						}
						if primary.Col > len([]rune(lines[primary.Row])) {
							primary.Col = len([]rune(lines[primary.Row]))
						}
					} else {
						fmt.Println("No more undos available")
					}
				} else if e.Keysym.Sym == sdl.K_e && e.Keysym.Mod&uint16(sdl.KMOD_CTRL) != 0 && e.State == sdl.PRESSED {
					fmt.Println("move to end of line")
					if primary.Row < len(strings.Split(buffer.Content, "\n")) {
						line := strings.Split(buffer.Content, "\n")[primary.Row]
						primary.Col = len([]rune(line))
					}
				}
			case *sdl.TextInputEvent:
				input := e.GetText()
				if input != "" {
					primary := cursorManager.GetPrimary()
					if cursorManager.HasSelection() {
						buffer.SetContent(DeleteSelectedText(buffer.Content, cursorManager))
						cursorManager.ClearAllSelections()
					}
					buffer.SetContent(insertAtCursor(buffer.Content, input, primary.Row, primary.Col))
					primary.Col += len([]rune(input))
				}

			}
		}

		delta := targetScrollOffsetY - actualScrollOffsetY
		actualScrollOffsetY += delta * scrollLerpSpeed
		scrollOffsetY = int32(actualScrollOffsetY + 0.5) //- uiHeight*5

		setColor(renderer, uiBackgroundColor)
		renderer.Clear()

		RenderTextWithSelection(renderer, atlas, buffer.Content, cursorManager)

		frameCount++
		currentTime := sdl.GetTicks64()
		if currentTime-lastFPSUpdate >= 1000 {
			fps = frameCount
			frameCount = 0
			lastFPSUpdate = currentTime
		}

		// DrawTabs(renderer, atlas, []string{filePath})
		DrawFPS(renderer, atlas, fps)

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
					_, _, w, _, _ := tx.Query()
					selectionRect := sdl.Rect{X: x, Y: y, W: w, H: int32(atlas.Size + atlas.Size/3)}
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
				x = 10                                // Reset to start of line if it exceeds window width
				y += int32(atlas.Size + atlas.Size/3) // Move to next line
			}

			cm.SetRenderPos(row, i, x, y)
			renderer.Copy(tx, nil, &sdl.Rect{X: int32(x), Y: int32(y), W: w, H: h})

			// Render cursor(s)

			x += w
		}

		cm.SetRenderPos(row, chars, x, y)

		y += int32(atlas.Size + atlas.Size/3) // Move to next line
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

func deleteAtCursor(text string, row, col int) string {
	lines := strings.Split(text, "\n")

	if row < 0 || row >= len(lines) {
		return text // Invalid row
	}

	line := lines[row]
	runes := []rune(line)

	if col < 0 || col > len(runes) {
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

		// handle empty lines
		if len(runes) == 0 {
			if curY <= y && curY+int32(atlas.Size+atlas.Size/3) > y {
				fmt.Println("Empty line clicked at row:", row, "col:", col)
				return row, 0 // Return column 0 for empty lines
			}
			curY += int32(atlas.Size + atlas.Size/3) // Move to next line
			curX = int32(10)                         // Reset X for the next line
			continue
		}

		for j, ch := range runes {
			col = j
			tx := atlas.GetTexture(string(ch), renderer)
			if tx == nil {
				continue
			}

			_, _, w, _, _ := tx.Query()

			if curX+w > rw-50 {
				curX = int32(10)                         // Reset X for the next line
				curY += int32(atlas.Size + atlas.Size/3) // Move to next line
			}

			if curX <= x && curX+w > x &&
				curY <= y && curY+int32(atlas.Size+atlas.Size/3) > y {
				fmt.Println("x:", x, "curX:", curX, "y:", y, "curY:", curY, "w:", w, "col:", col, "string(ch):", string(ch))
				return row, col
			}

			curX += w
		}

		// handle row click but no col
		if curY <= y && curY+int32(atlas.Size+atlas.Size/3) > y {
			fmt.Println("Row clicked at row:", row, "col:", col)
			return row, col + 1 // Return column 0 for row click
		}

		curY += int32(atlas.Size + atlas.Size/3) // Move to next line
		curX = int32(10)                         // Reset X for the next line
	}

	fmt.Println("No valid position found for click at x:", x, "y:", y)

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
