package httpx_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

func TestDecodeJSON_RejectsOversizedPayload(t *testing.T) {
	oversized := `{"data":"` + strings.Repeat("a", int(httpx.DefaultMaxJSONBytes)) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(oversized))

	var dst map[string]string
	err := httpx.DecodeJSON(req, &dst)
	if err == nil {
		t.Fatal("expected DecodeJSON to reject payload exceeding DefaultMaxJSONBytes, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("expected size limit error, got: %v", err)
	}
}

func TestJSON_SetsNoSniffHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.JSON(rec, http.StatusOK, map[string]string{"status": "ok"})

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got %q", got)
	}

	recNoContent := httptest.NewRecorder()
	httpx.NoContent(recNoContent)
	if got := recNoContent.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff on NoContent, got %q", got)
	}
}

func TestRequestLogger_LogsAndRecoversPanic(t *testing.T) {
	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logBuf, nil)))
	defer slog.SetDefault(origLogger)

	okHandler := httpx.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpx.OK(w, map[string]string{"ok": "true"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	rec := httptest.NewRecorder()
	okHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	logged := logBuf.String()
	if !strings.Contains(logged, `"path":"/api/v1/entities"`) || !strings.Contains(logged, `"status":200`) {
		t.Fatalf("expected request log with path and status 200, got: %s", logged)
	}

	logBuf.Reset()
	panicHandler := httpx.RequestLogger(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("unexpected boom")
	}))

	panicReq := httptest.NewRequest(http.MethodPost, "/api/v1/boom", nil)
	panicRec := httptest.NewRecorder()
	panicHandler.ServeHTTP(panicRec, panicReq)

	if panicRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 on panic recovery, got %d", panicRec.Code)
	}
	panicLogged := logBuf.String()
	if !strings.Contains(panicLogged, "unexpected boom") || !strings.Contains(panicLogged, `"status":500`) {
		t.Fatalf("expected panic log and status 500 log, got: %s", panicLogged)
	}
}
