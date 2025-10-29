package domain

import (
	"fmt"
)

const (
	ErrCodeValidation   string = "VALIDATION_ERROR"
	ErrCodeNotFound     string = "NOT_FOUND"
	ErrCodeUnauthorized string = "UNAUTHORIZED"
	ErrCodeForbidden    string = "FORBIDDEN"
	ErrCodeConflict     string = "CONFLICT"
	ErrCodeInternal     string = "INTERNAL_ERROR"
	ErrCodeExternal     string = "EXTERNAL_SERVICE_ERROR"
	ErrCodeRateLimited  string = "RATE_LIMITED"
)

type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause"`
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("Message:%s, Cause:%v", e.Message, e.Cause)
	}
	return fmt.Sprintf("Message:%s", e.Message)

}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func NewDomainError(code, msg string, cause error) *DomainError {
	return &DomainError{Code: code, Message: msg, Cause: cause}
}

var ErrTooManyAttempts = &DomainError{Code: ErrCodeRateLimited, Message: "too many request", Cause: nil}
var ErrInvalidOtp = &DomainError{Code: ErrCodeValidation, Message: "invalid otp", Cause: nil}
