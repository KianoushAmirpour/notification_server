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
	ErrCodePersisting   string = "PERSIST_IN_DATABASE"
)

type DomainError struct {
	Code    string
	Message string
	Cause   error
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

var (
	ErrTooManyAttempts         = &DomainError{Code: ErrCodeRateLimited, Message: "too many request", Cause: nil}
	ErrInvalidOtp              = &DomainError{Code: ErrCodeValidation, Message: "invalid otp", Cause: nil}
	ErrInvalidCredentials      = &DomainError{Code: ErrCodeUnauthorized, Message: "invalid credentials", Cause: nil}
	ErrUserNotFound            = &DomainError{Code: ErrCodeNotFound, Message: "user not found", Cause: nil}
	ErrEmailNotFound           = &DomainError{Code: ErrCodeNotFound, Message: "email not found", Cause: nil}
	ErrDbConnection            = &DomainError{Code: ErrCodeInternal, Message: "db connectin failed", Cause: nil}
	ErrPersistUser             = &DomainError{Code: ErrCodePersisting, Message: "persisting user failed", Cause: nil}
	ErrPersistVerification     = &DomainError{Code: ErrCodePersisting, Message: "persisting requestid-email failed", Cause: nil}
	ErrUnableToDeleteUser      = &DomainError{Code: ErrCodeInternal, Message: "unable to delete user from database", Cause: nil}
	ErrPersistStory            = &DomainError{Code: ErrCodePersisting, Message: "persisting story failed", Cause: nil}
	ErrPersistOtp              = &DomainError{Code: ErrCodePersisting, Message: "failed to save otp", Cause: nil}
	ErrRequestIDNotFound       = &DomainError{Code: ErrCodeNotFound, Message: "request id not found", Cause: nil}
	ErrOtpKeyNotFound          = &DomainError{Code: ErrCodeNotFound, Message: "key not found", Cause: nil}
	ErrTypeConvertion          = &DomainError{Code: ErrCodeValidation, Message: "failed to convert the type", Cause: nil}
	ErrFailedIncrementOtpRetry = &DomainError{Code: ErrCodeInternal, Message: "failed to increment the retry attempts", Cause: nil}
	ErrInvalidJWTToken         = &DomainError{Code: ErrCodeUnauthorized, Message: "invalid jwt token", Cause: nil}
	ErrInvalidJWTMethod        = &DomainError{Code: ErrCodeUnauthorized, Message: "invalid jwt method", Cause: nil}
	ErrPersistRefreshToken     = &DomainError{Code: ErrCodePersisting, Message: "persisting refresh token failed", Cause: nil}
)
