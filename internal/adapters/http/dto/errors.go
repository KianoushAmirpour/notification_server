package dto

import (
	"errors"
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

func MapErr(err error) HttpError {
	var de *domain.DomainError
	if errors.As(err, &de) {
		return MapDomainErrToHttpErr(de)
	}
	return HttpError{
		Message:    "internal server error",
		Code:       domain.ErrCodeInternal,
		StatusCode: http.StatusInternalServerError,
	}
}

func MapDomainErrToHttpErr(err *domain.DomainError) HttpError {
	switch err.Code {
	case domain.ErrCodeValidation:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusBadRequest,
		}
	case domain.ErrCodeNotFound:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusNotFound,
		}
	case domain.ErrCodeUnauthorized:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusUnauthorized,
		}
	case domain.ErrCodeForbidden:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusForbidden,
		}
	case domain.ErrCodeConflict:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusConflict,
		}
	case domain.ErrCodeRateLimited:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusTooManyRequests,
		}
	case domain.ErrCodeExternal:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusServiceUnavailable,
		}
	case domain.ErrCodePersisting:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusInternalServerError,
		}
	default:
		return HttpError{
			Message:    err.Message,
			Code:       err.Code,
			StatusCode: http.StatusInternalServerError,
		}
	}

}
