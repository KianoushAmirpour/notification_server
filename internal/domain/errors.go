package domain

import "errors"

type APIError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(err error, status int) *APIError {
	return &APIError{Message: err.Error(), StatusCode: status}
}

var ErrInvalidOtp = errors.New("invalid otp")
var ErrTooManyAttempts = errors.New("too many attempts")
