package editor

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
)

type Position struct {
	Line int
	Col  int
}

type UndoEntry struct {
	Lines   [][]rune
	CursorX int
	CursorY int
}

type Buffer struct {
	lines        [][]rune
	cursorX      int
	cursorY      int
	selStart     Position
	selEnd       Position
	hasSelection bool
	undoStack    []UndoEntry
	redoStack    []UndoEntry
	maxUndo      int
}

func NewBuffer(text string) *Buffer {
	lines := strings.Split(text, "\n")
	rlines := make([][]rune, len(lines))
	for i, l := range lines {
		rlines[i] = []rune(l)
	}
	if len(rlines) == 0 {
		rlines = [][]rune{{}}
	}
	return &Buffer{
		lines:   rlines,
		maxUndo: 100,
	}
}

func (b *Buffer) Text() string {
	var sb strings.Builder
	for i, l := range b.lines {
		sb.WriteString(string(l))
		if i < len(b.lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (b *Buffer) pushUndo() {
	linesCopy := make([][]rune, len(b.lines))
	for i, l := range b.lines {
		linesCopy[i] = make([]rune, len(l))
		copy(linesCopy[i], l)
	}
	b.undoStack = append(b.undoStack, UndoEntry{
		Lines:   linesCopy,
		CursorX: b.cursorX,
		CursorY: b.cursorY,
	})
	if len(b.undoStack) > b.maxUndo {
		b.undoStack = b.undoStack[1:]
	}
	b.redoStack = nil
}

func (b *Buffer) Undo() {
	if len(b.undoStack) == 0 {
		return
	}
	
	linesCopy := make([][]rune, len(b.lines))
	for i, l := range b.lines {
		linesCopy[i] = make([]rune, len(l))
		copy(linesCopy[i], l)
	}
	b.redoStack = append(b.redoStack, UndoEntry{
		Lines:   linesCopy,
		CursorX: b.cursorX,
		CursorY: b.cursorY,
	})

	entry := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]
	
	b.lines = entry.Lines
	b.cursorX = entry.CursorX
	b.cursorY = entry.CursorY
	b.ClearSelection()
}

func (b *Buffer) Redo() {
	if len(b.redoStack) == 0 {
		return
	}
	
	linesCopy := make([][]rune, len(b.lines))
	for i, l := range b.lines {
		linesCopy[i] = make([]rune, len(l))
		copy(linesCopy[i], l)
	}
	b.undoStack = append(b.undoStack, UndoEntry{
		Lines:   linesCopy,
		CursorX: b.cursorX,
		CursorY: b.cursorY,
	})

	entry := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]
	
	b.lines = entry.Lines
	b.cursorX = entry.CursorX
	b.cursorY = entry.CursorY
	b.ClearSelection()
}

func (b *Buffer) insertRuneRaw(r rune) {
	if r == '\n' {
		left := b.lines[b.cursorY][:b.cursorX]
		right := b.lines[b.cursorY][b.cursorX:]
		b.lines[b.cursorY] = left
		
		newLines := make([][]rune, 0, len(b.lines)+1)
		newLines = append(newLines, b.lines[:b.cursorY+1]...)
		newRight := make([]rune, len(right))
		copy(newRight, right)
		newLines = append(newLines, newRight)
		if b.cursorY+1 < len(b.lines) {
			newLines = append(newLines, b.lines[b.cursorY+1:]...)
		}
		b.lines = newLines
		b.cursorY++
		b.cursorX = 0
	} else {
		line := b.lines[b.cursorY]
		line = append(line[:b.cursorX], append([]rune{r}, line[b.cursorX:]...)...)
		b.lines[b.cursorY] = line
		b.cursorX++
	}
}

func (b *Buffer) InsertRune(r rune) {
	b.pushUndo()
	if b.HasSelection() {
		b.DeleteSelection()
	}
	b.insertRuneRaw(r)
}

func (b *Buffer) HasSelection() bool {
	if !b.hasSelection {
		return false
	}
	start, end := b.getSelectionRange()
	return start.Line != end.Line || start.Col != end.Col
}

func (b *Buffer) InsertString(s string) {
	b.pushUndo()
	if b.HasSelection() {
		b.DeleteSelection()
	}
	runes := []rune(s)
	for _, r := range runes {
		if r == '\r' {
			continue
		}
		b.insertRuneRaw(r)
	}
}

func (b *Buffer) IndentSelection() {
	if !b.HasSelection() {
		return
	}
	b.pushUndo()
	start, end := b.getSelectionRange()
	for l := start.Line; l <= end.Line; l++ {
		b.lines[l] = append([]rune("    "), b.lines[l]...)
	}
	b.selStart = Position{Line: start.Line, Col: start.Col + 4}
	b.selEnd = Position{Line: end.Line, Col: end.Col + 4}
	b.cursorX += 4
}

func (b *Buffer) UnindentSelection() {
	if !b.HasSelection() {
		return
	}
	b.pushUndo()
	start, end := b.getSelectionRange()
	for l := start.Line; l <= end.Line; l++ {
		line := b.lines[l]
		spaces := 0
		for spaces < 4 && spaces < len(line) && line[spaces] == ' ' {
			spaces++
		}
		if spaces > 0 {
			b.lines[l] = line[spaces:]
		}
	}
	colStart := start.Col - 4
	if colStart < 0 {
		colStart = 0
	}
	colEnd := end.Col - 4
	if colEnd < 0 {
		colEnd = 0
	}
	b.selStart = Position{Line: start.Line, Col: colStart}
	b.selEnd = Position{Line: end.Line, Col: colEnd}
	if b.cursorX >= 4 {
		b.cursorX -= 4
	} else {
		b.cursorX = 0
	}
}

func (b *Buffer) DeleteBack() {
	if b.HasSelection() {
		b.pushUndo()
		b.DeleteSelection()
		return
	}
	if b.cursorX > 0 {
		b.pushUndo()
		line := b.lines[b.cursorY]
		b.lines[b.cursorY] = append(line[:b.cursorX-1], line[b.cursorX:]...)
		b.cursorX--
	} else if b.cursorY > 0 {
		b.pushUndo()
		prevLen := len(b.lines[b.cursorY-1])
		b.lines[b.cursorY-1] = append(b.lines[b.cursorY-1], b.lines[b.cursorY]...)
		b.lines = append(b.lines[:b.cursorY], b.lines[b.cursorY+1:]...)
		b.cursorY--
		b.cursorX = prevLen
	}
}

func (b *Buffer) DeleteForward() {
	if b.HasSelection() {
		b.pushUndo()
		b.DeleteSelection()
		return
	}
	if b.cursorX < len(b.lines[b.cursorY]) {
		b.pushUndo()
		line := b.lines[b.cursorY]
		b.lines[b.cursorY] = append(line[:b.cursorX], line[b.cursorX+1:]...)
	} else if b.cursorY < len(b.lines)-1 {
		b.pushUndo()
		b.lines[b.cursorY] = append(b.lines[b.cursorY], b.lines[b.cursorY+1]...)
		b.lines = append(b.lines[:b.cursorY+1], b.lines[b.cursorY+2:]...)
	}
}

func (b *Buffer) getSelectionRange() (Position, Position) {
	start, end := b.selStart, b.selEnd
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}
	return start, end
}

func (b *Buffer) DeleteSelection() {
	if !b.HasSelection() {
		b.ClearSelection()
		return
	}
	start, end := b.getSelectionRange()
	
	if start.Line == end.Line {
		line := b.lines[start.Line]
		b.lines[start.Line] = append(line[:start.Col], line[end.Col:]...)
	} else {
		firstLine := b.lines[start.Line][:start.Col]
		lastLine := b.lines[end.Line][end.Col:]
		
		b.lines[start.Line] = append(firstLine, lastLine...)
		
		b.lines = append(b.lines[:start.Line+1], b.lines[end.Line+1:]...)
	}
	b.cursorY = start.Line
	b.cursorX = start.Col
	b.ClearSelection()
}

func (b *Buffer) MoveCursor(dx, dy int) {
	b.cursorY += dy
	if b.cursorY < 0 {
		b.cursorY = 0
	} else if b.cursorY >= len(b.lines) {
		b.cursorY = len(b.lines) - 1
	}
	
	b.cursorX += dx
	if b.cursorX < 0 {
		b.cursorX = 0
	}
	if b.cursorX > len(b.lines[b.cursorY]) {
		b.cursorX = len(b.lines[b.cursorY])
	}
}

func (b *Buffer) MoveWordLeft() {
	if b.cursorX == 0 {
		if b.cursorY > 0 {
			b.cursorY--
			b.cursorX = len(b.lines[b.cursorY])
		}
		return
	}
	
	line := b.lines[b.cursorY]
	x := b.cursorX - 1
	for x > 0 && unicode.IsSpace(line[x]) {
		x--
	}
	for x > 0 && !unicode.IsSpace(line[x-1]) {
		x--
	}
	b.cursorX = x
}

func (b *Buffer) MoveWordRight() {
	line := b.lines[b.cursorY]
	if b.cursorX == len(line) {
		if b.cursorY < len(b.lines)-1 {
			b.cursorY++
			b.cursorX = 0
		}
		return
	}
	
	x := b.cursorX
	for x < len(line) && !unicode.IsSpace(line[x]) {
		x++
	}
	for x < len(line) && unicode.IsSpace(line[x]) {
		x++
	}
	b.cursorX = x
}

func (b *Buffer) Home() {
	b.cursorX = 0
}

func (b *Buffer) End() {
	b.cursorX = len(b.lines[b.cursorY])
}

func (b *Buffer) PageUp(pageSize int) {
	b.cursorY -= pageSize
	if b.cursorY < 0 {
		b.cursorY = 0
	}
	if b.cursorX > len(b.lines[b.cursorY]) {
		b.cursorX = len(b.lines[b.cursorY])
	}
}

func (b *Buffer) PageDown(pageSize int) {
	b.cursorY += pageSize
	if b.cursorY >= len(b.lines) {
		b.cursorY = len(b.lines) - 1
	}
	if b.cursorX > len(b.lines[b.cursorY]) {
		b.cursorX = len(b.lines[b.cursorY])
	}
}

func (b *Buffer) StartSelection() {
	b.hasSelection = true
	b.selStart = Position{Line: b.cursorY, Col: b.cursorX}
	b.selEnd = b.selStart
}

func (b *Buffer) UpdateSelection() {
	if b.hasSelection {
		b.selEnd = Position{Line: b.cursorY, Col: b.cursorX}
	}
}

func (b *Buffer) ClearSelection() {
	b.hasSelection = false
}

func (b *Buffer) SelectedText() string {
	if !b.HasSelection() {
		return ""
	}
	start, end := b.getSelectionRange()
	
	if start.Line == end.Line {
		return string(b.lines[start.Line][start.Col:end.Col])
	}
	
	var sb strings.Builder
	sb.WriteString(string(b.lines[start.Line][start.Col:]))
	sb.WriteString("\n")
	
	for i := start.Line + 1; i < end.Line; i++ {
		sb.WriteString(string(b.lines[i]))
		sb.WriteString("\n")
	}
	
	sb.WriteString(string(b.lines[end.Line][:end.Col]))
	return sb.String()
}

func (b *Buffer) CurrentWord() string {
	line := b.lines[b.cursorY]
	if len(line) == 0 {
		return ""
	}
	
	x := b.cursorX
	if x >= len(line) {
		x = len(line) - 1
	}
	
	if unicode.IsSpace(line[x]) {
		return ""
	}
	
	start := x
	for start > 0 && !unicode.IsSpace(line[start-1]) {
		start--
	}
	
	end := x
	for end < len(line) && !unicode.IsSpace(line[end]) {
		end++
	}
	
	return string(line[start:end])
}

// SelectAll selects all text in the buffer across all lines.
func (b *Buffer) SelectAll() {
	if len(b.lines) == 0 {
		return
	}
	b.selStart = Position{Line: 0, Col: 0}
	lastLine := len(b.lines) - 1
	b.selEnd = Position{Line: lastLine, Col: len(b.lines[lastLine])}
	b.hasSelection = true
	b.cursorY = lastLine
	b.cursorX = len(b.lines[lastLine])
}

func (b *Buffer) WrapOrUnwrapSelection(prefix, suffix string) {
	if !b.HasSelection() {
		return
	}
	b.pushUndo()
	
	text := b.SelectedText()
	start, _ := b.getSelectionRange()

	if strings.Contains(text, "\n") {
		// Multi-line selection: format each line individually
		lines := strings.Split(text, "\n")
		isBlockPrefix := suffix == "" && (strings.HasPrefix(prefix, "#") || strings.HasPrefix(prefix, ">") || strings.HasPrefix(prefix, ":::"))

		allFormatted := true
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if len(trimmed) == 0 {
				continue
			}
			if isBlockPrefix {
				if !strings.HasPrefix(l, prefix) {
					allFormatted = false
					break
				}
			} else {
				if !strings.HasPrefix(l, prefix) || !strings.HasSuffix(l, suffix) {
					allFormatted = false
					break
				}
			}
		}

		var newLines []string
		for _, l := range lines {
			if len(strings.TrimSpace(l)) == 0 {
				newLines = append(newLines, l)
				continue
			}
			if allFormatted {
				// Unwrap / remove formatting
				if isBlockPrefix {
					if strings.HasPrefix(l, prefix) {
						l = l[len(prefix):]
					}
				} else {
					if strings.HasPrefix(l, prefix) && strings.HasSuffix(l, suffix) {
						l = l[len(prefix) : len(l)-len(suffix)]
					}
				}
			} else {
				// Wrap / apply formatting to each selected line
				if isBlockPrefix {
					if !strings.HasPrefix(l, prefix) {
						l = prefix + l
					}
				} else {
					if !strings.HasPrefix(l, prefix) || !strings.HasSuffix(l, suffix) {
						l = prefix + l + suffix
					}
				}
			}
			newLines = append(newLines, l)
		}
		text = strings.Join(newLines, "\n")
	} else {
		// Single-line selection
		if strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix) {
			text = text[len(prefix) : len(text)-len(suffix)]
		} else {
			text = prefix + text + suffix
		}
	}
	
	b.DeleteSelection()
	b.cursorY = start.Line
	b.cursorX = start.Col
	for _, r := range []rune(text) {
		if r == '\r' {
			continue
		}
		b.insertRuneRaw(r)
	}

	// Keep selection active over the newly formatted range
	b.selStart = Position{Line: start.Line, Col: start.Col}
	b.selEnd = Position{Line: b.cursorY, Col: b.cursorX}
	b.hasSelection = true
}

func (b *Buffer) WrapOrUnwrapWord(prefix, suffix string) {
	line := b.lines[b.cursorY]
	if len(line) == 0 {
		return
	}
	
	x := b.cursorX
	if x >= len(line) {
		x = len(line) - 1
	}
	if unicode.IsSpace(line[x]) {
		return
	}
	
	start := x
	for start > 0 && !unicode.IsSpace(line[start-1]) {
		start--
	}
	
	end := x
	for end < len(line) && !unicode.IsSpace(line[end]) {
		end++
	}
	
	word := string(line[start:end])
	b.pushUndo()
	
	var newWord string
	if strings.HasPrefix(word, prefix) && strings.HasSuffix(word, suffix) {
		newWord = word[len(prefix) : len(word)-len(suffix)]
	} else {
		newWord = prefix + word + suffix
	}
	
	b.lines[b.cursorY] = append(line[:start], append([]rune(newWord), line[end:]...)...)
	b.cursorX = start + len([]rune(newWord))
}

func (b *Buffer) LineCount() int {
	return len(b.lines)
}

type EditorMode int

const (
	EditMode EditorMode = iota
	PreviewMode
)

type Model struct {
	buffer         *Buffer
	width          int
	height         int
	scrollOffset   int
	mode           EditorMode
	showFormatPane bool
	wantsImage     bool
	dirty          bool
	lastEdit       time.Time
	saved          bool
	savedTimer     int
	theme          *markup.Theme
	focused        bool
	OnSave         func(content string)
}

func New(content string, theme *markup.Theme) Model {
	if theme == nil {
		theme = markup.NewDefaultTheme()
	}
	return Model{
		buffer: NewBuffer(content),
		mode:   EditMode,
		theme:  theme,
		dirty:  false,
	}
}

func (m Model) Content() string {
	return m.buffer.Text()
}

func (m Model) IsDirty() bool {
	return m.dirty
}

func (m Model) WantsImage() bool {
	return m.wantsImage
}

func (m *Model) ClearWantsImage() {
	m.wantsImage = false
}

func (m Model) IsFormatPaneOpen() bool {
	return m.showFormatPane
}

func (m Model) HasSelection() bool {
	return m.buffer.HasSelection()
}

func (m *Model) SetContent(s string) {
	m.buffer = NewBuffer(s)
	m.dirty = false
	m.saved = false
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) Focus() {
	m.focused = true
}

func (m *Model) Blur() {
	m.focused = false
}

// InsertText inserts text at the current cursor position.
func (m *Model) InsertText(s string) {
	m.buffer.InsertString(s)
	m.dirty = true
	m.lastEdit = time.Now()
}

type autoSaveMsg struct{}
type savedIndicatorTickMsg struct{}

func (m Model) Init() tea.Cmd {
	return tickAutosave()
}

func tickAutosave() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return autoSaveMsg{}
	})
}

func tickSavedIndicator() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return savedIndicatorTickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	
	case autoSaveMsg:
		if m.dirty && time.Since(m.lastEdit) >= 2*time.Second {
			if m.OnSave != nil {
				m.OnSave(m.buffer.Text())
			}
			m.dirty = false
			m.saved = true
			m.savedTimer = 2
			cmds = append(cmds, tickSavedIndicator())
		}
		cmds = append(cmds, tickAutosave())
		
	case savedIndicatorTickMsg:
		if m.savedTimer > 0 {
			m.savedTimer--
			cmds = append(cmds, tickSavedIndicator())
		} else {
			m.saved = false
		}
		
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		
		// Direct global shortcut handlers for editor
		strLower := strings.ToLower(msg.String())
		switch strLower {
		case "shift+f":
			if m.mode == EditMode {
				m.showFormatPane = !m.showFormatPane
			}
			return m, tea.Batch(cmds...)
		case "ctrl+f":
			// Reserved for Find/Search functionality in the future (preserves selection & content)
			return m, tea.Batch(cmds...)
		case "ctrl+r":
			if m.mode == EditMode {
				m.mode = PreviewMode
			} else {
				m.mode = EditMode
			}
			return m, tea.Batch(cmds...)
		case "esc":
			if m.showFormatPane {
				m.showFormatPane = false
				return m, tea.Batch(cmds...)
			}
			if m.buffer.HasSelection() {
				m.buffer.ClearSelection()
				return m, tea.Batch(cmds...)
			}
			return m, tea.Batch(cmds...)
		case "ctrl+z":
			m.buffer.Undo()
			m.markDirty()
			return m, tea.Batch(cmds...)
		case "ctrl+y":
			m.buffer.Redo()
			m.markDirty()
			return m, tea.Batch(cmds...)
		case "ctrl+b":
			if m.buffer.HasSelection() {
				m.buffer.WrapOrUnwrapSelection("**", "**")
			} else {
				m.buffer.WrapOrUnwrapWord("**", "**")
			}
			m.markDirty()
			return m, tea.Batch(cmds...)
		case "ctrl+u":
			if m.buffer.HasSelection() {
				m.buffer.WrapOrUnwrapSelection("__", "__")
			} else {
				m.buffer.WrapOrUnwrapWord("__", "__")
			}
			m.markDirty()
			return m, tea.Batch(cmds...)
		case "alt+i":
			m.wantsImage = true
			return m, tea.Batch(cmds...)
		case "ctrl+a":
			m.buffer.SelectAll()
			return m, tea.Batch(cmds...)
		case "ctrl+c":
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
			}
			return m, tea.Batch(cmds...)
		case "ctrl+x":
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
				m.buffer.DeleteSelection()
				m.markDirty()
			}
			return m, tea.Batch(cmds...)
		case "ctrl+v":
			text, err := clipboard.ReadAll()
			if err == nil && text != "" {
				m.buffer.InsertString(text)
				m.markDirty()
			}
			return m, tea.Batch(cmds...)
		case "ctrl", "alt", "shift":
			// Solitary modifier key press: NEVER touch selection or buffer!
			return m, tea.Batch(cmds...)
		}

		if msg.String() == "F" && m.buffer.HasSelection() && m.mode == EditMode {
			m.showFormatPane = !m.showFormatPane
			return m, tea.Batch(cmds...)
		}

		if m.showFormatPane && m.mode == EditMode {
			k := strings.ToLower(msg.String())
			handled := true
			switch k {
			case "b", "ctrl+b":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("**", "**")
				} else {
					m.buffer.WrapOrUnwrapWord("**", "**")
				}
			case "i", "ctrl+i":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("*", "*")
				} else {
					m.buffer.WrapOrUnwrapWord("*", "*")
				}
			case "u", "ctrl+u":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("__", "__")
				} else {
					m.buffer.WrapOrUnwrapWord("__", "__")
				}
			case "s", "ctrl+s":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("~~", "~~")
				} else {
					m.buffer.WrapOrUnwrapWord("~~", "~~")
				}
			case "h":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("==", "==")
				} else {
					m.buffer.WrapOrUnwrapWord("==", "==")
				}
			case "1":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("# ", "")
				} else {
					m.buffer.InsertString("# ")
				}
			case "2":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("## ", "")
				} else {
					m.buffer.InsertString("## ")
				}
			case "3":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("### ", "")
				} else {
					m.buffer.InsertString("### ")
				}
			case "q":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("> ", "")
				} else {
					m.buffer.InsertString("> ")
				}
			case "m":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection("`", "`")
				} else {
					m.buffer.WrapOrUnwrapWord("`", "`")
				}
			case "p", "alt+i":
				m.wantsImage = true
				m.showFormatPane = false
				return m, tea.Batch(cmds...)
			case "l":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection(":::left\n", "")
				} else {
					m.buffer.InsertString("\n:::left\n")
				}
			case "c":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection(":::center\n", "\n:::left")
				} else {
					m.buffer.InsertString("\n:::center\n")
				}
			case "r":
				if m.buffer.HasSelection() {
					m.buffer.WrapOrUnwrapSelection(":::right\n", "\n:::left")
				} else {
					m.buffer.InsertString("\n:::right\n")
				}
			case "esc", "shift+f", "f":
				m.showFormatPane = false
			default:
				if msg.String() == "H" {
					if m.buffer.HasSelection() {
						m.buffer.WrapOrUnwrapSelection("^^", "^^")
					} else {
						m.buffer.WrapOrUnwrapWord("^^", "^^")
					}
				} else {
					handled = false
				}
			}
			
			if handled {
				m.dirty = true
				m.lastEdit = time.Now()
				m.showFormatPane = false
				return m, tea.Batch(cmds...)
			}

			// If format pane is open and key is not handled, do not fall through to typing or deleting selection!
			if k != "ctrl" && k != "alt" && k != "shift" {
				m.showFormatPane = false
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.Type {
		case tea.KeyCtrlR:
			if m.mode == EditMode {
				m.mode = PreviewMode
			} else {
				m.mode = EditMode
			}
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlF:
			// Reserved for Find/Search functionality in the future (preserves selection & content)
			return m, tea.Batch(cmds...)
		case tea.KeyEsc:
			if m.showFormatPane {
				m.showFormatPane = false
				return m, tea.Batch(cmds...)
			}
			if m.buffer.HasSelection() {
				m.buffer.ClearSelection()
				return m, tea.Batch(cmds...)
			}
		}

		if m.mode == PreviewMode {
			switch msg.Type {
			case tea.KeyUp, tea.KeyPgUp:
				m.scrollOffset -= 5
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			case tea.KeyDown, tea.KeyPgDown:
				m.scrollOffset += 5
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+a":
			m.buffer.SelectAll()
			return m, tea.Batch(cmds...)
		case "shift+up":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(0, -1)
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		case "shift+down":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(0, 1)
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		case "shift+left":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(-1, 0)
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		case "shift+right":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(1, 0)
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		case "shift+home":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.Home()
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		case "shift+end":
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.End()
			m.buffer.UpdateSelection()
			return m, tea.Batch(cmds...)
		}

		// Edit mode bindings
		switch msg.Type {
		case tea.KeyCtrlA:
			m.buffer.SelectAll()
		case tea.KeyUp:
			m.buffer.ClearSelection()
			m.buffer.MoveCursor(0, -1)
		case tea.KeyDown:
			m.buffer.ClearSelection()
			m.buffer.MoveCursor(0, 1)
		case tea.KeyLeft:
			m.buffer.ClearSelection()
			m.buffer.MoveCursor(-1, 0)
		case tea.KeyRight:
			m.buffer.ClearSelection()
			m.buffer.MoveCursor(1, 0)
		case tea.KeyShiftUp:
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(0, -1)
			m.buffer.UpdateSelection()
		case tea.KeyShiftDown:
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(0, 1)
			m.buffer.UpdateSelection()
		case tea.KeyShiftLeft:
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(-1, 0)
			m.buffer.UpdateSelection()
		case tea.KeyShiftRight:
			if !m.buffer.hasSelection {
				m.buffer.StartSelection()
			}
			m.buffer.MoveCursor(1, 0)
			m.buffer.UpdateSelection()
		case tea.KeyCtrlLeft:
			m.buffer.ClearSelection()
			m.buffer.MoveWordLeft()
		case tea.KeyCtrlRight:
			m.buffer.ClearSelection()
			m.buffer.MoveWordRight()
		case tea.KeyHome:
			m.buffer.ClearSelection()
			m.buffer.Home()
		case tea.KeyEnd:
			m.buffer.ClearSelection()
			m.buffer.End()
		case tea.KeyPgUp:
			m.buffer.ClearSelection()
			m.buffer.PageUp(m.height - 2)
		case tea.KeyPgDown:
			m.buffer.ClearSelection()
			m.buffer.PageDown(m.height - 2)
		case tea.KeyBackspace:
			m.buffer.DeleteBack()
			m.markDirty()
		case tea.KeyDelete:
			m.buffer.DeleteForward()
			m.markDirty()
		case tea.KeyEnter:
			m.buffer.InsertRune('\n')
			m.markDirty()
		case tea.KeyTab:
			if m.buffer.HasSelection() {
				m.buffer.IndentSelection()
			} else {
				m.buffer.InsertString("    ")
			}
			m.markDirty()
			return m, tea.Batch(cmds...)
		case tea.KeyShiftTab:
			if m.buffer.HasSelection() {
				m.buffer.UnindentSelection()
				m.markDirty()
			}
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlZ:
			m.buffer.Undo()
			m.markDirty()
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlY:
			m.buffer.Redo()
			m.markDirty()
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlB:
			if m.buffer.HasSelection() {
				m.buffer.WrapOrUnwrapSelection("**", "**")
			} else {
				m.buffer.WrapOrUnwrapWord("**", "**")
			}
			m.markDirty()
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlU:
			if m.buffer.HasSelection() {
				m.buffer.WrapOrUnwrapSelection("__", "__")
			} else {
				m.buffer.WrapOrUnwrapWord("__", "__")
			}
			m.markDirty()
			return m, tea.Batch(cmds...)
		case tea.KeyNull:
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlC:
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
			}
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlX:
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
				m.buffer.DeleteSelection()
				m.markDirty()
			}
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlV:
			text, err := clipboard.ReadAll()
			if err == nil && text != "" {
				m.buffer.InsertString(text)
				m.markDirty()
			}
			return m, tea.Batch(cmds...)
		case tea.KeyRunes, tea.KeySpace:
			strVal := strings.ToLower(msg.String())
			if strings.HasPrefix(strVal, "ctrl") || strings.HasPrefix(strVal, "alt") {
				// Holding Ctrl or pressing Ctrl combinations MUST NEVER delete selection or insert control characters!
				return m, tea.Batch(cmds...)
			}
			if len(msg.Runes) > 0 && unicode.IsControl(msg.Runes[0]) {
				return m, tea.Batch(cmds...)
			}
			if msg.Alt {
				// Alt bindings for formats
				switch msg.String() {
				case "alt+b":
					m.toggleFormat("**", "**")
				case "alt+i":
					m.toggleFormat("*", "*")
				case "alt+u":
					m.toggleFormat("__", "__")
				case "alt+s":
					m.toggleFormat("~~", "~~")
				case "alt+l":
					m.buffer.InsertString("\n:::left\n")
					m.markDirty()
				case "alt+e", "alt+c":
					m.buffer.InsertString("\n:::center\n")
					m.markDirty()
				case "alt+r":
					m.buffer.InsertString("\n:::right\n")
					m.markDirty()
				}
			} else {
				if len(msg.Runes) > 0 && !unicode.IsControl(msg.Runes[0]) {
					m.buffer.InsertString(string(msg.Runes))
					m.markDirty()
				} else if msg.Type == tea.KeySpace {
					m.buffer.InsertRune(' ')
					m.markDirty()
				}
			}
		}

	case tea.MouseMsg:
		if !m.focused {
			return m, nil
		}
		if msg.Type == tea.MouseWheelUp {
			if m.mode == PreviewMode {
				m.scrollOffset -= 3
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			} else {
				m.buffer.MoveCursor(0, -3)
			}
		} else if msg.Type == tea.MouseWheelDown {
			if m.mode == PreviewMode {
				m.scrollOffset += 3
			} else {
				m.buffer.MoveCursor(0, 3)
			}
		} else if msg.Button == tea.MouseButtonLeft && m.mode == EditMode {
			contentWidth := m.width - 9
			if contentWidth < 10 {
				contentWidth = 10
			}

			clickedVisualY := msg.Y - 2 + m.scrollOffset
			if clickedVisualY < 0 {
				clickedVisualY = 0
			}
			clickedCol := msg.X - 7
			if clickedCol < 0 {
				clickedCol = 0
			}

			curVisRow := 0
			targetLine := 0
			targetCol := 0
			found := false

			for bIdx, lRunes := range m.buffer.lines {
				numRows := len(lRunes) / contentWidth
				if len(lRunes)%contentWidth != 0 || len(lRunes) == 0 {
					numRows++
				}
				if clickedVisualY >= curVisRow && clickedVisualY < curVisRow+numRows {
					targetLine = bIdx
					rowOffset := clickedVisualY - curVisRow
					targetCol = rowOffset*contentWidth + clickedCol
					if targetCol > len(lRunes) {
						targetCol = len(lRunes)
					}
					found = true
					break
				}
				curVisRow += numRows
			}

			if !found && len(m.buffer.lines) > 0 {
				targetLine = len(m.buffer.lines) - 1
				targetCol = len(m.buffer.lines[targetLine])
			}

			if msg.Action == tea.MouseActionPress {
				if msg.Shift {
					m.buffer.cursorY = targetLine
					m.buffer.cursorX = targetCol
					m.buffer.UpdateSelection()
				} else {
					m.buffer.cursorY = targetLine
					m.buffer.cursorX = targetCol
					m.buffer.StartSelection()
				}
			} else if msg.Action == tea.MouseActionMotion {
				m.buffer.cursorY = targetLine
				m.buffer.cursorX = targetCol
				m.buffer.UpdateSelection()
			} else if msg.Action == tea.MouseActionRelease {
				if !m.buffer.HasSelection() {
					m.buffer.ClearSelection()
				}
			}
		}
	}

	if m.mode == EditMode {
		contentWidth := m.width - 9
		if contentWidth < 10 {
			contentWidth = 10
		}
		cursorVisualY := 0
		for i, lineRunes := range m.buffer.lines {
			rows := len(lineRunes) / contentWidth
			if len(lineRunes)%contentWidth != 0 || len(lineRunes) == 0 {
				rows++
			}
			if i == m.buffer.cursorY {
				cx := m.buffer.cursorX
				if cx == len(lineRunes) && cx > 0 && cx%contentWidth == 0 {
					cursorVisualY += (cx / contentWidth) - 1
				} else {
					cursorVisualY += cx / contentWidth
				}
				break
			}
			cursorVisualY += rows
		}

		maxLines := m.height - 3
		if maxLines < 3 {
			maxLines = 3
		}
		if cursorVisualY < m.scrollOffset {
			m.scrollOffset = cursorVisualY
		} else if cursorVisualY >= m.scrollOffset+maxLines {
			m.scrollOffset = cursorVisualY - maxLines + 1
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) markDirty() {
	m.dirty = true
	m.lastEdit = time.Now()
}

func (m *Model) toggleFormat(prefix, suffix string) {
	if m.buffer.hasSelection {
		m.buffer.WrapOrUnwrapSelection(prefix, suffix)
	} else {
		m.buffer.WrapOrUnwrapWord(prefix, suffix)
	}
	m.markDirty()
}

func (m Model) View() string {
	if m.mode == PreviewMode {
		rendered := markup.RenderToTerminal(m.buffer.Text(), m.width, m.theme)
		lines := strings.Split(rendered, "\n")
		
		if m.scrollOffset > len(lines) {
			m.scrollOffset = len(lines) - 1
		}
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		
		end := m.scrollOffset + m.height
		if end > len(lines) {
			end = len(lines)
		}
		
		var view []string
		view = append(view, lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("PREVIEW MODE (Ctrl+R to edit)"))
		for i := m.scrollOffset; i < end && i < len(lines); i++ {
			view = append(view, lines[i])
		}
		return strings.Join(view, "\n")
	}

	// Edit Mode View
	var sb strings.Builder

	// Professional pitch black code editor styles
	lineNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5268")).Width(4).Align(lipgloss.Right)
	cursorLineNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623")).Bold(true).Width(4).Align(lipgloss.Right)
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2E3342"))
	
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EDE6D6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#606880"))
	selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color("#E58BB0")).Foreground(lipgloss.Color("#000000"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#F5A623")).Foreground(lipgloss.Color("#000000"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8")).Underline(true)
	
	maxLines := m.height - 3
	if maxLines < 3 {
		maxLines = 3
	}

	contentWidth := m.width - 9
	if contentWidth < 10 {
		contentWidth = 10
	}

	type vRow struct {
		bufIdx   int
		runes    []rune
		isCont   bool
		startCol int
	}
	var visualRows []vRow

	for i, lineRunes := range m.buffer.lines {
		if len(lineRunes) == 0 {
			visualRows = append(visualRows, vRow{bufIdx: i, runes: []rune{}, isCont: false, startCol: 0})
			continue
		}
		for start := 0; start < len(lineRunes); start += contentWidth {
			end := start + contentWidth
			if end > len(lineRunes) {
				end = len(lineRunes)
			}
			visualRows = append(visualRows, vRow{
				bufIdx:   i,
				runes:    lineRunes[start:end],
				isCont:   start > 0,
				startCol: start,
			})
		}
	}

	startRow := m.scrollOffset
	endRow := startRow + maxLines
	if endRow > len(visualRows) {
		endRow = len(visualRows)
	}

	for idx := startRow; idx < endRow; idx++ {
		vr := visualRows[idx]
		i := vr.bufIdx
		
		isCursorLine := (i == m.buffer.cursorY)
		
		if !vr.isCont {
			if isCursorLine {
				sb.WriteString(cursorLineNumberStyle.Render(fmt.Sprintf("%d", i+1)))
				sb.WriteString(dividerStyle.Render(" │ "))
			} else {
				sb.WriteString(lineNumberStyle.Render(fmt.Sprintf("%d", i+1)))
				sb.WriteString(dividerStyle.Render(" │ "))
			}
		} else {
			sb.WriteString(dimStyle.Render("   "))
			sb.WriteString(dividerStyle.Render(" · "))
		}

		// Highlight pasted URLs in bright blue with underline
		isURLMap := make(map[int]bool)
		lineRunesFull := m.buffer.lines[i]
		for pos := 0; pos < len(lineRunesFull); {
			subLower := strings.ToLower(string(lineRunesFull[pos:]))
			if strings.HasPrefix(subLower, "http://") || strings.HasPrefix(subLower, "https://") || strings.HasPrefix(subLower, "www.") {
				end := pos
				for end < len(lineRunesFull) && lineRunesFull[end] > ' ' && lineRunesFull[end] != ')' && lineRunesFull[end] != ']' && lineRunesFull[end] != '>' {
					end++
				}
				for p := pos; p < end; p++ {
					isURLMap[p] = true
				}
				pos = end
			} else {
				pos++
			}
		}

		var lineStr strings.Builder
		isLastRowOfLine := idx == len(visualRows)-1 || visualRows[idx+1].bufIdx != i
		
		renderLen := len(vr.runes)
		if isLastRowOfLine {
			renderLen++
		}

		for jOffset := 0; jOffset < renderLen; jOffset++ {
			j := vr.startCol + jOffset
			
			isSelected := false
			if m.buffer.hasSelection {
				sStart, sEnd := m.buffer.getSelectionRange()
				if i > sStart.Line && i < sEnd.Line {
					isSelected = true
				} else if i == sStart.Line && i == sEnd.Line {
					if j >= sStart.Col && j < sEnd.Col {
						isSelected = true
					}
				} else if i == sStart.Line {
					if j >= sStart.Col {
						isSelected = true
					}
				} else if i == sEnd.Line {
					if j < sEnd.Col {
						isSelected = true
					}
				}
			}

			isCursor := m.focused && i == m.buffer.cursorY && j == m.buffer.cursorX

			var r rune
			if jOffset < len(vr.runes) {
				r = vr.runes[jOffset]
			} else {
				r = ' '
				if !isCursor && !isSelected {
					continue
				}
			}

			charStr := string(r)
			if isCursor {
				charStr = cursorStyle.Render(charStr)
			} else if isSelected {
				charStr = selectionStyle.Render(charStr)
			} else if isURLMap[j] {
				charStr = urlStyle.Render(charStr)
			} else {
				if r == '*' || r == '_' || r == '~' || r == '=' || r == '^' || r == '#' || r == '`' || r == '>' || r == ':' {
					charStr = dimStyle.Render(charStr)
				} else {
					charStr = textStyle.Render(charStr)
				}
			}
			lineStr.WriteString(charStr)
		}
		
		sb.WriteString(lineStr.String())
		if idx < endRow-1 {
			sb.WriteString("\n")
		}
	}
	
	// Pad empty lines so the text editor area is fully expanded and the status bar is pinned at the bottom!
	linesRendered := endRow - startRow
	for i := linesRendered; i < maxLines; i++ {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(dividerStyle.Render("     │ "))
	}
	
	mainView := sb.String()
	
	if m.showFormatPane {
		pane := `┌─ Format ─────────┐
│ b  Bold      **  │
│ i  Italic    *   │
│ u  Underline __  │
│ s  Strike    ~~  │
│ h  Highlight ==  │
│ H  Sentence  ^^  │
│ 1  Heading 1 #   │
│ 2  Heading 2 ##  │
│ 3  Heading 3 ### │
│ q  Quote     >   │
│ m  Code      ` + "`" + `   │
│ p  Image    !img │
│ l  Left      ::: │
│ c  Center    ::: │
│ r  Right     ::: │
└──────────────────┘`
		paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#F5A623")).Padding(0, 1)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, mainView, paneStyle.Render(pane))
	}
	
	// Clean pitch-black status bar with statistics
	fullText := m.buffer.Text()
	wordCount := len(strings.Fields(fullText))
	charCount := len(fullText)
	lineCount := len(m.buffer.lines)

	saveStatus := "Saved ✓"
	if m.dirty {
		saveStatus = "Unsaved Changes •"
	}

	statusDivider := dividerStyle.Render(strings.Repeat("─", m.width))
	statusText := fmt.Sprintf("  %s  │  Ln %d, Col %d  │  Lines: %d  │  Words: %d  │  Chars: %d  │  %s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F5A623")).Render("EDIT"),
		m.buffer.cursorY+1, m.buffer.cursorX+1,
		lineCount, wordCount, charCount,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E58BB0")).Render(saveStatus),
	)

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusDivider, statusText)
}
