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

// var (
// 	ErrInvalidOtp                = &APIError{Message: "the provided otp is invalid", StatusCode: http.StatusUnauthorized, Code: "INVALID_OTP"}
// 	ErrTooManyAttempts           = &APIError{Message: "too many request. please try again later", StatusCode: http.StatusTooManyRequests, Code: "TOO_MANY_ATTEMPTS"}
// 	ErrUserNotFound              = &APIError{Message: "user not found", StatusCode: http.StatusNotFound, Code: "USER_NOT_FOUND"}
// 	ErrFailedToCreateUser        = &APIError{Message: "failed to register user in staging table", StatusCode: http.StatusInternalServerError, Code: "FAILED_TO_CREATE_USER"}
// 	ErrFailedToSaveEmail         = &APIError{Message: "failed to save by email", StatusCode: http.StatusInternalServerError, Code: "FAILED_TO_SAVE_EMAIL"}
// 	ErrEmailNotFound             = &APIError{Message: "email not found", StatusCode: http.StatusNotFound, Code: "EMAIL_NOT_FOUND"}
// 	ErrFailedToDeleteUser        = &APIError{Message: "failed to delete user from database", StatusCode: http.StatusInternalServerError, Code: "FAILED_TO_DELETE_USER"}
// 	ErrFailedToSaveStoryMetaData = &APIError{Message: "failed to store generated story metadata", StatusCode: http.StatusInternalServerError, Code: "FAILED_TO_STORE_STORY_METADATA"}
// 	ErrFailedToUploadStory       = &APIError{Message: "failed to upload story", StatusCode: http.StatusInternalServerError, Code: "FAILED_TO_UPLOAD_STORY"}
// 	ErrDatabaseTransactionFailed = &APIError{Message: "transaction failed", StatusCode: http.StatusInternalServerError, Code: "DB_TRANSACTION_FAILED"}
// 	ErrEmailRegisteredAlready    = &APIError{Message: "user with this email already exists", StatusCode: http.StatusConflict, Code: "EMAIL_ALREADY_REGISTERED"}
// 	ErrFailedToHashPassword      = &APIError{Message: "failed to hash password", StatusCode: http.StatusInternalServerError, Code: "PASSWORD_HASH_FAILED"}
// )
