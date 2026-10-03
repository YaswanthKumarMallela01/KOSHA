package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
)

type PassphraseModel struct {
	app      *App
	input    textinput.Model
	attempts int
	errMsg   string
}

func NewPassphraseModel() PassphraseModel {
	ti := textinput.New()
	ti.Placeholder = "Enter master passphrase..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorMutedText)
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Prompt = "  > "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorSaffron).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorBodyText)
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 36

	return PassphraseModel{
		input: ti,
	}
}

func (m *PassphraseModel) Reset() {
	m.input.Reset()
	m.input.Focus()
	m.errMsg = ""
}

func (m PassphraseModel) Update(msg tea.Msg) (PassphraseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			pass := m.input.Value()
			if pass == "" {
				return m, nil
			}

			if m.app == nil || m.app.config == nil {
				m.errMsg = "Configuration missing"
				return m, nil
			}

			// Validate passphrase
			masterKey := crypto.DeriveMasterKey([]byte(pass), []byte(m.app.config.MasterSalt), m.app.config.ArgonParams)
			if crypto.ValidateKeyCheck([]byte(m.app.config.KeyCheck), masterKey) {
				// Success!
				m.app.library.MasterKey = masterKey
				_ = m.app.library.LoadBooks()
				m.app.buildSearchIndex()
				m.app.locked = false
				m.app.screen = ScreenLibrary
				m.app.libraryList.Refresh()
				m.Reset()
				m.attempts = 0
				m.app.statusMsg = "Vault unlocked"
				return m, tea.ClearScreen
			}

			// Failed
			crypto.ZeroBytes(masterKey)
			m.attempts++
			remaining := 5 - m.attempts
			if remaining <= 0 {
				m.errMsg = "Too many failed attempts. Vault locked."
			} else {
				m.errMsg = fmt.Sprintf("Incorrect passphrase. %d attempt(s) remaining.", remaining)
			}
			m.input.Reset()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m PassphraseModel) View(width, height int) string {
	if width < 50 {
		width = 50
	}
	if height < 15 {
		height = 15
	}

	boxWidth := 56
	boxStyle := lipgloss.NewStyle().
		Background(ColorBackground).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(1, 4).
		Width(boxWidth).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorSaffron)
	subStyle := lipgloss.NewStyle().Foreground(ColorMutedText)

	logo := titleStyle.Render("⚡   K O S H A   ⚡") + "\n" + subStyle.Render("[ Encrypted Notes Vault ]")

	errLine := ""
	if m.errMsg != "" {
		errLine = "\n" + ErrorStyle.Render(m.errMsg)
	}

	promptText := lipgloss.NewStyle().Foreground(ColorBodyText).Render("Vault is locked. Enter your master passphrase:")
	hintText := MutedStyle.Render("[Enter] Unlock   •   [Ctrl+C] Exit")

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s%s",
		logo,
		promptText,
		m.input.View(),
		hintText,
		errLine,
	)

	renderedBox := boxStyle.Render(content)
	boxLines := strings.Split(renderedBox, "\n")
	boxH := len(boxLines)
	boxW := boxWidth + 2 // including border

	topPad := (height - boxH) / 2
	if topPad < 0 {
		topPad = 0
	}

	leftPad := (width - boxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	blankRow := strings.Repeat(" ", width)
	bgStyle := lipgloss.NewStyle().Background(ColorBackground).Foreground(ColorBodyText)

	var fullScreen []string

	// 1. Top padding rows (full width pitch black)
	for i := 0; i < topPad; i++ {
		fullScreen = append(fullScreen, bgStyle.Render(blankRow))
	}

	// 2. Box rows (padded on left and right to exact width with explicit pitch black background)
	for _, bLine := range boxLines {
		bLen := lipgloss.Width(bLine)
		rightPad := width - leftPad - bLen
		if rightPad < 0 {
			rightPad = 0
		}
		leftPadStr := bgStyle.Render(strings.Repeat(" ", leftPad))
		rightPadStr := bgStyle.Render(strings.Repeat(" ", rightPad))
		fullScreen = append(fullScreen, leftPadStr+bLine+rightPadStr)
	}

	// 3. Bottom padding rows down to exact height
	for len(fullScreen) < height {
		fullScreen = append(fullScreen, bgStyle.Render(blankRow))
	}
	if len(fullScreen) > height {
		fullScreen = fullScreen[:height]
	}

	return strings.Join(fullScreen, "\n")
}
