package main

import (
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

// Selection represents a text selection region
type Selection struct {
	StartRow, StartCol int
	EndRow, EndCol     int
	Active             bool
}

// Cursor represents a single cursor position and its selection
type Cursor struct {
	Row, Col  int
	X, Y      int32 // X and Y are the render positions
	Selection Selection
}

func (c *Cursor) Render(renderer *sdl.Renderer, atlas *GlyphAtlas) {
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.FillRect(&sdl.Rect{X: c.X, Y: c.Y, W: 4, H: int32(atlas.Size)}) // Assuming height of cursor is 16
}

// CursorManager manages multiple cursors (future-ready)
type CursorManager struct {
	Cursors       []Cursor
	PrimaryCursor int // Index of the primary cursor
}

func NewCursorManager() *CursorManager {
	return &CursorManager{
		Cursors: []Cursor{
			{Row: 0, Col: 0, Selection: Selection{}},
		},
		PrimaryCursor: 0,
	}
}

func (cm *CursorManager) GetPrimary() *Cursor {
	return &cm.Cursors[cm.PrimaryCursor]
}

func (cm *CursorManager) ClearAllSelections() {
	for i := range cm.Cursors {
		cm.Cursors[i].Selection.Active = false
	}
}

func (cm *CursorManager) HasSelection() bool {
	for _, cursor := range cm.Cursors {
		if cursor.Selection.Active {
			return true
		}
	}
	return false
}

// GetSelectedText returns the selected text for a given cursor
func (cm *CursorManager) GetSelectedText(cursorIndex int, text string) string {
	if cursorIndex >= len(cm.Cursors) {
		return ""
	}

	cursor := cm.Cursors[cursorIndex]
	if !cursor.Selection.Active {
		return ""
	}

	return GetTextInRange(text, cursor.Selection.StartRow, cursor.Selection.StartCol,
		cursor.Selection.EndRow, cursor.Selection.EndCol)
}

func (cm *CursorManager) SetRenderPos(row, col int, x, y int32) {
	for i := range cm.Cursors {
		if cm.Cursors[i].Row == row && cm.Cursors[i].Col == col {
			cm.Cursors[i].X = x
			cm.Cursors[i].Y = y
			return
		}
	}
}

func RenderCursors(renderer *sdl.Renderer, atlas *GlyphAtlas, cm *CursorManager) {
	for _, cursor := range cm.Cursors {
		cursor.Render(renderer, atlas)
	}
}

func DeleteSelectedText(text string, cm *CursorManager) string {
	// For now, only handle primary cursor
	primary := cm.GetPrimary()
	if !primary.Selection.Active {
		return text
	}

	lines := strings.Split(text, "\n")
	startRow, startCol := primary.Selection.StartRow, primary.Selection.StartCol
	endRow, endCol := primary.Selection.EndRow, primary.Selection.EndCol

	// Normalize selection
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	if startRow < 0 || startRow >= len(lines) || endRow < 0 || endRow >= len(lines) {
		return text
	}

	// Update cursor position to start of deleted selection
	primary.Row = startRow
	primary.Col = startCol

	if startRow == endRow {
		// Single line deletion
		line := lines[startRow]
		runes := []rune(line)
		if startCol < len(runes) && endCol <= len(runes) {
			lines[startRow] = string(runes[:startCol]) + string(runes[endCol:])
		}
	} else {
		// Multi-line deletion
		startLine := lines[startRow]
		endLine := lines[endRow]
		startRunes := []rune(startLine)
		endRunes := []rune(endLine)

		var newLine string
		if startCol < len(startRunes) {
			newLine = string(startRunes[:startCol])
		}
		if endCol < len(endRunes) {
			newLine += string(endRunes[endCol:])
		}

		// Replace the range with the merged line
		newLines := make([]string, 0, len(lines)-(endRow-startRow))
		newLines = append(newLines, lines[:startRow]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, lines[endRow+1:]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

func IsCharacterSelected(row, col int, selection Selection) bool {
	if !selection.Active {
		return false
	}

	startRow, startCol := selection.StartRow, selection.StartCol
	endRow, endCol := selection.EndRow, selection.EndCol

	// Normalize selection (ensure start comes before end)
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	if row < startRow || row > endRow {
		return false
	}

	if row == startRow && row == endRow {
		return col >= startCol && col < endCol
	} else if row == startRow {
		return col >= startCol
	} else if row == endRow {
		return col < endCol
	} else {
		return true // Middle rows are fully selected
	}
}

func GetTextInRange(text string, startRow, startCol, endRow, endCol int) string {
	lines := strings.Split(text, "\n")

	// Normalize selection
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	if startRow < 0 || startRow >= len(lines) || endRow < 0 || endRow >= len(lines) {
		return ""
	}

	var result strings.Builder

	for row := startRow; row <= endRow; row++ {
		line := lines[row]
		runes := []rune(line)

		var startIndex, endIndex int
		if row == startRow {
			startIndex = startCol
		} else {
			startIndex = 0
		}

		if row == endRow {
			endIndex = endCol
		} else {
			endIndex = len(runes)
		}

		if startIndex < len(runes) {
			if endIndex > len(runes) {
				endIndex = len(runes)
			}
			result.WriteString(string(runes[startIndex:endIndex]))
		}

		if row < endRow {
			result.WriteString("\n")
		}
	}

	return result.String()
}
