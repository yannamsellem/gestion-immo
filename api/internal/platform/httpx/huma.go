package httpx

import (
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func init() {
	huma.DefaultArrayNullable = false
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		details := make([]*huma.ErrorDetail, 0, len(errs))
		for _, err := range errs {
			if err == nil {
				continue
			}
			if detail, ok := err.(*huma.ErrorDetail); ok {
				details = append(details, detail)
			} else {
				details = append(details, &huma.ErrorDetail{Message: err.Error()})
			}
		}
		errMsg := msg
		if len(details) > 0 && details[0].Message != "" {
			errMsg = fmt.Sprintf("%s: %s", msg, details[0].Message)
		}
		return &ErrorModel{
			Status:   status,
			Title:    http.StatusText(status),
			Detail:   msg,
			ErrorMsg: errMsg,
			Errors:   details,
		}
	}
}

// ErrorModel implements huma.StatusError while preserving the top-level "error" JSON field.
type ErrorModel struct {
	Status   int                 `json:"status,omitempty" doc:"HTTP status code"`
	Title    string              `json:"title,omitempty" doc:"Short human-readable summary of the error"`
	Detail   string              `json:"detail,omitempty" doc:"Human-readable explanation specific to this occurrence"`
	ErrorMsg string              `json:"error" doc:"Error message"`
	Code     string              `json:"code,omitempty" doc:"Optional machine-readable error code"`
	Errors   []*huma.ErrorDetail `json:"errors,omitempty" doc:"Optional list of individual validation error details"`
}

// Error satisfies the standard Go error interface.
func (e *ErrorModel) Error() string {
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	return e.Detail
}

// GetStatus satisfies huma.StatusError.
func (e *ErrorModel) GetStatus() int {
	return e.Status
}

// StatusError constructs a huma.StatusError with the given HTTP status and message.
func StatusError(status int, message string) huma.StatusError {
	return &ErrorModel{
		Status:   status,
		Title:    http.StatusText(status),
		Detail:   message,
		ErrorMsg: message,
	}
}

// NewHumaAPI creates a configured huma.API mounted on mux with bearerAuth security scheme and nosniff headers.
func NewHumaAPI(mux *http.ServeMux) huma.API {
	cfg := huma.DefaultConfig("Gestion Immo API", "1.0.0")
	cfg.CreateHooks = nil
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""

	if cfg.Components.SecuritySchemes == nil {
		cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	cfg.Components.SecuritySchemes["bearerAuth"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "Ed25519-signed JWT access token",
	}

	api := humago.New(mux, cfg)
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		ctx.SetHeader("X-Content-Type-Options", "nosniff")
		next(ctx)
	})
	return api
}
