package errors

import "fmt"

const (
	InvalidFilterCondition  ErrorCode = "INVALID_FILTER_CONDITION"
	InvalidKeyCondition     ErrorCode = "INVALID_KEY_CONDITION"
	FailedToBuildExpression ErrorCode = "FAILED_TO_BUILD_EXPRESSION"
	MissingRequiredInput    ErrorCode = "MISSING_REQUIRED_INPUT"
	InvalidOption           ErrorCode = "INVALID_OPTION"
)

type DDBViewError struct {
	error
	Code    ErrorCode
	Message string
}

func NewDDBViewError(code ErrorCode, message string) *DDBViewError {
	return &DDBViewError{
		Code:    code,
		Message: message,
	}
}

func (e *DDBViewError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func WrapDynamoDBSearchError(err error, code ErrorCode, msg string) error {
    var message = err.Error()
	return &DDBViewError{
		error: fmt.Errorf("%s: [%w]", msg, err),
		Code:  code,
        Message: message,
	}
}
