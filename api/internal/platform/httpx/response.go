package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents an immutable JSON error payload.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// JSON serializes payload as JSON with the given HTTP status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

// DecodeJSON decodes a JSON request body into dst with strict error reporting.
func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}
