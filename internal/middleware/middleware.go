package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/transport"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func AuthenticateMiddleware(secretKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// token, err := c.Cookie("access-token")
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			httpErr := transport.HttpError{Message: "Invalid Authorization format", Code: domain.ErrCodeValidation, StatusCode: http.StatusUnauthorized}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		tokenString := parts[1]
		token, err := adapters.VerifyJWTToken(tokenString, secretKey)
		if err != nil {
			httpErr := transport.HttpError{Message: "Invalid or expired token", Code: domain.ErrCodeValidation, StatusCode: http.StatusUnauthorized}
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
			httpErr := transport.HttpError{Message: "invalid content type, expected application/json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Next()
	}
}

func RegisterMiddelware[T any](maxsize int) gin.HandlerFunc {
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
				httpErr := transport.HttpError{Message: "body must not be empty", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.Is(err, io.ErrUnexpectedEOF):
				httpErr := transport.HttpError{Message: "body contains badly-formed json", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case err.Error() == "http: request body too large":
				httpErr := transport.HttpError{Message: fmt.Sprintf("body must not be larger than %d bytes", maxsize), Code: domain.ErrCodeValidation, StatusCode: http.StatusRequestEntityTooLarge}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &syntanxErr):
				httpErr := transport.HttpError{Message: fmt.Sprintf("body contains badly-formed json at character %d", syntanxErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &unmarshalTypeErr):
				httpErr := transport.HttpError{Message: fmt.Sprintf("body contains incorrect json type for %q at %d", unmarshalTypeErr.Field, unmarshalTypeErr.Offset), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case strings.HasPrefix(err.Error(), "json: unknown field"):
				fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
				httpErr := transport.HttpError{Message: fmt.Sprintf("body contains unknow key %s", fieldname), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			case errors.As(err, &invalidmarshaltype):
				httpErr := transport.HttpError{Message: fmt.Sprintf("error unmarshaling json: %s", invalidmarshaltype.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			default:
				httpErr := transport.HttpError{Message: fmt.Sprintf("error happend: %s", err.Error()), Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
				return

			}
		}

		dec := json.NewDecoder(c.Request.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(&struct{}{})
		if err != io.EOF {
			httpErr := transport.HttpError{Message: "body must contain only one json value", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}

		validate := validator.New()
		err = validate.RegisterValidation("passwod_strength", PasswordValidator)
		if err != nil {
			httpErr := transport.HttpError{Message: "failed to register validation", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		err = validate.Struct(u)
		if err != nil {
			httpErr := transport.HttpError{Message: err.Error(), Code: domain.ErrCodeInternal, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Set("payload", u)
		c.Next()

	}
}

func PasswordValidator(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`\d`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`[@$!%*?&]`).MatchString(password) {
		return false
	}

	return true
}

// func ValidationIDParam() gin.HandlerFunc {
// 	return func(c *gin.Context) {

// 		id := c.Param("id")
// 		intid, err := strconv.Atoi(id)
// 		if err != nil || intid <= 0 {
// 			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid id parameter; must be a positive integer"})
// 			return
// 		}
// 		c.Set("id", intid)
// 		c.Next()

// 	}
// }

func RateLimiterMiddelware(ipratelimiter *adapters.IPRateLimiter, capacity, fillrate float64, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		log := logger.With(slog.String("service", "rate_limit"), slog.String("request_id", c.GetString("RequestID")))
		if ip == "" {
			log.Warn("extract_user_ip", slog.String("reason", "invalid_user_ip"))
			httpErr := transport.HttpError{Message: "invalid ip", Code: domain.ErrCodeValidation, StatusCode: http.StatusBadRequest}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		rateLimiter := ipratelimiter.RequestRateLimiter(ip, capacity, fillrate)

		if !rateLimiter.AllowRequest() {
			log.Warn("rate_limit_check", slog.String("reason", "rate_limit_exceeded"), slog.String("user_ip", ip))
			httpErr := transport.HttpError{Message: "Rate Limit Exceeded", Code: domain.ErrCodeRateLimited, StatusCode: http.StatusTooManyRequests}
			c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			return
		}
		c.Next()
	}
}

func LoggingRequestMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		logger.Info("http_request_start",
			slog.String("request_id", c.GetString("RequestID")),
			slog.String("method", c.Request.Method),
			slog.String("user-agent", c.Request.UserAgent()),
			slog.String("path", c.FullPath()))

		c.Next()
	}
}

func PanicRecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		defer func() {
			if r := recover(); r != nil {
				logger.Error("internal server error",
					slog.String("request_id", c.GetString("RequestID")),
					slog.String("method", c.Request.Method),
					slog.String("path", c.FullPath()),
					slog.String("reason", fmt.Sprintf("%v", r)),
					slog.String("stack", string(debug.Stack())),
				)

				httpErr := transport.HttpError{Message: "internal server error", Code: domain.ErrCodeInternal, StatusCode: http.StatusInternalServerError}
				c.AbortWithStatusJSON(httpErr.StatusCode, httpErr)
			}
		}()

		c.Next()
	}
}
