package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/utils"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
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
			httpErr := utils.HttpError{Message: "Invalid Authorization format", Code: domain.ErrCodeUnauthorized, StatusCode: http.StatusUnauthorized}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		tokenString := parts[1]
		token, err := auth.VerifyJWTToken(tokenString, secretKey)
		if err != nil {
			httpErr := utils.HttpError{Message: "Invalid or expired token", Code: domain.ErrCodeUnauthorized, StatusCode: http.StatusUnauthorized}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		user_id := token.UserID
		c.Set("user_id", int(user_id))
		c.Next()
	}
}

func AddRequestID() gin.HandlerFunc {

	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()

		}
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
			httpErr := utils.HttpError{Message: "invalid content type, expected application/json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
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
				httpErr := utils.HttpError{Message: "body must not be empty", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.Is(err, io.ErrUnexpectedEOF):
				httpErr := utils.HttpError{Message: "body contains badly-formed json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case err.Error() == "http: request body too large":
				httpErr := utils.HttpError{Message: fmt.Sprintf("body must not be larger than %d bytes", maxsize), Code: domain.ErrCodeValidation, StatusCode: http.StatusRequestEntityTooLarge}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &syntanxErr):
				httpErr := utils.HttpError{Message: fmt.Sprintf("body contains badly-formed json at character %d", syntanxErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &unmarshalTypeErr):
				httpErr := utils.HttpError{Message: fmt.Sprintf("body contains incorrect json type for %q at %d", unmarshalTypeErr.Field, unmarshalTypeErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case strings.HasPrefix(err.Error(), "json: unknown field"):
				fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
				httpErr := utils.HttpError{Message: fmt.Sprintf("body contains unknow key %s", fieldname), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &invalidmarshaltype):
				httpErr := utils.HttpError{Message: fmt.Sprintf("error unmarshaling json: %s", invalidmarshaltype.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			default:
				httpErr := utils.HttpError{Message: fmt.Sprintf("error happend: %s", err.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			}
		}

		dec := json.NewDecoder(c.Request.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(&struct{}{})
		if err != io.EOF {
			httpErr := utils.HttpError{Message: "body must contain only one json value", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}

		validate := validator.New()
		err = validate.RegisterValidation("passwod_strength", utils.PasswordValidator)
		if err != nil {
			httpErr := utils.HttpError{Message: "failed to register validation", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		err = validate.RegisterValidation("user_preferences_check", utils.UserPreferencesValidation)
		if err != nil {
			httpErr := utils.HttpError{Message: "failed to register validation", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		err = validate.Struct(u)
		if err != nil {
			httpErr := utils.HttpError{Message: err.Error(), Code: domain.ErrCodeInternal, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Set("payload", u)
		c.Next()

	}
}

func RateLimiterMiddelware(ipratelimiter *IPRateLimiter, capacity, fillrate float64, logger domain.LoggingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		log := logger.With("service", "rate_limiter", "request_id", c.GetString("RequestID"))
		if ip == "" {
			log.Warn("extract_user_ip", "reason", "invalid_user_ip")
			httpErr := utils.HttpError{Message: "invalid ip", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		rateLimiter := ipratelimiter.RequestRateLimiter(ip, capacity, fillrate)

		if !rateLimiter.AllowRequest() {
			log.Warn("rate_limit_check", "reason", "rate_limit_exceeded", "user_ip", ip)
			httpErr := utils.HttpError{Message: "Rate Limit Exceeded", Code: domain.ErrCodeRateLimited, StatusCode: http.StatusTooManyRequests}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Next()
	}
}

func LoggingRequestMiddleware(logger domain.LoggingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		logger.Info("http_request_start",
			"request_id", c.GetString("RequestID"),
			"method", c.Request.Method,
			"user-agent", c.Request.UserAgent(),
			"path", c.FullPath())

		c.Next()
	}
}

func PanicRecoveryMiddleware(logger domain.LoggingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		defer func() {
			if r := recover(); r != nil {
				logger.Error("internal server error",
					"request_id", c.GetString("RequestID"),
					"method", c.Request.Method,
					"path", c.FullPath(),
					"reason", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)

				httpErr := utils.HttpError{Message: "internal server error", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			}
		}()

		c.Next()
	}
}
