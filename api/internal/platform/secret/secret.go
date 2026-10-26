package secret

import "log/slog"

// RedactedPlaceholder is the replacement string emitted whenever a secret.String is logged or formatted.
const RedactedPlaceholder = "[REDACTED]"

// String wraps a sensitive string value (password, JWT, refresh token, API key, DB DSN)
// so that it is never accidentally written to structured logs or standard output.
type String struct {
	value string
}

// New constructs a new redacted secret String.
func New(val string) String {
	return String{value: val}
}

// Value explicitly extracts the underlying raw secret string for cryptographic or wire operations.
func (s String) Value() string {
	return s.value
}

// IsEmpty reports whether the underlying secret is empty.
func (s String) IsEmpty() bool {
	return s.value == ""
}

// String implements fmt.Stringer to prevent accidental leakage via fmt verbs.
func (s String) String() string {
	return RedactedPlaceholder
}

// GoString implements fmt.GoStringer to prevent accidental leakage via %#v.
func (s String) GoString() string {
	return RedactedPlaceholder
}

// LogValue implements slog.LogValuer and always returns "[REDACTED]".
func (s String) LogValue() slog.Value {
	return slog.StringValue(RedactedPlaceholder)
}
