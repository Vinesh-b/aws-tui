package errors

import "fmt"

type ErrorCode string

const (
	InvalidDataDimentions ErrorCode = "INVALID_DATA_DIMENTIONS"
)

type CoreTableError struct {
	error
	Code    ErrorCode
	Message string
}

func NewCoreTableError(code ErrorCode, message string) *CoreTableError {
	return &CoreTableError{
		Code:    code,
		Message: message,
	}
}

func (e *CoreTableError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func WrapCoreTableError(err error, code ErrorCode, msg string) error {
	var message = err.Error()
	return &CoreTableError{
		error:   fmt.Errorf("%s: [%w]", msg, err),
		Code:    code,
		Message: message,
	}
}
