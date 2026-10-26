package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	// DefaultMaxJSONBytes is the maximum allowed JSON request body size (2 MiB).
	DefaultMaxJSONBytes int64 = 2 << 20
	// MaxUploadBodyBytes is the maximum allowed file/base64 upload request body size (25 MiB).
	MaxUploadBodyBytes int64 = 25 << 20
)

// ErrorResponse represents an immutable JSON error payload.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// JSON serializes payload as JSON with the given HTTP status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if payload == nil || status == http.StatusNoContent {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteJSON is an alias for JSON for handler ergonomics.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	JSON(w, status, payload)
}

// Error writes an immutable JSON ErrorResponse with the given HTTP status code.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{Error: message})
}

// WriteError is an alias for Error for handler ergonomics.
func WriteError(w http.ResponseWriter, status int, message string) {
	Error(w, status, message)
}

// ErrorWithCode writes an immutable JSON ErrorResponse with an explicit machine-readable code.
func ErrorWithCode(w http.ResponseWriter, status int, code string, message string) {
	JSON(w, status, ErrorResponse{Error: message, Code: code})
}

// OK writes a 200 OK JSON response.
func OK(w http.ResponseWriter, payload any) {
	JSON(w, http.StatusOK, payload)
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, payload any) {
	JSON(w, http.StatusCreated, payload)
}

// Accepted writes a 202 Accepted JSON response.
func Accepted(w http.ResponseWriter, payload any) {
	JSON(w, http.StatusAccepted, payload)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNoContent)
}

// BadRequest writes a 400 Bad Request JSON error response.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized writes a 401 Unauthorized JSON error response.
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden writes a 403 Forbidden JSON error response.
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound writes a 404 Not Found JSON error response.
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// Conflict writes a 409 Conflict JSON error response.
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, message)
}

// UnprocessableEntity writes a 422 Unprocessable Entity JSON error response.
func UnprocessableEntity(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, message)
}

// InternalError writes a 500 Internal Server Error JSON error response.
func InternalError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}

// LogInternalError logs the underlying error via slog and writes a 500 Internal Server Error JSON response.
func LogInternalError(w http.ResponseWriter, message string, err error) {
	slog.Error(message, slog.Any("error", err))
	InternalError(w, message)
}

// DecodeJSON decodes a JSON request body into dst with the default 2 MiB size limit.
func DecodeJSON(r *http.Request, dst any) error {
	return DecodeJSONWithLimit(r, dst, DefaultMaxJSONBytes)
}

// DecodeJSONWithLimit decodes a JSON request body into dst, rejecting bodies larger than maxBytes.
func DecodeJSONWithLimit(r *http.Request, dst any, maxBytes int64) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	if maxBytes <= 0 {
		maxBytes = DefaultMaxJSONBytes
	}
	limited := &io.LimitedReader{R: r.Body, N: maxBytes + 1}
	dec := json.NewDecoder(limited)
	if err := dec.Decode(dst); err != nil {
		if limited.N <= 0 {
			return fmt.Errorf("request body exceeds maximum size of %d bytes", maxBytes)
		}
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	if _, err := io.Copy(io.Discard, limited); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	if limited.N <= 0 {
		return fmt.Errorf("request body exceeds maximum size of %d bytes", maxBytes)
	}
	return nil
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.status = http.StatusOK
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// RequestLogger wraps next with panic recovery and structured request logging via slog.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("http handler panic recovered",
					slog.Any("panic", recovered),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				if !rec.wroteHeader {
					InternalError(rec, "internal server error")
				}
			}
			slog.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		}()
		next.ServeHTTP(rec, r)
	})
}
