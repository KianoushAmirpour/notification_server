package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/dto"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/utils"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func AuthenticateMiddleware(auth domain.JwtTokenRepository, secretKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// token, err := c.Cookie("access-token")
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			httpErr := dto.HttpError{Message: "Invalid Authorization format", Code: domain.ErrCodeUnauthorized, StatusCode: http.StatusUnauthorized}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		tokenString := parts[1]
		token, err := auth.VerifyJWTToken(tokenString, secretKey)
		if err != nil {
			httpErr := dto.HttpError{Message: "Invalid or expired token", Code: domain.ErrCodeUnauthorized, StatusCode: http.StatusUnauthorized}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		user_id := token.UserID
		c.Set("user_id", int(user_id))
		c.Next()
	}
}

func AddRequestIDAndTime() gin.HandlerFunc {

	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()

		}

		ctx := observability.WithRequestID(c.Request.Context(), requestID)
		ctx = observability.WithrequestStartTimeKey(ctx)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set("X-Request-Id", requestID)
		c.Set("RequestID", requestID)
		c.Next()
	}
}

func AddCorrelationID() gin.HandlerFunc {

	return func(c *gin.Context) {
		corelationId := c.GetHeader("X-Correlation-Id")
		if corelationId == "" {
			corelationId = uuid.New().String()

		}
		c.Writer.Header().Set("X-Correlation-Id", corelationId)
		c.Set("CorrelationID", corelationId)
		c.Next()
	}
}

func CheckContentType() gin.HandlerFunc {

	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")

		parts := strings.Split(contentType, ";")
		if len(parts) == 0 || strings.TrimSpace(strings.ToLower(parts[0])) != "application/json" {
			httpErr := dto.HttpError{Message: "invalid content type, expected application/json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Next()
	}
}

func CheckContentBody[T any](maxsize int) gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(maxsize))

		var u T

		err := c.ShouldBindJSON(&u)

		if err != nil {
			var syntanxErr *json.SyntaxError
			var unmarshalTypeErr *json.UnmarshalTypeError
			var invalidmarshaltype *json.InvalidUnmarshalError

			switch {

			case errors.Is(err, io.EOF):
				httpErr := dto.HttpError{Message: "body must not be empty", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.Is(err, io.ErrUnexpectedEOF):
				httpErr := dto.HttpError{Message: "body contains badly-formed json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case err.Error() == "http: request body too large":
				httpErr := dto.HttpError{Message: fmt.Sprintf("body must not be larger than %d bytes", maxsize), Code: domain.ErrCodeValidation, StatusCode: http.StatusRequestEntityTooLarge}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &syntanxErr):
				httpErr := dto.HttpError{Message: fmt.Sprintf("body contains badly-formed json at character %d", syntanxErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &unmarshalTypeErr):
				httpErr := dto.HttpError{Message: fmt.Sprintf("body contains incorrect json type for %q at %d", unmarshalTypeErr.Field, unmarshalTypeErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case strings.HasPrefix(err.Error(), "json: unknown field"):
				fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
				httpErr := dto.HttpError{Message: fmt.Sprintf("body contains unknow key %s", fieldname), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &invalidmarshaltype):
				httpErr := dto.HttpError{Message: fmt.Sprintf("error unmarshaling json: %s", invalidmarshaltype.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			default:
				httpErr := dto.HttpError{Message: fmt.Sprintf("error happend: %s", err.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			}
		}

		dec := json.NewDecoder(c.Request.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(&struct{}{})
		if err != io.EOF {
			httpErr := dto.HttpError{Message: "body must contain only one json value", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}

		validate := validator.New()
		err = validate.RegisterValidation("passwod_strength", utils.PasswordValidator)
		if err != nil {
			httpErr := dto.HttpError{Message: "failed to register validation", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		err = validate.RegisterValidation("user_preferences_check", utils.UserPreferencesValidation)
		if err != nil {
			httpErr := dto.HttpError{Message: "failed to register validation", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		err = validate.Struct(u)
		if err != nil {
			httpErr := dto.HttpError{Message: err.Error(), Code: domain.ErrCodeInternal, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Set("payload", u)
		c.Next()

	}
}

func RateLimiterMiddelware(ipratelimiter *RedisRateLimiter, logger domain.LoggingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		log := logger.With(
			"service.name", "rate_limiter_middleware",
			"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
			"event.category", []string{"authentication", "web"},
			"http.request.method", c.Request.Method,
			"http.request.path", c.FullPath(),
			"http.request.agent", c.Request.UserAgent(),
			"http.request.id", observability.GetRequestID(c.Request.Context()),
			"client.ip", c.ClientIP(),
		)
		if ip == "" {
			log.Error(
				"invalid ip address",
				"error.message", "invalid ip address",
				"error.code", http.StatusBadRequest,
				"event.action", "middleware.rate_limiter",
				"event.outcome", "failed",
				"event.type", []string{"end", "denied"},
			)
			httpErr := dto.HttpError{Message: "invalid ip", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		// rateLimiter := ipratelimiter.RequestRateLimiter(ip, capacity, fillrate)
		ok, err := ipratelimiter.AllowRequest(c, ip)
		if err != nil {
			log.Error(
				"redis is not available",
				"error.message", "Service Unavailable",
				"error.code", http.StatusServiceUnavailable,
				"event.action", "middleware.rate_limiter",
				"event.outcome", "failed",
				"event.type", []string{"end", "denied"},
			)
			httpErr := dto.HttpError{Message: "Rate Limit Exceeded", Code: domain.ErrCodeExternal, StatusCode: http.StatusServiceUnavailable}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		if !ok {
			log.Error(
				"rate limit exceeded",
				"error.message", "rate limit exceeded",
				"error.code", http.StatusTooManyRequests,
				"event.action", "middleware.rate_limiter",
				"event.outcome", "failed",
				"event.type", []string{"end", "denied"},
			)
			httpErr := dto.HttpError{Message: "Rate Limit Exceeded", Code: domain.ErrCodeRateLimited, StatusCode: http.StatusTooManyRequests}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Next()
	}
}

// func LoggingRequestMiddleware(logger domain.LoggingRepository) gin.HandlerFunc {
// 	return func(c *gin.Context) {

// 		log := logger.With(
// 			"service.name", "user_verification",
// 			"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
// 			"event.category", []string{"web"},
// 			"http.request.method", c.Request.Method,
// 			"http.request.path", c.FullPath(),
// 			"http.request.agent", c.Request.UserAgent(),
// 			"http.request.id", c.GetString("RequestID"),
// 			"client.ip", c.ClientIP(),
// 		)

// 		log.Info(
// 			"http request started",
// 			"event.action", "http.request",
// 			"event.type", "start",
// 		)

// 		c.Next()
// 	}
// }

func PanicRecoveryMiddleware(logger domain.LoggingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		defer func() {
			log := logger.With(
				"service.name", "middleware_panic_recovery",
				"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
				"event.category", []string{"authentication", "web"},
				"http.request.method", c.Request.Method,
				"http.request.path", c.FullPath(),
				"http.request.agent", c.Request.UserAgent(),
				"http.request.id", observability.GetRequestID(c.Request.Context()),
				"client.ip", c.ClientIP(),
			)
			if r := recover(); r != nil {
				log.Error(
					"internal server error",
					"error.message", fmt.Sprintf("%v", r),
					"error.code", http.StatusInternalServerError,
					"event.action", "middleware.panic_recovery",
					"event.outcome", "failed",
					"event.type", []string{"end", "error"},
				)

				httpErr := dto.HttpError{Message: "internal server error", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			}
		}()

		c.Next()
	}
}
