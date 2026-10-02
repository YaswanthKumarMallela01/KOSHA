package markup

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the styling colors and styles for Kosha markup.
type Theme struct {
	Background     lipgloss.Color
	GlassFill      lipgloss.Color
	GlassBorder    lipgloss.Color
	FrostEdge      lipgloss.Color
	BodyText       lipgloss.Color
	Saffron        lipgloss.Color
	LotusPink      lipgloss.Color
	MutedText      lipgloss.Color
	ErrorColor     lipgloss.Color
	Shadow         lipgloss.Color
	GradientTop    lipgloss.Color
	GradientBottom lipgloss.Color

	Heading1          lipgloss.Style
	Heading2          lipgloss.Style
	Heading3          lipgloss.Style
	Bold              lipgloss.Style
	Italic            lipgloss.Style
	Underline         lipgloss.Style
	Strikethrough     lipgloss.Style
	ImportantWord     lipgloss.Style
	ImportantSentence lipgloss.Style
	Blockquote        lipgloss.Style
	InlineCode        lipgloss.Style
	Tag               lipgloss.Style
	Link              lipgloss.Style
}

// NewDefaultTheme creates a theme with the default colors.
func NewDefaultTheme() *Theme {
	t := &Theme{
		Background:     lipgloss.Color("#0E1022"),
		GlassFill:      lipgloss.Color("#1A1E3C"),
		GlassBorder:    lipgloss.Color("#2B3166"),
		FrostEdge:      lipgloss.Color("#3A4180"),
		BodyText:       lipgloss.Color("#EDE6D6"),
		Saffron:        lipgloss.Color("#F2A33A"),
		LotusPink:      lipgloss.Color("#E58BB0"),
		MutedText:      lipgloss.Color("#7C82A8"),
		ErrorColor:     lipgloss.Color("#E5646E"),
		Shadow:         lipgloss.Color("#090A18"),
		GradientTop:    lipgloss.Color("#0E1022"),
		GradientBottom: lipgloss.Color("#161A38"),
	}

	t.Heading1 = lipgloss.NewStyle().Bold(true).Foreground(t.Saffron)
	t.Heading2 = lipgloss.NewStyle().Bold(true).Foreground(t.BodyText)
	t.Heading3 = lipgloss.NewStyle().Italic(true).Foreground(t.BodyText)
	t.Bold = lipgloss.NewStyle().Bold(true)
	t.Italic = lipgloss.NewStyle().Italic(true)
	t.Underline = lipgloss.NewStyle().Underline(true)
	t.Strikethrough = lipgloss.NewStyle().Strikethrough(true)
	t.ImportantWord = lipgloss.NewStyle().Foreground(t.Saffron).Bold(true)
	t.ImportantSentence = lipgloss.NewStyle().Foreground(t.Saffron).Underline(true)
	t.Blockquote = lipgloss.NewStyle().Foreground(t.MutedText)
	t.InlineCode = lipgloss.NewStyle().Background(t.GlassFill).Foreground(t.BodyText)
	t.Tag = lipgloss.NewStyle().Foreground(t.LotusPink)
	t.Link = lipgloss.NewStyle().Foreground(t.LotusPink).Underline(true)

	return t
}

type Node struct {
	Type     string
	Content  string
	Children []*Node
}

func parseInline(text string) []*Node {
	var nodes []*Node
	runes := []rune(text)
	length := len(runes)

	i := 0
	for i < length {
		if runes[i] == '\\' && i+1 < length {
			nodes = append(nodes, &Node{Type: "text", Content: string(runes[i+1])})
			i += 2
			continue
		}

		if i+1 < length && string(runes[i:i+2]) == "**" {
			end := findEnd(runes, i+2, "**")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "bold", Children: parseInline(string(runes[i+2 : end]))})
				i = end + 2
				continue
			}
		}

		if i+1 < length && string(runes[i:i+2]) == "__" {
			end := findEnd(runes, i+2, "__")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "underline", Children: parseInline(string(runes[i+2 : end]))})
				i = end + 2
				continue
			}
		}

		if i+1 < length && string(runes[i:i+2]) == "~~" {
			end := findEnd(runes, i+2, "~~")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "strike", Children: parseInline(string(runes[i+2 : end]))})
				i = end + 2
				continue
			}
		}

		if i+1 < length && string(runes[i:i+2]) == "==" {
			end := findEnd(runes, i+2, "==")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "imp_word", Children: parseInline(string(runes[i+2 : end]))})
				i = end + 2
				continue
			}
		}

		if i+1 < length && string(runes[i:i+2]) == "^^" {
			end := findEnd(runes, i+2, "^^")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "imp_sent", Children: parseInline(string(runes[i+2 : end]))})
				i = end + 2
				continue
			}
		}

		if i+1 < length && string(runes[i:i+2]) == "[[" {
			end := findEnd(runes, i+2, "]]")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "link", Content: string(runes[i+2 : end])})
				i = end + 2
				continue
			}
		}

		if string(runes[i]) == "`" {
			end := findEnd(runes, i+1, "`")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "code", Content: string(runes[i+1 : end])})
				i = end + 1
				continue
			}
		}

		if string(runes[i]) == "*" {
			end := findEnd(runes, i+1, "*")
			if end != -1 {
				nodes = append(nodes, &Node{Type: "italic", Children: parseInline(string(runes[i+1 : end]))})
				i = end + 1
				continue
			}
		}

		if string(runes[i]) == "#" && (i == 0 || runes[i-1] == ' ' || runes[i-1] == '\t') {
			end := i + 1
			for end < length && (isAlphaNumeric(runes[end]) || runes[end] == '-' || runes[end] == '_') {
				end++
			}
			if end > i+1 {
				nodes = append(nodes, &Node{Type: "tag", Content: string(runes[i+1 : end])})
				i = end
				continue
			}
		}

		// Text
		end := i + 1
		for end < length && runes[end] != '\\' && runes[end] != '*' && runes[end] != '_' && runes[end] != '~' && runes[end] != '=' && runes[end] != '^' && runes[end] != '[' && runes[end] != '`' && runes[end] != '#' {
			end++
		}
		nodes = append(nodes, &Node{Type: "text", Content: string(runes[i:end])})
		i = end
	}

	return nodes
}

func findEnd(runes []rune, start int, marker string) int {
	markerRunes := []rune(marker)
	for i := start; i <= len(runes)-len(markerRunes); i++ {
		if runes[i-1] == '\\' {
			continue
		}
		match := true
		for j := 0; j < len(markerRunes); j++ {
			if runes[i+j] != markerRunes[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// RenderToTerminal parses the markup and renders to ANSI-styled string.
func RenderToTerminal(input string, width int, theme *Theme) string {
	if theme == nil {
		theme = NewDefaultTheme()
	}

	lines := strings.Split(input, "\n")
	var out []string
	align := "left"

	for _, line := range lines {
		if strings.TrimSpace(line) == ":::left" {
			align = "left"
			continue
		} else if strings.TrimSpace(line) == ":::center" {
			align = "center"
			continue
		} else if strings.TrimSpace(line) == ":::right" {
			align = "right"
			continue
		}

		var renderedLine string
		var style lipgloss.Style
		var prefix string

		if strings.HasPrefix(line, "### ") {
			renderedLine = renderNodes(parseInline(line[4:]), theme)
			style = theme.Heading3
			renderedLine = style.Render("### " + renderedLine)
		} else if strings.HasPrefix(line, "## ") {
			renderedLine = renderNodes(parseInline(line[3:]), theme)
			style = theme.Heading2
			renderedLine = style.Render("## " + renderedLine)
		} else if strings.HasPrefix(line, "# ") {
			renderedLine = renderNodes(parseInline(line[2:]), theme)
			style = theme.Heading1
			renderedLine = style.Render("# " + strings.ToUpper(renderedLine))
		} else if strings.HasPrefix(line, "> ") {
			renderedLine = renderNodes(parseInline(line[2:]), theme)
			prefix = lipgloss.NewStyle().Foreground(theme.MutedText).Render("▍ ")
			renderedLine = prefix + theme.Blockquote.Render(renderedLine)
		} else {
			renderedLine = renderNodes(parseInline(line), theme)
		}

		// Handle alignment
		if align == "center" && width > 0 {
			renderedLine = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(renderedLine)
		} else if align == "right" && width > 0 {
			renderedLine = lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(renderedLine)
		} else if align == "left" && width > 0 {
			renderedLine = lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(renderedLine)
		}

		out = append(out, renderedLine)
	}

	return strings.Join(out, "\n")
}

func renderNodes(nodes []*Node, theme *Theme) string {
	var sb strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			sb.WriteString(n.Content)
		case "bold":
			sb.WriteString(theme.Bold.Render(renderNodes(n.Children, theme)))
		case "italic":
			sb.WriteString(theme.Italic.Render(renderNodes(n.Children, theme)))
		case "underline":
			sb.WriteString(theme.Underline.Render(renderNodes(n.Children, theme)))
		case "strike":
			sb.WriteString(theme.Strikethrough.Render(renderNodes(n.Children, theme)))
		case "imp_word":
			sb.WriteString(theme.ImportantWord.Render(renderNodes(n.Children, theme)))
		case "imp_sent":
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Saffron).Render("▍ ") + theme.ImportantSentence.Render(renderNodes(n.Children, theme)))
		case "link":
			sb.WriteString(theme.Link.Render(n.Content))
		case "code":
			sb.WriteString(theme.InlineCode.Render(n.Content))
		case "tag":
			sb.WriteString(theme.Tag.Render("#" + n.Content))
		}
	}
	return sb.String()
}

// RenderToMarkdown converts Kosha markup to standard Markdown.
func RenderToMarkdown(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	align := "left"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ":::left" {
			if align != "left" {
				out = append(out, "</div>")
				align = "left"
			}
			continue
		} else if trimmed == ":::center" {
			if align != "left" {
				out = append(out, "</div>")
			}
			out = append(out, "<div align=\"center\">")
			align = "center"
			continue
		} else if trimmed == ":::right" {
			if align != "left" {
				out = append(out, "</div>")
			}
			out = append(out, "<div align=\"right\">")
			align = "right"
			continue
		}

		renderedLine := renderNodesMD(parseInline(line))
		out = append(out, renderedLine)
	}

	if align != "left" {
		out = append(out, "</div>")
	}

	return strings.Join(out, "\n")
}

func renderNodesMD(nodes []*Node) string {
	var sb strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			sb.WriteString(n.Content)
		case "bold":
			sb.WriteString("**" + renderNodesMD(n.Children) + "**")
		case "italic":
			sb.WriteString("*" + renderNodesMD(n.Children) + "*")
		case "underline":
			sb.WriteString("<u>" + renderNodesMD(n.Children) + "</u>")
		case "strike":
			sb.WriteString("~~" + renderNodesMD(n.Children) + "~~")
		case "imp_word":
			sb.WriteString("**" + renderNodesMD(n.Children) + "**")
		case "imp_sent":
			sb.WriteString("**" + renderNodesMD(n.Children) + "**")
		case "link":
			sb.WriteString("[" + n.Content + "]()")
		case "code":
			sb.WriteString("`" + n.Content + "`")
		case "tag":
			sb.WriteString("#" + n.Content)
		}
	}
	return sb.String()
}

// StripMarkup removes all markup tokens and returns plain text.
func StripMarkup(input string) string {
	// Simple approach: parse lines and strip nodes, excluding directives
	lines := strings.Split(input, "\n")
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ":::left" || trimmed == ":::center" || trimmed == ":::right" {
			continue
		}

		if strings.HasPrefix(line, "### ") {
			line = line[4:]
		} else if strings.HasPrefix(line, "## ") {
			line = line[3:]
		} else if strings.HasPrefix(line, "# ") {
			line = line[2:]
		} else if strings.HasPrefix(line, "> ") {
			line = line[2:]
		}

		out = append(out, renderNodesStrip(parseInline(line)))
	}

	return strings.Join(out, "\n")
}

func renderNodesStrip(nodes []*Node) string {
	var sb strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			sb.WriteString(n.Content)
		case "bold", "italic", "underline", "strike", "imp_word", "imp_sent":
			sb.WriteString(renderNodesStrip(n.Children))
		case "link", "code", "tag":
			sb.WriteString(n.Content)
		}
	}
	return sb.String()
}
