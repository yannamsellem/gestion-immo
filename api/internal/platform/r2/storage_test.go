package r2_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInMemoryStorage_UploadDownloadAndPresign(t *testing.T) {
	ctx := context.Background()
	store := r2.NewInMemoryStorage()

	payload := []byte("contrat de bail signé")
	if err := store.Upload(ctx, "leases/lse_1/bail.pdf", "application/pdf", bytes.NewReader(payload)); err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	rc, err := store.Download(ctx, "leases/lse_1/bail.pdf")
	if err != nil {
		t.Fatalf("unexpected download error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("expected %q, got %q (err=%v)", payload, got, err)
	}

	_, err = store.Download(ctx, "leases/unknown.pdf")
	if !errors.Is(err, r2.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}

	signedURL, err := store.PresignGetURL(ctx, "leases/lse_1/bail.pdf", 10*time.Minute)
	if err != nil || !strings.Contains(signedURL, "leases/lse_1/bail.pdf") {
		t.Fatalf("unexpected presigned URL: %q (err=%v)", signedURL, err)
	}
}

func TestS3Storage_SignsRequestsWithSigV4(t *testing.T) {
	ctx := context.Background()
	objects := make(map[string][]byte)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=test_ak/") {
			http.Error(w, "missing AWS4-HMAC-SHA256 Authorization header", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			objects[r.URL.Path] = data
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			data, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}

	s3 := r2.NewS3Storage("https://r2.cloudflarestorage.test", "test_ak", "test_sk", "gestion-immo-bucket").WithHTTPClient(client)

	if err := s3.Upload(ctx, "dossiers/gli_1.zip", "application/zip", strings.NewReader("PK-ZIP-DATA")); err != nil {
		t.Fatalf("unexpected S3 upload error: %v", err)
	}

	rc, err := s3.Download(ctx, "dossiers/gli_1.zip")
	if err != nil {
		t.Fatalf("unexpected S3 download error: %v", err)
	}
	defer rc.Close()

	got, _ := io.ReadAll(rc)
	if string(got) != "PK-ZIP-DATA" {
		t.Fatalf("expected PK-ZIP-DATA, got %q", string(got))
	}

	_, err = s3.Download(ctx, "dossiers/missing.zip")
	if !errors.Is(err, r2.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound on missing S3 key, got %v", err)
	}

	presigned, err := s3.PresignGetURL(ctx, "dossiers/gli_1.zip", 15*time.Minute)
	if err != nil || !strings.Contains(presigned, "X-Amz-Algorithm=AWS4-HMAC-SHA256") || !strings.Contains(presigned, "X-Amz-Signature=") {
		t.Fatalf("unexpected presigned URL: %q (err=%v)", presigned, err)
	}
}
