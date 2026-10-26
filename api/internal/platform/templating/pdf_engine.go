package templating

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// PdfEngine converts HTML/CSS documents into PDF bytes.
type PdfEngine interface {
	RenderHTMLToPDF(ctx context.Context, htmlContent string) ([]byte, error)
}

var htmlTagsRe = regexp.MustCompile(`(?s)<[^>]*>`)

// GotenbergPdfEngine renders HTML to PDF via a Gotenberg Chromium HTTP endpoint,
// falling back to a deterministic minimal %PDF-1.4 document when no Gotenberg URL is configured.
type GotenbergPdfEngine struct {
	gotenbergURL string
	httpClient   *http.Client
}

// NewPdfEngine constructs a PdfEngine backed by Gotenberg when gotenbergURL is provided,
// or an offline deterministic PDF renderer when gotenbergURL is empty.
func NewPdfEngine(gotenbergURL string) PdfEngine {
	return NewGotenbergPdfEngine(gotenbergURL)
}

// NewGotenbergPdfEngine constructs a GotenbergPdfEngine for the given Gotenberg base URL.
func NewGotenbergPdfEngine(gotenbergURL string) *GotenbergPdfEngine {
	return &GotenbergPdfEngine{
		gotenbergURL: strings.TrimRight(strings.TrimSpace(gotenbergURL), "/"),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// WithHTTPClient returns a shallow copy of GotenbergPdfEngine configured with the given *http.Client.
func (g *GotenbergPdfEngine) WithHTTPClient(client *http.Client) *GotenbergPdfEngine {
	if client == nil {
		return g
	}
	copyG := *g
	copyG.httpClient = client
	return &copyG
}

// NewInMemoryPdfEngine constructs an offline deterministic PdfEngine for unit and integration tests.
func NewInMemoryPdfEngine() PdfEngine {
	return NewGotenbergPdfEngine("")
}

// RenderHTMLToPDF converts htmlContent to a PDF byte slice via Gotenberg's
// /forms/chromium/convert/html endpoint, or via deterministic PDF generation when gotenbergURL is empty.
func (g *GotenbergPdfEngine) RenderHTMLToPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return nil, fmt.Errorf("templating: htmlContent must not be empty")
	}

	if g.gotenbergURL == "" {
		return RenderFallbackPDF(htmlContent), nil
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	part, err := mw.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("templating: create multipart form file: %w", err)
	}
	if _, err := io.WriteString(part, htmlContent); err != nil {
		return nil, fmt.Errorf("templating: write html content: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("templating: close multipart writer: %w", err)
	}

	endpoint := g.gotenbergURL + "/forms/chromium/convert/html"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("templating: build gotenberg request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("templating: gotenberg request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errText, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("templating: gotenberg returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(errText)))
	}

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("templating: read gotenberg response: %w", err)
	}
	return pdfBytes, nil
}

// RenderFallbackPDF builds a valid minimal %PDF-1.4 document embedding the plain text extracted from htmlContent.
func RenderFallbackPDF(htmlContent string) []byte {
	plain := htmlTagsRe.ReplaceAllString(htmlContent, " ")
	plain = strings.Join(strings.Fields(plain), " ")
	escaped := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(plain)

	streamContent := fmt.Sprintf("BT /F1 10 Tf 36 750 Td (%s) Tj ET", escaped)

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	buf.WriteString("2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n")
	buf.WriteString("3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n")
	buf.WriteString(fmt.Sprintf("4 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", len(streamContent), streamContent))
	buf.WriteString("5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n")
	buf.WriteString("trailer << /Root 1 0 R >>\n%%EOF\n")
	return buf.Bytes()
}
