package templating

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// DocxEngine interpolates {{key}} placeholders and dynamic table rows inside .docx OpenXML archives.
type DocxEngine struct{}

// NewDocxEngine constructs a new DocxEngine instance.
func NewDocxEngine() *DocxEngine {
	return &DocxEngine{}
}

const (
	maxDocxZipEntries = 256
	maxDocxEntryBytes = 16 << 20
	maxDocxTotalBytes = 32 << 20
)

var (
	splitTagBetweenBracesRe = regexp.MustCompile(`\{\{([^{}]*?)\}\}`)
	xmlTagInsideBracesRe    = regexp.MustCompile(`<[^>]+>`)
	tableRowRe              = regexp.MustCompile(`(?s)<w:tr\b[^>]*>.*?</w:tr>`)
)

// RenderDocx reads a .docx ZIP archive from templateBytes, normalizes split XML runs around
// {{...}} tags, expands dynamic table rows from tables, replaces {{key}} variables with XML-escaped
// values, and returns the resulting .docx archive bytes.
func (e *DocxEngine) RenderDocx(
	templateBytes []byte,
	vars map[string]string,
	tables map[string][]map[string]string,
) ([]byte, error) {
	if len(templateBytes) == 0 {
		return nil, fmt.Errorf("templating: empty docx template bytes")
	}

	zr, err := zip.NewReader(bytes.NewReader(templateBytes), int64(len(templateBytes)))
	if err != nil {
		return nil, fmt.Errorf("templating: invalid docx zip archive: %w", err)
	}
	if len(zr.File) > maxDocxZipEntries {
		return nil, fmt.Errorf("templating: docx zip archive exceeds max entries (%d > %d)", len(zr.File), maxDocxZipEntries)
	}

	var outBuf bytes.Buffer
	zw := zip.NewWriter(&outBuf)

	foundDocumentXML := false
	var totalReadBytes int64
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: open zip entry %s: %w", f.Name, err)
		}
		raw, err := io.ReadAll(io.LimitReader(rc, maxDocxEntryBytes+1))
		_ = rc.Close()
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: read zip entry %s: %w", f.Name, err)
		}
		if int64(len(raw)) > maxDocxEntryBytes {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: zip entry %s exceeds max size (%d bytes)", f.Name, maxDocxEntryBytes)
		}
		totalReadBytes += int64(len(raw))
		if totalReadBytes > maxDocxTotalBytes {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: docx archive exceeds max total decompressed size (%d bytes)", maxDocxTotalBytes)
		}

		if isWordXMLTarget(f.Name) {
			if f.Name == "word/document.xml" {
				foundDocumentXML = true
			}
			rendered := renderWordXML(string(raw), vars, tables, f.Name == "word/document.xml")
			raw = []byte(rendered)
		}

		hdr := f.FileHeader
		hdr.Method = zip.Deflate
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: create zip entry %s: %w", f.Name, err)
		}
		if _, err := w.Write(raw); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("templating: write zip entry %s: %w", f.Name, err)
		}
	}

	if !foundDocumentXML {
		_ = zw.Close()
		return nil, fmt.Errorf("templating: word/document.xml not found in docx archive")
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("templating: finalize docx archive: %w", err)
	}
	return outBuf.Bytes(), nil
}

func isWordXMLTarget(name string) bool {
	return name == "word/document.xml" ||
		strings.HasPrefix(name, "word/header") ||
		strings.HasPrefix(name, "word/footer")
}

func renderWordXML(
	xmlContent string,
	vars map[string]string,
	tables map[string][]map[string]string,
	isMainDocument bool,
) string {
	// 1. Normalize split runs where Word broke {{placeholder}} across multiple <w:r><w:t> elements.
	normalized := normalizeSplitPlaceholders(xmlContent)

	// 2. Expand table rows for each table in tables.
	if len(tables) > 0 {
		tableNames := make([]string, 0, len(tables))
		for name := range tables {
			tableNames = append(tableNames, name)
		}
		sort.Strings(tableNames)

		for _, tableName := range tableNames {
			rows := tables[tableName]
			var matchedRow bool
			normalized, matchedRow = expandTableRows(normalized, tableName, rows)
			if !matchedRow && isMainDocument && len(rows) > 0 {
				normalized = injectWordTableBeforeBodyEnd(normalized, tableName, rows)
			}
		}
	}

	// 3. Replace scalar {{key}} placeholders in deterministic order.
	if len(vars) > 0 {
		keys := make([]string, 0, len(vars))
		for k := range vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			placeholder := "{{" + k + "}}"
			normalized = strings.ReplaceAll(normalized, placeholder, escapeXMLText(vars[k]))
		}
	}

	return normalized
}

// normalizeSplitPlaceholders removes intermediate XML tags that split a {{...}} token across runs.
func normalizeSplitPlaceholders(xmlContent string) string {
	return splitTagBetweenBracesRe.ReplaceAllStringFunc(xmlContent, func(match string) string {
		if !strings.Contains(match, "<") {
			return match
		}
		inner := match[2 : len(match)-2]
		cleaned := xmlTagInsideBracesRe.ReplaceAllString(inner, "")
		return "{{" + strings.TrimSpace(cleaned) + "}}"
	})
}

func expandTableRows(xmlContent string, tableName string, rows []map[string]string) (string, bool) {
	matched := false
	prefixMarker := "{{" + tableName + "."
	loopStartMarker := "{{#" + tableName + "}}"
	loopEndMarker := "{{/" + tableName + "}}"

	result := tableRowRe.ReplaceAllStringFunc(xmlContent, func(trBlock string) string {
		if !strings.Contains(trBlock, prefixMarker) && !strings.Contains(trBlock, loopStartMarker) {
			return trBlock
		}
		matched = true
		if len(rows) == 0 {
			return ""
		}

		var expanded strings.Builder
		for _, rowData := range rows {
			rowXML := strings.ReplaceAll(trBlock, loopStartMarker, "")
			rowXML = strings.ReplaceAll(rowXML, loopEndMarker, "")

			colKeys := make([]string, 0, len(rowData))
			for k := range rowData {
				colKeys = append(colKeys, k)
			}
			sort.Strings(colKeys)

			for _, col := range colKeys {
				escaped := escapeXMLText(rowData[col])
				rowXML = strings.ReplaceAll(rowXML, "{{"+tableName+"."+col+"}}", escaped)
				rowXML = strings.ReplaceAll(rowXML, "{{"+col+"}}", escaped)
			}
			expanded.WriteString(rowXML)
		}
		return expanded.String()
	})

	placeholderTableMarker := "{{table:" + tableName + "}}"
	if strings.Contains(result, placeholderTableMarker) {
		matched = true
		result = strings.ReplaceAll(result, placeholderTableMarker, buildWordTableXML(rows))
	}

	return result, matched
}

func injectWordTableBeforeBodyEnd(xmlContent string, _ string, rows []map[string]string) string {
	tblXML := buildWordTableXML(rows)
	const bodyClose = "</w:body>"
	if idx := strings.LastIndex(xmlContent, bodyClose); idx >= 0 {
		return xmlContent[:idx] + tblXML + xmlContent[idx:]
	}
	return xmlContent + tblXML
}

func buildWordTableXML(rows []map[string]string) string {
	if len(rows) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<w:tbl>`)
	for _, row := range rows {
		colKeys := make([]string, 0, len(row))
		for k := range row {
			colKeys = append(colKeys, k)
		}
		sort.Strings(colKeys)

		sb.WriteString(`<w:tr>`)
		for _, k := range colKeys {
			sb.WriteString(`<w:tc><w:p><w:r><w:t xml:space="preserve">`)
			sb.WriteString(escapeXMLText(row[k]))
			sb.WriteString(`</w:t></w:r></w:p></w:tc>`)
		}
		sb.WriteString(`</w:tr>`)
	}
	sb.WriteString(`</w:tbl>`)
	return sb.String()
}

func escapeXMLText(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// BuildMinimalDocx builds a valid OpenXML .docx archive containing bodyText inside word/document.xml.
func BuildMinimalDocx(bodyText string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
			`<w:body><w:p><w:r><w:t xml:space="preserve">` + escapeXMLText(bodyText) + `</w:t></w:r></w:p></w:body>` +
			`</w:document>`,
	}

	names := []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"}
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildMinimalTestDocx is a test helper that builds a valid .docx archive containing bodyText.
func BuildMinimalTestDocx(t testing.TB, bodyText string) []byte {
	t.Helper()
	out, err := BuildMinimalDocx(bodyText)
	if err != nil {
		t.Fatalf("BuildMinimalTestDocx failed: %v", err)
	}
	return out
}

// ExtractDocxText extracts all <w:t> text runs from a .docx archive stream.
func ExtractDocxText(r io.Reader) (string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("templating: read docx stream: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", fmt.Errorf("templating: open docx zip: %w", err)
	}

	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		dec := xml.NewDecoder(io.LimitReader(rc, maxDocxEntryBytes+1))
		var out strings.Builder
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("templating: decode document.xml: %w", err)
			}
			switch se := tok.(type) {
			case xml.StartElement:
				if se.Name.Local == "t" {
					var text string
					if err := dec.DecodeElement(&text, &se); err != nil {
						return "", err
					}
					out.WriteString(text)
				} else if se.Name.Local == "p" || se.Name.Local == "tr" {
					if out.Len() > 0 {
						out.WriteByte(' ')
					}
				} else if se.Name.Local == "tc" {
					if out.Len() > 0 {
						out.WriteString(" | ")
					}
				}
			}
		}
		return strings.TrimSpace(out.String()), nil
	}

	return "", fmt.Errorf("templating: word/document.xml not found")
}

// ExtractTextFromDocx is a test helper that extracts plain text from a .docx reader.
func ExtractTextFromDocx(t testing.TB, r io.Reader) string {
	t.Helper()
	text, err := ExtractDocxText(r)
	if err != nil {
		t.Fatalf("ExtractTextFromDocx failed: %v", err)
	}
	return text
}
