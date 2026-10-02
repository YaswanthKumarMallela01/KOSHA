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

func (b *Buffer) InsertRune(r rune) {
	b.pushUndo()
	if b.hasSelection {
		b.DeleteSelection()
	}
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

func (b *Buffer) InsertString(s string) {
	b.pushUndo()
	if b.hasSelection {
		b.DeleteSelection()
	}
	runes := []rune(s)
	for _, r := range runes {
		if r == '\r' {
			continue
		}
		b.InsertRune(r)
	}
}

func (b *Buffer) DeleteBack() {
	if b.hasSelection {
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
	if b.hasSelection {
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
	if !b.hasSelection {
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
		if b.cursorY > 0 && dx < 0 {
			b.cursorY--
			b.cursorX = len(b.lines[b.cursorY])
		} else {
			b.cursorX = 0
		}
	}
	if b.cursorX > len(b.lines[b.cursorY]) {
		if b.cursorY < len(b.lines)-1 && dx > 0 {
			b.cursorY++
			b.cursorX = 0
		} else {
			b.cursorX = len(b.lines[b.cursorY])
		}
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
	if !b.hasSelection {
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

func (b *Buffer) WrapOrUnwrapSelection(prefix, suffix string) {
	if !b.hasSelection {
		return
	}
	b.pushUndo()
	
	text := b.SelectedText()
	if strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix) {
		text = text[len(prefix) : len(text)-len(suffix)]
	} else {
		text = prefix + text + suffix
	}
	
	start, _ := b.getSelectionRange()
	b.DeleteSelection()
	
	b.cursorY = start.Line
	b.cursorX = start.Col
	b.InsertString(text)
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
		
		if m.showFormatPane && len(msg.String()) == 1 && m.mode == EditMode {
			k := msg.String()
			handled := true
			switch k {
			case "b":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("**", "**")
				} else {
					m.buffer.WrapOrUnwrapWord("**", "**")
				}
			case "i":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("*", "*")
				} else {
					m.buffer.WrapOrUnwrapWord("*", "*")
				}
			case "u":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("__", "__")
				} else {
					m.buffer.WrapOrUnwrapWord("__", "__")
				}
			case "s":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("~~", "~~")
				} else {
					m.buffer.WrapOrUnwrapWord("~~", "~~")
				}
			case "h":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("==", "==")
				} else {
					m.buffer.WrapOrUnwrapWord("==", "==")
				}
			case "H":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("^^", "^^")
				} else {
					m.buffer.WrapOrUnwrapWord("^^", "^^")
				}
			case "1":
				m.buffer.InsertString("# ")
			case "2":
				m.buffer.InsertString("## ")
			case "3":
				m.buffer.InsertString("### ")
			case "q":
				m.buffer.InsertString("> ")
			case "m":
				if m.buffer.hasSelection {
					m.buffer.WrapOrUnwrapSelection("`", "`")
				} else {
					m.buffer.WrapOrUnwrapWord("`", "`")
				}
			case "l":
				m.buffer.InsertString("\n:::left\n")
			case "c":
				m.buffer.InsertString("\n:::center\n")
			case "r":
				m.buffer.InsertString("\n:::right\n")
			case "esc", "ctrl+f":
				m.showFormatPane = false
			default:
				handled = false
			}
			
			if handled {
				m.dirty = true
				m.lastEdit = time.Now()
				m.showFormatPane = false
				return m, tea.Batch(cmds...)
			}
		}

		switch msg.Type {
		case tea.KeyCtrlR:
			if m.mode == EditMode {
				m.mode = PreviewMode
			} else {
				m.mode = EditMode
			}
		case tea.KeyCtrlF:
			m.showFormatPane = !m.showFormatPane
		case tea.KeyEsc:
			m.showFormatPane = false
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

		// Edit mode bindings
		switch msg.Type {
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
			m.buffer.InsertString("    ")
			m.markDirty()
		case tea.KeyCtrlZ:
			m.buffer.Undo()
			m.markDirty()
		case tea.KeyCtrlY:
			m.buffer.Redo()
			m.markDirty()
		case tea.KeyCtrlC:
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
			}
		case tea.KeyCtrlX:
			if m.buffer.hasSelection {
				_ = clipboard.WriteAll(m.buffer.SelectedText())
				m.buffer.DeleteSelection()
				m.markDirty()
			}
		case tea.KeyCtrlV:
			text, err := clipboard.ReadAll()
			if err == nil && text != "" {
				m.buffer.InsertString(text)
				m.markDirty()
			}
		case tea.KeyRunes, tea.KeySpace:
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
				m.buffer.InsertString(msg.String())
				m.markDirty()
			}
		}

		maxLines := m.height - 3
		if maxLines < 3 {
			maxLines = 3
		}
		if m.buffer.cursorY < m.scrollOffset {
			m.scrollOffset = m.buffer.cursorY
		} else if m.buffer.cursorY >= m.scrollOffset+maxLines {
			m.scrollOffset = m.buffer.cursorY - maxLines + 1
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
	
	maxLines := m.height - 3
	if maxLines < 3 {
		maxLines = 3
	}

	startLine := m.scrollOffset
	endLine := startLine + maxLines
	if endLine > len(m.buffer.lines) {
		endLine = len(m.buffer.lines)
	}

	for i := startLine; i < endLine; i++ {
		lineRunes := m.buffer.lines[i]
		
		isCursorLine := (i == m.buffer.cursorY)
		if isCursorLine {
			sb.WriteString(cursorLineNumberStyle.Render(fmt.Sprintf("%d", i+1)))
			sb.WriteString(dividerStyle.Render(" │ "))
		} else {
			sb.WriteString(lineNumberStyle.Render(fmt.Sprintf("%d", i+1)))
			sb.WriteString(dividerStyle.Render(" │ "))
		}

		var lineStr strings.Builder
		for j := 0; j <= len(lineRunes); j++ {
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
			if j < len(lineRunes) {
				r = lineRunes[j]
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
		if i < endLine-1 {
			sb.WriteString("\n")
		}
	}
	
	// Pad empty lines so the text editor area is fully expanded and the status bar is pinned at the bottom!
	linesRendered := endLine - startLine
	for i := linesRendered; i < maxLines; i++ {
		sb.WriteString("\n")
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
