package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization format"})
			return
		}
		tokenString := parts[1]
		token, err := adapters.VerifyJWTToken(tokenString, secretKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})

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
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid content type, expected application/json"})
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
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body must not be empty"})
				return

			case errors.Is(err, io.ErrUnexpectedEOF):
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body contains badly-formed json"})
				return

			case err.Error() == "http: request body too large":
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("body must not be larger than %d bytes", maxsize)})
				return

			case errors.As(err, &syntanxErr):
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("body contains badly-formed json at character %d", syntanxErr.Offset)})
				return

			case errors.As(err, &unmarshalTypeErr):
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("body contains incorrect json type for %q at %d", unmarshalTypeErr.Field, unmarshalTypeErr.Offset)})
				return

			case strings.HasPrefix(err.Error(), "json: unknown field"):
				fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("body contains unknow key %s", fieldname)})
				return

			case errors.As(err, &invalidmarshaltype):
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error unmarshaling json: %s", invalidmarshaltype.Error())})
				return

			default:
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("error happend: %s", err.Error())})
				return

			}
		}

		dec := json.NewDecoder(c.Request.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(&struct{}{})
		if err != io.EOF {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body must contain only one json value"})
			return
		}

		validate := validator.New()
		err = validate.RegisterValidation("passwod_strength", PasswordValidator)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to register validation"})
		}
		err = validate.Struct(u)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func ValidationIDParam() gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")
		intid, err := strconv.Atoi(id)
		if err != nil || intid <= 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid id parameter; must be a positive integer"})
			return
		}
		c.Set("id", intid)
		c.Next()

	}
}

func RateLimiterMiddelware(ipratelimiter *adapters.IPRateLimiter, capacity, fillrate float64, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		log := logger.With(slog.String("service", "rate_limit"), slog.String("request_id", c.GetString("RequestID")))
		if ip == "" {
			log.Warn("extract_user_ip", slog.String("reason", "invalid_user_ip"))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid ip"})
			return
		}
		rateLimiter := ipratelimiter.RequestRateLimiter(ip, capacity, fillrate)

		if !rateLimiter.AllowRequest() {
			log.Warn("rate_limit_check", slog.String("reason", "rate_limit_exceeded"), slog.String("user_ip", ip))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate Limit Exceeded"})
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
