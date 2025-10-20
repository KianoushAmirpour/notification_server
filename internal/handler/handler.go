package handler

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	UserSvc       *service.UserRegisterService
	ImageSvc      *service.StoryGenerationService
	Config        *config.Config
	IpRateLimiter *adapters.IPRateLimiter
}

func NewUserHandler(usersvc *service.UserRegisterService, imgsvc *service.StoryGenerationService, c *config.Config, i *adapters.IPRateLimiter) *UserHandler {
	return &UserHandler{UserSvc: usersvc, ImageSvc: imgsvc, Config: c, IpRateLimiter: i}
}

func (h *UserHandler) HomePageHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Message": "Welcome"})
}

func (h *UserHandler) RegisterHandler(c *gin.Context) {

	req := c.MustGet("payload").(domain.RegisteredUser)
	reqID := c.GetString("CorrelationID")

	res, err := h.UserSvc.RegisterUser(c.Request.Context(), req, h.Config, reqID)
	if err != nil {
		c.JSON(err.StatusCode, gin.H{"Message": err.Message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"Message": res.Message})
}

func (h *UserHandler) VerificationHandler(c *gin.Context) {

	req := c.MustGet("payload").(domain.RegisterVerify)
	reqID := c.GetHeader("X-Correlation-Id")

	resp, err := h.UserSvc.VerifyUser(c.Request.Context(), req, reqID)
	if err != nil {
		c.JSON(err.StatusCode, gin.H{"Message": err.Message})
		c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"Message": resp.Message})

}

func (h *UserHandler) LoginHandler(c *gin.Context) {
	req := c.MustGet("payload").(domain.LoginUser)

	resp, err := h.UserSvc.AuthenticateUser(c.Request.Context(), req, h.Config)
	if err != nil {
		c.JSON(err.StatusCode, gin.H{"Message": err.Message})
		c.Redirect(http.StatusSeeOther, "http://localhost:4000/api/user/register")
		return
	}
	c.Header("Authorization", "Bearer "+resp.Message)

	c.JSON(http.StatusCreated, gin.H{"Message": "You are logged in"})

	// c.SetCookie("access-token", token, 3600, "/", "", true, true)
}

func (h *UserHandler) DeleteUserHandler(c *gin.Context) {
	req := c.MustGet("payload").(domain.User)

	resp, err := h.UserSvc.DeleteUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(err.StatusCode, gin.H{"Message": err.Message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"Message": fmt.Sprintf("%s. Good buy🙌", resp.Message)})

}

func (h *UserHandler) ImageGenerationHandler(c *gin.Context) {

	userID := c.GetInt("user_id")
	resp, err := h.ImageSvc.GenerateStory(c.Request.Context(), userID)
	if err != nil {
		c.JSON(err.StatusCode, gin.H{"Message": err.Message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"Message": resp.Message})

}

func (h *UserHandler) HealthHandler(c *gin.Context) {

	status := http.StatusOK

	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	var dbhealthy bool = true
	dbErr := h.UserSvc.DbPool.Ping(ctx)
	if dbErr != nil {
		dbhealthy = false
		status = http.StatusServiceUnavailable
	}

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	c.JSON(status, gin.H{"status": gin.H{"status code": status}, "memory": gin.H{"allocated heap objects (MB)": memStat.Alloc / 1024 / 1024,
		"cumulative allocated for heap objects (MB)": memStat.TotalAlloc / 1024 / 1024,
		"total memory obtained from the OS (MB)":     memStat.Sys / 1024 / 1024,
		"number of completed GC cycles":              memStat.NumGC,
		"number of goroutines that currently exist":  runtime.NumGoroutine()},
		"database": gin.H{"Postgres health": dbhealthy}},
	)
}
