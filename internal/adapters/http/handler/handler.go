package handler

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/dto"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/middleware"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/utils"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/usecase"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	UserSvc           *usecase.UserService
	ImageSvc          *usecase.StoryGenerationService
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
	imgsvc *usecase.StoryGenerationService,
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

func (h *UserHandler) HomePageHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Message": "Welcome"})
}

func (h *UserHandler) RegisterHandler(c *gin.Context) {

	req := c.MustGet("payload").(dto.RegisteredUser)
	reqID := c.GetString("CorrelationID")

	reqRu := domain.RegisteredUser{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		Preferences: req.Preferences,
	}

	res, err := h.UserSvc.RegisterUser(c.Request.Context(), reqRu, reqID, h.OtpExpiration)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"Message": res.Message})
}

func (h *UserHandler) VerificationHandler(c *gin.Context) {

	req := c.MustGet("payload").(dto.RegisterVerify)
	reqID := c.GetHeader("X-Correlation-Id")

	reqVu := domain.RegisterVerify{SentOtpbyUser: req.SentOtpbyUser}

	resp, err := h.UserSvc.VerifyUser(c.Request.Context(), reqVu, reqID)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		// c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}
	h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"), "status", http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"Message": resp.Message})

}

func (h *UserHandler) LoginHandler(c *gin.Context) {
	req := c.MustGet("payload").(dto.LoginUser)

	reqLu := domain.LoginUser{
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := h.UserSvc.AuthenticateUser(c.Request.Context(), reqLu)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		// c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}
	c.Header("Authorization", "Bearer "+resp.AccessToken)
	c.Header("X-Refresh-Token", resp.RefreshToken)
	h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"), "status", http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"Message": "You are logged in"})

	// c.SetCookie("access-token", token, 3600, "/", "", true, true)
}

func (h *UserHandler) JwtRefreshHandler(c *gin.Context) {
	refreshToken := c.GetHeader("X-Refresh-Token")
	resp, err := h.UserSvc.RefreshJwtToken(c.Request.Context(), refreshToken, h.JwtRefresh)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	c.Header("Authorization", "Bearer "+resp.AccessToken)
	c.Header("X-Refresh-Token", resp.RefreshToken)
	h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"), "status", http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"Message": "You are logged in"})

}

func (h *UserHandler) DeleteUserHandler(c *gin.Context) {
	req := c.MustGet("payload").(dto.User)

	reqU := domain.User{
		ID:    req.ID,
		Email: req.Email,
	}

	resp, err := h.UserSvc.DeleteUser(c.Request.Context(), reqU)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"), "status", http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"Message": fmt.Sprintf("%s. Good buy🙌", resp.Message)})

}

func (h *UserHandler) StoryGenerationHandler(c *gin.Context) {

	userID := c.GetInt("user_id")
	resp, err := h.ImageSvc.GenerateStory(c.Request.Context(), userID)
	if err != nil {
		httpErr := utils.MapErr(err)
		c.JSON(httpErr.StatusCode, httpErr)
		return
	}
	h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"), "status", http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"Message": resp.Message})

}

func (h *UserHandler) HealthHandler(c *gin.Context) {

	status := http.StatusOK

	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			h.Logger.Info("http_request_end", "request_id", c.GetString("RequestID"),
				"reason", "canceled_context",
				"status", http.StatusCreated)
			return
		case <-time.After(5 * time.Second):
			http.ListenAndServe("localhost:6060", nil)
		}

	}(c.Request.Context())

	c.JSON(status, gin.H{"status": gin.H{"status code": status}, "memory": gin.H{"allocated heap objects (MB)": memStat.Alloc / 1024 / 1024,
		"cumulative allocated for heap objects (MB)": memStat.TotalAlloc / 1024 / 1024,
		"total memory obtained from the OS (MB)":     memStat.Sys / 1024 / 1024,
		"number of completed GC cycles":              memStat.NumGC,
		"number of goroutines that currently exist":  runtime.NumGoroutine()},
	},
	)
}
