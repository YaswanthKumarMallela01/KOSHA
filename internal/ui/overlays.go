package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputOverlayModel struct {
	Prompt   string
	Action   string // "new_book", "new_chapter", "new_note", "rename"
	TargetID string
	Input    textinput.Model
	Active   bool
	w, h     int
}

func NewInputOverlayModel() InputOverlayModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	return InputOverlayModel{
		Input: ti,
	}
}

func (m *InputOverlayModel) Open(prompt, defaultValue, action, targetID string) {
	m.Prompt = prompt
	m.Action = action
	m.TargetID = targetID
	m.Input.SetValue(defaultValue)
	m.Input.Focus()
	m.Active = true
}

func (m *InputOverlayModel) Close() {
	m.Active = false
	m.Input.Reset()
}

func (m InputOverlayModel) Update(msg tea.Msg) (InputOverlayModel, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m InputOverlayModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(1, 3).
		Width(50)

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		SaffronStyle.Render(m.Prompt),
		m.Input.View(),
		MutedStyle.Render("[Enter] Confirm  [Esc] Cancel"),
	)

	return boxStyle.Render(content)
}

type ConfirmDialogModel struct {
	Message       string
	Action        string // "delete_book", "delete_chapter", "delete_note"
	TargetID      string
	Active        bool
	PasswordInput textinput.Model
}

func NewConfirmDialogModel() ConfirmDialogModel {
	ti := textinput.New()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Prompt = "  > "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorSaffron).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorBodyText)
	ti.Placeholder = "Enter master passphrase..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorMutedText)
	ti.CharLimit = 100
	ti.Width = 36
	return ConfirmDialogModel{
		PasswordInput: ti,
	}
}

func (m *ConfirmDialogModel) Open(message, action, targetID string) {
	m.Message = message
	m.Action = action
	m.TargetID = targetID
	m.Active = true
	m.PasswordInput.Reset()
	m.PasswordInput.Focus()
}

func (m *ConfirmDialogModel) Close() {
	m.Active = false
	m.PasswordInput.Reset()
}

func (m ConfirmDialogModel) Update(msg tea.Msg) (ConfirmDialogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.PasswordInput, cmd = m.PasswordInput.Update(msg)
	return m, cmd
}

func (m ConfirmDialogModel) View() string {
	borderColor := ColorError
	titleStyle := ErrorStyle
	headerText := "⚠ CONFIRM DELETION (PASSPHRASE REQUIRED)"
	btnText := "Confirm Delete"

	if strings.HasPrefix(m.Action, "export") {
		borderColor = ColorSaffron
		titleStyle = SaffronStyle
		if m.Action == "export_pdf" {
			headerText = "🔐 AUTHENTICATE EXPORT — PDF DOCUMENT"
			btnText = "Export PDF"
		} else if m.Action == "export_docx" {
			headerText = "🔐 AUTHENTICATE EXPORT — WORD (.DOCX)"
			btnText = "Export Word"
		} else {
			headerText = "🔐 AUTHENTICATE EXPORT (PASSPHRASE REQUIRED)"
			btnText = "Confirm Export"
		}
	}

	boxStyle := lipgloss.NewStyle().
		Background(ColorBackground).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 4).
		Width(60).
		Align(lipgloss.Center)

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		titleStyle.Render(headerText),
		lipgloss.NewStyle().Foreground(ColorBodyText).Render(m.Message),
		m.PasswordInput.View(),
		MutedStyle.Render(fmt.Sprintf("[Enter] %s  •  [Esc] Cancel", btnText)),
	)

	return boxStyle.Render(content)
}
