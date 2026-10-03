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

// ExportPDF exports a note to a PDF file at the given path.
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

// renderBodyToPDF renders Kosha markup to PDF formatting.
func renderBodyToPDF(pdf *gofpdf.Fpdf, body string) {
	lines := strings.Split(body, "\n")
	align := "L"

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

		// Headings
		if strings.HasPrefix(line, "### ") {
			pdf.SetFont("Helvetica", "I", 13)
			pdf.SetTextColor(237, 230, 214) // Parchment
			renderInlineToPDF(pdf, line[4:], align)
			pdf.Ln(3)
			continue
		} else if strings.HasPrefix(line, "## ") {
			pdf.SetFont("Helvetica", "B", 15)
			pdf.SetTextColor(50, 50, 50)
			renderInlineToPDF(pdf, line[3:], align)
			pdf.Ln(3)
			continue
		} else if strings.HasPrefix(line, "# ") {
			pdf.SetFont("Helvetica", "B", 18)
			pdf.SetTextColor(242, 163, 58) // Saffron
			renderInlineToPDF(pdf, strings.ToUpper(line[2:]), align)
			pdf.Ln(4)
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			x := pdf.GetX()
			pdf.SetDrawColor(180, 180, 180)
			pdf.SetLineWidth(0.5)
			pdf.Line(x, pdf.GetY(), x, pdf.GetY()+5)
			pdf.SetX(x + 5)
			pdf.SetFont("Helvetica", "I", 11)
			pdf.SetTextColor(120, 120, 120)
			renderInlineToPDF(pdf, line[2:], align)
			pdf.Ln(2)
			continue
		}

		// Normal text
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetTextColor(30, 30, 30)
		renderInlineToPDF(pdf, line, align)
		pdf.Ln(1)
	}
}

// renderInlineToPDF renders inline markup to PDF.
func renderInlineToPDF(pdf *gofpdf.Fpdf, text string, align string) {
	// Strip markup for PDF (keep it simple and readable)
	plain := markup.StripMarkup(text)
	pdf.MultiCell(0, 6, plain, "", align, false)
}

// ExportDOCX exports a note to a DOCX file at the given path.
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
	body.WriteString(buildParagraph(note.Title, "Heading1", "center", true, false, false, "D4A22E", 36))

	// Meta
	metaStr := fmt.Sprintf("Created: %s  |  Updated: %s", note.CreatedAt.Format("2006-01-02 15:04"), note.UpdatedAt.Format("2006-01-02 15:04"))
	if len(note.Tags) > 0 {
		metaStr += "  |  Tags: " + strings.Join(note.Tags, " ")
	}
	body.WriteString(buildParagraph(metaStr, "", "left", false, true, false, "7C82A8", 18))

	// Divider (empty paragraph with bottom border)
	body.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="4" w:space="1" w:color="CCCCCC"/></w:pBdr></w:pPr></w:p>`)

	// Body
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

		plain := markup.StripMarkup(line)

		if strings.HasPrefix(line, "# ") {
			body.WriteString(buildParagraph(strings.ToUpper(plain), "Heading1", align, true, false, false, "D4A22E", 28))
		} else if strings.HasPrefix(line, "## ") {
			body.WriteString(buildParagraph(plain, "Heading2", align, true, false, false, "333333", 24))
		} else if strings.HasPrefix(line, "### ") {
			body.WriteString(buildParagraph(plain, "Heading3", align, false, true, false, "555555", 22))
		} else if strings.HasPrefix(line, "> ") {
			body.WriteString(buildParagraph(plain, "", align, false, true, false, "888888", 22))
		} else {
			body.WriteString(buildParagraph(plain, "", align, false, false, false, "1A1A1A", 22))
		}
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>%s</w:body>
</w:document>`, body.String())
}

func buildParagraph(text, style, align string, bold, italic, underline bool, color string, fontSize int) string {
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
