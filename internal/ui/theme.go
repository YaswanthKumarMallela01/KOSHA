package ui

import "github.com/charmbracelet/lipgloss"

// Colors - "Indigo Ink & Saffron" theme. NO GREEN ANYWHERE.
const (
	ColorBackground  = lipgloss.Color("#0E1022")
	ColorGlassFill   = lipgloss.Color("#1A1E3C")
	ColorGlassBorder = lipgloss.Color("#2B3166")
	ColorFrostEdge   = lipgloss.Color("#3A4180")
	ColorBodyText    = lipgloss.Color("#EDE6D6")
	ColorSaffron     = lipgloss.Color("#F2A33A")
	ColorLotusPink   = lipgloss.Color("#E58BB0")
	ColorMutedText   = lipgloss.Color("#7C82A8")
	ColorError       = lipgloss.Color("#E5646E")
	ColorShadow      = lipgloss.Color("#090A18")
	ColorGradientTop = lipgloss.Color("#0E1022")
	ColorGradientBot = lipgloss.Color("#161A38")
)

// Styles
var (
	GlassPanel = lipgloss.NewStyle().
			Background(ColorGlassFill).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Foreground(ColorBodyText).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorGlassFill).
			Foreground(ColorSaffron).
			Bold(true).
			Padding(0, 2)

	FooterStyle = lipgloss.NewStyle().
			Background(ColorGlassFill).
			Foreground(ColorMutedText).
			Padding(0, 2)

	ListItemStyle = lipgloss.NewStyle().
			Foreground(ColorBodyText).
			Padding(0, 2)

	SelectedItemStyle = lipgloss.NewStyle().
			Background(ColorLotusPink).
			Foreground(ColorBackground).
			Bold(true).
			Padding(0, 2)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMutedText)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	SaffronStyle = lipgloss.NewStyle().
			Foreground(ColorSaffron).
			Bold(true)

	PinkStyle = lipgloss.NewStyle().
			Foreground(ColorLotusPink)

	ShadowStyle = lipgloss.NewStyle().
			Foreground(ColorShadow)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(ColorMutedText)

	PinMarker = lipgloss.NewStyle().
			Foreground(ColorSaffron).
			SetString("📌 ")

	TagStyle = lipgloss.NewStyle().
			Foreground(ColorLotusPink)
)
