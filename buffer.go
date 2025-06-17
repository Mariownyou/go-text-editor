package main

const (
	undoStackLimit = 100 // Maximum number of states to keep in the undo stack
)

type UndoStack []string

func NewUndoStack() UndoStack {
	return make(UndoStack, 0)
}

// Push adds a new state to the undo stack.
func (s *UndoStack) Push(state string) {
	if len(*s) >= undoStackLimit {
		// If the stack is full, remove the oldest state
		*s = (*s)[1:]
	}
	*s = append(*s, state)
}

// Pop removes the most recent state from the undo stack and returns it.
func (s *UndoStack) Pop() (string, bool) {
	if len(*s) == 0 {
		return "", false
	}
	state := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return state, true
}

type Buffer struct {
	Content   string
	UndoStack UndoStack
}

func NewBuffer() *Buffer {
	return &Buffer{
		Content:   "",
		UndoStack: NewUndoStack(),
	}
}

func (Buffer *Buffer) SetContent(content string) {
	Buffer.UndoStack.Push(Buffer.Content) // Save the current state before changing
	Buffer.Content = content
}

func (Buffer *Buffer) Undo() bool {
	if state, ok := Buffer.UndoStack.Pop(); ok {
		Buffer.Content = state
		return true
	}
	return false
}
