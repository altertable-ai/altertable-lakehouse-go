package altertable

import (
	"fmt"
	"io"
)

var ioEOF = io.EOF

type apiErrorBase struct {
	Message         string
	Operation       string
	Method          string
	Path            string
	StatusCodeValue int
	RetriableFlag   bool
	RequestID       string
	CorrelationID   string
	Cause           error
}

func (e apiErrorBase) Error() string {
	status := ""
	if e.StatusCodeValue > 0 {
		status = fmt.Sprintf(" status=%d", e.StatusCodeValue)
	}
	return fmt.Sprintf("%s %s%s: %s", e.Method, e.Path, status, e.Message)
}

func (e apiErrorBase) Unwrap() error   { return e.Cause }
func (e apiErrorBase) Retriable() bool { return e.RetriableFlag }
func (e apiErrorBase) StatusCode() int { return e.StatusCodeValue }

type ApiError struct{ apiErrorBase }
type AuthError struct{ apiErrorBase }
type BadRequestError struct{ apiErrorBase }
type NetworkError struct{ apiErrorBase }
type TimeoutError struct{ apiErrorBase }
type SerializationError struct{ apiErrorBase }
type ConfigurationError struct{ apiErrorBase }

type ParseError struct {
	apiErrorBase
	LineIndex  int
	RawContent string
}

func (e *ParseError) Error() string {
	if e.RawContent == "" {
		return fmt.Sprintf("%s (line=%d)", e.Message, e.LineIndex)
	}
	return fmt.Sprintf("%s (line=%d raw=%q)", e.Message, e.LineIndex, e.RawContent)
}
