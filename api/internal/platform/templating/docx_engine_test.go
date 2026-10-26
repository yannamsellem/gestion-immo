package templating_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRenderDocx_ReplacesPlaceholdersAndUnpaidMonthsTable(t *testing.T) {
	engine := templating.NewDocxEngine()
	sampleDocx := templating.BuildMinimalTestDocx(t, "Bailleur: {{societe.nom}} - Locataire: {{locataire.nom}} - Total impayé: {{impaye.total}}")

	out, err := engine.RenderDocx(sampleDocx, map[string]string{
		"societe.nom":   "SARL INVESTISSIMO",
		"locataire.nom": "Jean Dupont",
		"impaye.total":  "1 600,00 €",
	}, map[string][]map[string]string{
		"impayes": {
			{"periode": "2026-01", "montant_du": "800,00 €", "statut": "IMPAYE"},
			{"periode": "2026-02", "montant_du": "800,00 €", "statut": "IMPAYE"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected docx error: %v", err)
	}
	extracted := templating.ExtractTextFromDocx(t, bytes.NewReader(out))
	if !strings.Contains(extracted, "SARL INVESTISSIMO") || !strings.Contains(extracted, "1 600,00 €") {
		t.Fatalf("expected replaced variables in docx, got: %s", extracted)
	}
	if !strings.Contains(extracted, "2026-01") || !strings.Contains(extracted, "2026-02") || !strings.Contains(extracted, "800,00 €") {
		t.Fatalf("expected unpaid months table rows in docx, got: %s", extracted)
	}
}

func TestRenderDocx_EscapesXMLCharacters(t *testing.T) {
	engine := templating.NewDocxEngine()
	sampleDocx := templating.BuildMinimalTestDocx(t, "Société: {{societe.nom}}")

	out, err := engine.RenderDocx(sampleDocx, map[string]string{
		"societe.nom": "SCI DUPONT & FILS <Sud>",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected docx error: %v", err)
	}
	extracted := templating.ExtractTextFromDocx(t, bytes.NewReader(out))
	if !strings.Contains(extracted, "SCI DUPONT & FILS <Sud>") {
		t.Fatalf("expected XML-escaped entity name to round-trip cleanly, got: %s", extracted)
	}
}

func TestPdfEngine_FallbackAndGotenbergHTTP(t *testing.T) {
	ctx := context.Background()

	// 1. Offline deterministic fallback PDF
	fallbackEngine := templating.NewInMemoryPdfEngine()
	pdfBytes, err := fallbackEngine.RenderHTMLToPDF(ctx, "<html><body><h1>Quittance de Loyer - SARL INVESTISSIMO</h1></body></html>")
	if err != nil {
		t.Fatalf("unexpected fallback pdf error: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")) || !bytes.Contains(pdfBytes, []byte("SARL INVESTISSIMO")) {
		t.Fatalf("unexpected fallback pdf content: %s", string(pdfBytes))
	}

	// 2. Gotenberg HTTP multipart endpoint (in-process RoundTripper)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/forms/chromium/convert/html" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4\n%GOTENBERG-RENDERED\n%%EOF"))
	})

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}

	gotenbergEngine := templating.NewGotenbergPdfEngine("http://gotenberg.internal:3000").WithHTTPClient(client)
	gotPDF, err := gotenbergEngine.RenderHTMLToPDF(ctx, "<html><body>Test Gotenberg</body></html>")
	if err != nil {
		t.Fatalf("unexpected gotenberg pdf error: %v", err)
	}
	if !bytes.Contains(gotPDF, []byte("GOTENBERG-RENDERED")) {
		t.Fatalf("expected gotenberg response bytes, got: %s", string(gotPDF))
	}
}
