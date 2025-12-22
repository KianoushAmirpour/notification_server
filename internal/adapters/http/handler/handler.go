package handler

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/dto"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/middleware"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/observability"
	"github.com/KianoushAmirpour/notification_server/internal/usecase"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	UserSvc           *usecase.UserService
	ImageSvc          *usecase.StorySchedulerService
	IpRateLimiter     *middleware.IPRateLimiter
	JwtHandler        domain.JwtTokenRepository
	Logger            domain.LoggingRepository
	OtpExpiration     int
	JwtIss            string
	JwtSecret         string
	JwtRefresh        string
	RateLimitCapacity float64
	RateLimitFillRate float64
	MaxAllowedSize    int
}

func NewUserHandler(
	usersvc *usecase.UserService,
	imgsvc *usecase.StorySchedulerService,
	i *middleware.IPRateLimiter,
	auth domain.JwtTokenRepository,
	logger domain.LoggingRepository,
	otpexpiration int,
	jwtiss string,
	jwtsecret string,
	jwtrefresh string,
	ratelimitcapacity float64,
	ratelimitfillrate float64,
	maxallowedsize int,
) *UserHandler {
	return &UserHandler{UserSvc: usersvc, ImageSvc: imgsvc, IpRateLimiter: i, JwtHandler: auth, Logger: logger,
		OtpExpiration: otpexpiration, JwtIss: jwtiss, JwtSecret: jwtsecret, JwtRefresh: jwtrefresh,
		RateLimitCapacity: ratelimitcapacity, RateLimitFillRate: ratelimitfillrate, MaxAllowedSize: maxallowedsize}
}

// HomePageHandler godoc
// @Summary Home page
// @Description Returns a welcome message
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string "Welcome message"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /home [get]
func (h *UserHandler) HomePageHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Message": "Welcome"})
}

// RegisterHandler godoc
// @Summary Register a new user
// @Description Registers a user with first name, last name, email, password, and preferences
// @Tags Users
// @Accept json
// @Produce json
// @Param X-Correlation-Id header string true "Correlation ID for request tracing"
// @Param request body dto.RegisteredUser true "User registration payload"
// @Success 201 {object} map[string]string "User registered successfully"
// @Failure 400 {object} dto.HttpError "Bad request"
// @Failure 409 {object} dto.HttpError "User already exists"
// @Failure 413 {object} dto.HttpError "Payload too large"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Failure 503 {object} dto.HttpError "Service unavailable"
// @Router /users/register [post]
func (h *UserHandler) RegisterHandler(c *gin.Context) {

	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "register",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"iam", "web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", []string{"start"},
	)
	req := c.MustGet("payload").(dto.RegisteredUser)

	reqRu := domain.RegisteredUser{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		Preferences: req.Preferences,
	}

	res, err := h.UserSvc.RegisterUser(c.Request.Context(), reqRu, h.OtpExpiration)
	var duration time.Duration
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.register",
			"event.outcome", "failure",
			"event.type", []string{"error", "end"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}

	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "creation"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)

	c.JSON(http.StatusCreated, gin.H{"Message": res.Message})
}

// VerificationHandler godoc
// @Summary Verify a user
// @Description Verifies a user using a one-time password (OTP)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param X-Correlation-Id header string true "Correlation ID for request tracing"
// @Param request body dto.RegisterVerify true "User verification payload"
// @Success 200 {object} map[string]string "User verified successfully"
// @Failure 400 {object} dto.HttpError "Bad request"
// @Failure 404 {object} dto.HttpError "Verification data not found"
// @Failure 413 {object} dto.HttpError "Payload too large"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /auth/verify [post]
func (h *UserHandler) VerificationHandler(c *gin.Context) {
	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "verification",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"iam", "web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", []string{"start"},
	)

	req := c.MustGet("payload").(dto.RegisterVerify)
	registerReqID := c.GetHeader("X-Request-Id")

	reqVu := domain.RegisterVerify{SentOtpbyUser: req.SentOtpbyUser}

	resp, err := h.UserSvc.VerifyUser(c.Request.Context(), reqVu, registerReqID)
	var duration time.Duration
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.verify",
			"event.outcome", "failure",
			"event.type", []string{"error", "end", "denied"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		// c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}
	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "allowed"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)
	c.JSON(http.StatusOK, gin.H{"Message": resp.Message})

}

// LoginHandler godoc
// @Summary Login a user
// @Description Authenticates a user and returns access and refresh tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginUser true "User login payload"
// @Success 200 {object} map[string]string "User logged in successfully"
// @Failure 400 {object} dto.HttpError "Bad request"
// @Failure 401 {object} dto.HttpError "Unauthorized"
// @Failure 413 {object} dto.HttpError "Payload too large"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 404 {object} dto.HttpError "User not found"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /auth/login [post]
func (h *UserHandler) LoginHandler(c *gin.Context) {
	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "login",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"authentication", "web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", []string{"start"},
	)
	req := c.MustGet("payload").(dto.LoginUser)

	reqLu := domain.LoginUser{
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := h.UserSvc.AuthenticateUser(c.Request.Context(), reqLu)
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	var duration time.Duration
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.authenticate",
			"event.outcome", "failure",
			"event.type", []string{"error", "denied", "end"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		// c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}
	c.Header("Authorization", "Bearer "+resp.AccessToken)
	c.Header("X-Refresh-Token", resp.RefreshToken)
	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "allowed"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)
	c.JSON(http.StatusOK, gin.H{"Message": "You are logged in"})

	// c.SetCookie("access-token", token, 3600, "/", "", true, true)
}

// JwtRefreshHandler godoc
// @Summary Refresh JWT token
// @Description Refreshes access and refresh tokens using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param X-Refresh-Token header string true "Refresh token"
// @Success 201 {object} map[string]string "JWT tokens refreshed successfully"
// @Failure 400 {object} dto.HttpError "Bad request"
// @Failure 401 {object} dto.HttpError "Unauthorized"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 404 {object} dto.HttpError "Token not found"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /refresh [post]
func (h *UserHandler) JwtRefreshHandler(c *gin.Context) {
	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "jwt-refresh",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"authentication", "web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", "start",
	)
	refreshToken := c.GetHeader("X-Refresh-Token")
	resp, err := h.UserSvc.RefreshJwtToken(c.Request.Context(), refreshToken, h.JwtRefresh)
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	var duration time.Duration
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.refreshjwt",
			"event.outcome", "failure",
			"event.type", []string{"error", "denied", "end"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	c.Header("Authorization", "Bearer "+resp.AccessToken)
	c.Header("X-Refresh-Token", resp.RefreshToken)
	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "allowed"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)
	c.JSON(http.StatusCreated, gin.H{"Message": "You are logged in"})

}

// DeleteUserHandler godoc
// @Summary Delete a user
// @Description Deletes a user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.DeleteUser true "User deletion payload"
// @Success 200 {object} map[string]string "User deleted successfully"
// @Failure 400 {object} dto.HttpError "Bad request"
// @Failure 401 {object} dto.HttpError "Unauthorized"
// @Failure 413 {object} dto.HttpError "Payload too large"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 404 {object} dto.HttpError "User not found"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /users [delete]
func (h *UserHandler) DeleteUserHandler(c *gin.Context) {
	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "deletion",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"iam", "web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", "start",
	)
	req := c.MustGet("payload").(dto.DeleteUser)

	reqU := domain.DeleteUser{
		ID: req.UserID,
	}

	resp, err := h.UserSvc.DeleteUser(c.Request.Context(), reqU)
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	var duration time.Duration
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.delete",
			"event.outcome", "failure",
			"event.type", []string{"error", "denied", "end"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "deletion"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)
	c.JSON(http.StatusOK, gin.H{"Message": fmt.Sprintf("%s. Good buy🙌", resp.Message)})

}

// StoryGenerationHandler godoc
// @Summary Generate a story
// @Description Schedules story generation for a user
// @Tags Stories
// @Accept json
// @Produce json
// @Success 201 {object} map[string]string "Story generation scheduled successfully"
// @Failure 401 {object} dto.HttpError "Unauthorized"
// @Failure 404 {object} dto.HttpError "User not found"
// @Failure 413 {object} dto.HttpError "Payload too large"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Failure 503 {object} dto.HttpError "External service error"
// @Router /stories [post]
func (h *UserHandler) StoryGenerationHandler(c *gin.Context) {
	reqID := observability.GetRequestID(c.Request.Context())
	log := h.Logger.With(
		"service.name", "story",
		"@timestamp", time.Now().UTC().Format(time.RFC3339Nano),
		"event.category", []string{"web"},
		"http.request.method", c.Request.Method,
		"http.request.path", c.FullPath(),
		"http.request.agent", c.Request.UserAgent(),
		"http.request.id", reqID,
		"client.ip", c.ClientIP(),
	)

	log.Info(
		"http request started",
		"event.action", "http.request",
		"event.type", "start",
	)
	userID := c.GetInt("user_id")
	resp, err := h.ImageSvc.ScheduleStoryGeneration(c.Request.Context(), userID)
	t, b := observability.GetrequestStartTimeKey(c.Request.Context())
	var duration time.Duration
	if b {
		duration = time.Since(t)
	}
	if err != nil {
		httpErr := dto.MapErr(err)
		log.Error(
			"http request failed",
			"event.action", "http.request.story",
			"event.outcome", "failure",
			"event.type", []string{"error", "end"},
			"http.response.status_code", httpErr.StatusCode,
			"error.message", err.Error(),
			"error.code", httpErr.Code,
			"event.duration", duration.Nanoseconds(),
		)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	log.Info(
		"http request completed",
		"event.action", "http.request",
		"event.outcome", "success",
		"event.type", []string{"end", "creation"},
		"http.response.status_code", http.StatusCreated,
		"event.duration", duration.Nanoseconds(),
	)
	c.JSON(http.StatusCreated, gin.H{"Message": resp.Message})

}

// HealthHandler godoc
// @Summary Check server health
// @Description Returns server status and runtime memory statistics
// @Tags System
// @Produce json
// @Success 200 {object} dto.HealthResponse "Health check response"
// @Failure 429 {object} dto.HttpError "Too many verification attempts"
// @Failure 500 {object} dto.HttpError "Internal server error"
// @Router /health [get]
func (h *UserHandler) HealthHandler(c *gin.Context) {

	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)

	resp := dto.HealthResponse{
		Status: struct {
			StatusCode int `json:"status_code"`
		}{StatusCode: http.StatusOK},
		Memory: struct {
			AllocMB      uint64 `json:"allocated_heap_objects_MB"`
			TotalAllocMB uint64 `json:"cumulative_allocated_MB"`
			SysMB        uint64 `json:"total_memory_from_OS_MB"`
			NumGC        uint32 `json:"gc_cycles"`
			NumGoroutine int    `json:"num_goroutines"`
		}{
			AllocMB:      memStat.Alloc / 1024 / 1024,
			TotalAllocMB: memStat.TotalAlloc / 1024 / 1024,
			SysMB:        memStat.Sys / 1024 / 1024,
			NumGC:        memStat.NumGC,
			NumGoroutine: runtime.NumGoroutine(),
		},
	}

	c.JSON(http.StatusOK, resp)
}
