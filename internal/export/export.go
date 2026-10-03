package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

// NoteExport holds a note with its metadata for export.
type NoteExport struct {
	Title     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Tags      []string
}

// ExportPDF exports a note to a PDF file at the given path preserving full formatting and alignments.
func ExportPDF(note *NoteExport, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(242, 163, 58) // Saffron
	pdf.MultiCell(0, 10, note.Title, "", "L", false)
	pdf.Ln(3)

	// Meta line
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(124, 130, 168) // Muted
	metaStr := fmt.Sprintf("Created: %s  |  Updated: %s", note.CreatedAt.Format("2006-01-02 15:04"), note.UpdatedAt.Format("2006-01-02 15:04"))
	if len(note.Tags) > 0 {
		metaStr += "  |  Tags: " + strings.Join(note.Tags, " ")
	}
	pdf.MultiCell(0, 5, metaStr, "", "L", false)
	pdf.Ln(2)

	// Divider
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(5)

	// Body - parse and render to PDF
	renderBodyToPDF(pdf, note.Body)

	return pdf.OutputFileAndClose(outputPath)
}

// renderBodyToPDF renders Kosha markup to PDF formatting with full bold, italic, underline, links, and alignments.
func renderBodyToPDF(pdf *gofpdf.Fpdf, body string) {
	lines := strings.Split(body, "\n")
	align := "L"

	html := pdf.HTMLBasicNew()
	html.Link.ClrR = 2
	html.Link.ClrG = 132
	html.Link.ClrB = 199 // Electric blue
	html.Link.Underscore = true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Alignment directives
		if trimmed == ":::left" {
			align = "L"
			continue
		} else if trimmed == ":::center" {
			align = "C"
			continue
		} else if trimmed == ":::right" {
			align = "R"
			continue
		}

		// Empty line = paragraph break
		if trimmed == "" {
			pdf.Ln(4)
			continue
		}

		// Embedded image line: !img[alt](path|align|width%)
		if strings.HasPrefix(trimmed, "!img[") {
			imgInfo := markup.ParseImageLine(trimmed)
			if imgInfo != nil {
				renderImageToPDF(pdf, imgInfo)
				continue
			}
		}

		// Headings
		if strings.HasPrefix(line, "### ") {
			pdf.SetFont("Helvetica", "I", 13)
			pdf.SetTextColor(70, 70, 70)
			renderLineWithHTML(pdf, &html, line[4:], align, 6)
			pdf.Ln(3)
			continue
		} else if strings.HasPrefix(line, "## ") {
			pdf.SetFont("Helvetica", "B", 15)
			pdf.SetTextColor(40, 40, 40)
			renderLineWithHTML(pdf, &html, line[3:], align, 7)
			pdf.Ln(3)
			continue
		} else if strings.HasPrefix(line, "# ") {
			pdf.SetFont("Helvetica", "B", 18)
			pdf.SetTextColor(242, 163, 58) // Saffron
			renderLineWithHTML(pdf, &html, strings.ToUpper(line[2:]), align, 8)
			pdf.Ln(4)
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			x := pdf.GetX()
			pdf.SetDrawColor(242, 163, 58)
			pdf.SetLineWidth(0.8)
			pdf.Line(x, pdf.GetY(), x, pdf.GetY()+6)
			pdf.SetX(x + 5)
			pdf.SetFont("Helvetica", "I", 11)
			pdf.SetTextColor(110, 110, 110)
			renderLineWithHTML(pdf, &html, line[2:], align, 6)
			pdf.Ln(2)
			continue
		}

		// Normal text
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetTextColor(30, 30, 30)
		renderLineWithHTML(pdf, &html, line, align, 6)
	}
}

func renderLineWithHTML(pdf *gofpdf.Fpdf, html *gofpdf.HTMLBasicType, lineText string, align string, lineHt float64) {
	nodes := markup.ParseInline(lineText)
	htmlStr := markup.RenderNodesToHTML(nodes)
	plainText := markup.StripMarkup(lineText)

	pageW, _ := pdf.GetPageSize()
	marginL, _, marginR, _ := pdf.GetMargins()
	contentW := pageW - marginL - marginR

	strW := pdf.GetStringWidth(plainText)

	x := marginL
	if align == "C" && strW < contentW {
		x = marginL + (contentW-strW)/2
	} else if align == "R" && strW < contentW {
		x = marginL + (contentW - strW)
	}

	pdf.SetX(x)
	html.Write(lineHt, htmlStr)
	pdf.Ln(lineHt)
}

func renderImageToPDF(pdf *gofpdf.Fpdf, info *markup.ImageInfo) {
	pageW, _ := pdf.GetPageSize()
	marginL, _, marginR, _ := pdf.GetMargins()
	contentW := pageW - marginL - marginR

	// Check if file exists
	if _, err := os.Stat(info.Path); err == nil {
		targetW := contentW * 0.7
		if info.Width == "100%" {
			targetW = contentW
		} else if info.Width == "50%" {
			targetW = contentW * 0.5
		}

		x := marginL
		if info.Align == "center" {
			x = marginL + (contentW-targetW)/2
		} else if info.Align == "right" {
			x = marginL + (contentW - targetW)
		}

		pdf.SetX(x)
		opt := gofpdf.ImageOptions{ReadDpi: true}
		pdf.ImageOptions(info.Path, x, pdf.GetY(), targetW, 0, false, opt, 0, "")
		pdf.Ln(4)
		return
	}

	// Fallback styled placeholder box
	pdf.SetFont("Helvetica", "I", 10)
	pdf.SetTextColor(2, 132, 199)
	pdf.SetDrawColor(200, 200, 200)
	msg := fmt.Sprintf("[Image: %s | %s | %s]", info.Alt, filepath.Base(info.Path), info.Width)
	pdf.CellFormat(0, 8, msg, "1", 1, "C", false, 0, "")
	pdf.Ln(3)
}

// ExportDOCX exports a note to a Word .docx file preserving all inline bold, italic, underline, strike, highlights, links, and alignments.
func ExportDOCX(note *NoteExport, outputPath string) error {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	writeZipFile(w, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	writeZipFile(w, "_rels/.rels", rels)

	// word/_rels/document.xml.rels
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
	writeZipFile(w, "word/_rels/document.xml.rels", docRels)

	// word/document.xml
	docXML := buildDocumentXML(note)
	writeZipFile(w, "word/document.xml", docXML)

	if err := w.Close(); err != nil {
		return fmt.Errorf("closing docx zip: %w", err)
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0600)
}

func writeZipFile(w *zip.Writer, name, content string) {
	f, err := w.Create(name)
	if err != nil {
		return
	}
	_, _ = f.Write([]byte(content))
}

func buildDocumentXML(note *NoteExport) string {
	var body strings.Builder

	// Title paragraph
	body.WriteString(buildSimpleParagraph(note.Title, "Heading1", "center", true, false, false, "F2A33A", 36))

	// Meta
	metaStr := fmt.Sprintf("Created: %s  |  Updated: %s", note.CreatedAt.Format("2006-01-02 15:04"), note.UpdatedAt.Format("2006-01-02 15:04"))
	if len(note.Tags) > 0 {
		metaStr += "  |  Tags: " + strings.Join(note.Tags, " ")
	}
	body.WriteString(buildSimpleParagraph(metaStr, "", "left", false, true, false, "7C82A8", 18))

	// Divider
	body.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="E2E8F0"/></w:pBdr></w:pPr></w:p>`)

	// Body lines
	lines := strings.Split(note.Body, "\n")
	align := "left"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == ":::left" {
			align = "left"
			continue
		} else if trimmed == ":::center" {
			align = "center"
			continue
		} else if trimmed == ":::right" {
			align = "right"
			continue
		}

		if trimmed == "" {
			body.WriteString(`<w:p/>`)
			continue
		}

		// Image line
		if strings.HasPrefix(trimmed, "!img[") {
			imgInfo := markup.ParseImageLine(trimmed)
			if imgInfo != nil {
				imgText := fmt.Sprintf("[🖼  Image: %s | %s | %s]", imgInfo.Alt, filepath.Base(imgInfo.Path), imgInfo.Width)
				body.WriteString(buildSimpleParagraph(imgText, "", imgInfo.Align, true, true, false, "0284C7", 20))
				continue
			}
		}

		// Headings
		if strings.HasPrefix(line, "# ") {
			nodes := markup.ParseInline(strings.ToUpper(line[2:]))
			body.WriteString(buildFormattedParagraph(nodes, "Heading1", align, 28, "F2A33A", true, false))
		} else if strings.HasPrefix(line, "## ") {
			nodes := markup.ParseInline(line[3:])
			body.WriteString(buildFormattedParagraph(nodes, "Heading2", align, 24, "1E293B", true, false))
		} else if strings.HasPrefix(line, "### ") {
			nodes := markup.ParseInline(line[4:])
			body.WriteString(buildFormattedParagraph(nodes, "Heading3", align, 22, "475569", false, true))
		} else if strings.HasPrefix(line, "> ") {
			nodes := markup.ParseInline(line[2:])
			body.WriteString(buildFormattedParagraph(nodes, "", align, 22, "64748B", false, true))
		} else {
			nodes := markup.ParseInline(line)
			body.WriteString(buildFormattedParagraph(nodes, "", align, 22, "0F172A", false, false))
		}
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>%s</w:body>
</w:document>`, body.String())
}

// buildFormattedParagraph creates a Word paragraph containing multiple runs with exact inline formatting.
func buildFormattedParagraph(nodes []*markup.Node, style, align string, fontSize int, defaultColor string, defaultBold, defaultItalic bool) string {
	var pPr strings.Builder
	pPr.WriteString("<w:pPr>")
	if style != "" {
		pPr.WriteString(fmt.Sprintf(`<w:pStyle w:val="%s"/>`, style))
	}
	jcMap := map[string]string{"left": "left", "center": "center", "right": "right"}
	if jc, ok := jcMap[align]; ok {
		pPr.WriteString(fmt.Sprintf(`<w:jc w:val="%s"/>`, jc))
	}
	pPr.WriteString("</w:pPr>")

	var runs strings.Builder
	renderNodesToWordRuns(&runs, nodes, defaultBold, defaultItalic, false, false, false, defaultColor, fontSize)

	return fmt.Sprintf(`<w:p>%s%s</w:p>`, pPr.String(), runs.String())
}

// renderNodesToWordRuns recursively generates Word <w:r> runs for each inline formatting node.
func renderNodesToWordRuns(sb *strings.Builder, nodes []*markup.Node, bold, italic, underline, strike, highlight bool, color string, fontSize int) {
	for _, n := range nodes {
		switch n.Type {
		case "text":
			writeWordRun(sb, n.Content, bold, italic, underline, strike, highlight, false, color, fontSize)
		case "bold":
			renderNodesToWordRuns(sb, n.Children, true, italic, underline, strike, highlight, color, fontSize)
		case "italic":
			renderNodesToWordRuns(sb, n.Children, bold, true, underline, strike, highlight, color, fontSize)
		case "underline":
			renderNodesToWordRuns(sb, n.Children, bold, italic, true, strike, highlight, color, fontSize)
		case "strike":
			renderNodesToWordRuns(sb, n.Children, bold, italic, underline, true, highlight, color, fontSize)
		case "imp_word", "imp_sent":
			renderNodesToWordRuns(sb, n.Children, true, italic, true, strike, true, "D97706", fontSize)
		case "code":
			writeWordRun(sb, n.Content, false, false, false, false, false, true, "0F172A", fontSize)
		case "link":
			writeWordRun(sb, "[["+n.Content+"]]", false, false, true, false, false, false, "0284C7", fontSize)
		case "url":
			writeWordRun(sb, n.Content, false, false, true, false, false, false, "0284C7", fontSize)
		case "tag":
			writeWordRun(sb, "#"+n.Content, true, false, false, false, false, false, "DB2777", fontSize)
		}
	}
}

func writeWordRun(sb *strings.Builder, text string, bold, italic, underline, strike, highlight, isCode bool, color string, fontSize int) {
	var rPr strings.Builder
	rPr.WriteString("<w:rPr>")

	if bold {
		rPr.WriteString("<w:b/>")
	}
	if italic {
		rPr.WriteString("<w:i/>")
	}
	if underline {
		rPr.WriteString(`<w:u w:val="single"/>`)
	}
	if strike {
		rPr.WriteString("<w:strike/>")
	}
	if highlight {
		rPr.WriteString(`<w:highlight w:val="yellow"/>`)
	}
	if isCode {
		rPr.WriteString(`<w:rFonts w:ascii="Consolas" w:hAnsi="Consolas"/>`)
		rPr.WriteString(`<w:shd w:fill="F1F5F9"/>`)
	}
	if color != "" {
		rPr.WriteString(fmt.Sprintf(`<w:color w:val="%s"/>`, color))
	}
	if fontSize > 0 {
		rPr.WriteString(fmt.Sprintf(`<w:sz w:val="%d"/>`, fontSize))
	}

	rPr.WriteString("</w:rPr>")
	escapedText := xmlEscape(text)
	sb.WriteString(fmt.Sprintf(`<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r>`, rPr.String(), escapedText))
}

func buildSimpleParagraph(text, style, align string, bold, italic, underline bool, color string, fontSize int) string {
	var pPr strings.Builder
	pPr.WriteString("<w:pPr>")
	if style != "" {
		pPr.WriteString(fmt.Sprintf(`<w:pStyle w:val="%s"/>`, style))
	}
	jcMap := map[string]string{"left": "left", "center": "center", "right": "right"}
	if jc, ok := jcMap[align]; ok {
		pPr.WriteString(fmt.Sprintf(`<w:jc w:val="%s"/>`, jc))
	}
	pPr.WriteString("</w:pPr>")

	var rPr strings.Builder
	rPr.WriteString("<w:rPr>")
	if bold {
		rPr.WriteString("<w:b/>")
	}
	if italic {
		rPr.WriteString("<w:i/>")
	}
	if underline {
		rPr.WriteString(`<w:u w:val="single"/>`)
	}
	if color != "" {
		rPr.WriteString(fmt.Sprintf(`<w:color w:val="%s"/>`, color))
	}
	if fontSize > 0 {
		rPr.WriteString(fmt.Sprintf(`<w:sz w:val="%d"/>`, fontSize))
	}
	rPr.WriteString("</w:rPr>")

	escapedText := xmlEscape(text)
	return fmt.Sprintf(`<w:p>%s<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p>`, pPr.String(), rPr.String(), escapedText)
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// ExportNoteToPDF exports a single note model to PDF.
func ExportNoteToPDF(note *model.Note, outputPath string) error {
	ne := &NoteExport{
		Title:     note.Title,
		Body:      note.Body,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
		Tags:      model.ExtractTags(note.Body),
	}
	return ExportPDF(ne, outputPath)
}

// ExportNoteToDOCX exports a single note model to DOCX.
func ExportNoteToDOCX(note *model.Note, outputPath string) error {
	ne := &NoteExport{
		Title:     note.Title,
		Body:      note.Body,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
		Tags:      model.ExtractTags(note.Body),
	}
	return ExportDOCX(ne, outputPath)
}

// ExportChapterToPDF exports all notes in a chapter to a single PDF.
func ExportChapterToPDF(chapterTitle string, notes []*model.Note, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetMargins(20, 20, 20)

	for _, note := range notes {
		pdf.AddPage()

		// Title
		pdf.SetFont("Helvetica", "B", 18)
		pdf.SetTextColor(242, 163, 58)
		pdf.MultiCell(0, 10, note.Title, "", "L", false)
		pdf.Ln(2)

		// Meta
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(124, 130, 168)
		tags := model.ExtractTags(note.Body)
		metaStr := fmt.Sprintf("Created: %s  |  Updated: %s", note.CreatedAt.Format("2006-01-02 15:04"), note.UpdatedAt.Format("2006-01-02 15:04"))
		if len(tags) > 0 {
			metaStr += "  |  Tags: " + strings.Join(tags, " ")
		}
		pdf.MultiCell(0, 5, metaStr, "", "L", false)
		pdf.Ln(2)

		pdf.SetDrawColor(200, 200, 200)
		pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
		pdf.Ln(5)

		renderBodyToPDF(pdf, note.Body)
	}

	return pdf.OutputFileAndClose(outputPath)
}

// GetExportDir returns the export directory in the book folder.
func GetExportDir(bookDir string) string {
	dir := filepath.Join(bookDir, "exports")
	_ = os.MkdirAll(dir, 0700)
	return dir
}
