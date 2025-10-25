package transport

import (
	"net/http"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type HttpError struct {
	Message    string `json:"message"`
	Code       string `json:"code"`
	StatusCode int    `json:"status_code"`
}

func (e *HttpError) Error() string {
	return e.Message
}

func MapDomainErrToHttpErr(err *domain.DomainError) *HttpError {
	switch err.Code {
	case domain.ErrCodeValidation:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusBadRequest,
		}
	case domain.ErrCodeNotFound:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusNotFound,
		}
	case domain.ErrCodeUnauthorized:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusUnauthorized,
		}
	case domain.ErrCodeForbidden:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusForbidden,
		}
	case domain.ErrCodeConflict:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusConflict,
		}
	case domain.ErrCodeRateLimited:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusTooManyRequests,
		}
	case domain.ErrCodeExternal:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusServiceUnavailable,
		}
	default:
		return &HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusInternalServerError,
		}
	}

}
