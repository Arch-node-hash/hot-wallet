package errors

import "fmt"

// Code adalah kode error yang stabil untuk API.
type Code string

const (
	CodeNotFound     Code = "not_found"
	CodeInvalid      Code = "invalid"
	CodeConflict     Code = "conflict"
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeInternal     Code = "internal"
)

// Error adalah error aplikasi dengan kode dan pesan.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}
