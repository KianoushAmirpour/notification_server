package router

import (
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/dto"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/handler"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	UserHandler *handler.UserHandler
}

func SetupRoutes(config RouterConfig) *gin.Engine {

	g := gin.Default()
	g.Use(
		cors.New(cors.Config{
			AllowOrigins:     []string{"https://*", "http://*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		middleware.PanicRecoveryMiddleware(config.UserHandler.Logger),
		middleware.RateLimiterMiddelware(
			config.UserHandler.IpRateLimiter,
			config.UserHandler.RateLimitCapacity,
			config.UserHandler.RateLimitFillRate,
			config.UserHandler.Logger),
		middleware.AddRequestID(),
		middleware.LoggingRequestMiddleware(config.UserHandler.Logger),
	)

	// protected routes
	protected := g.Group("")
	protected.Use(middleware.AuthenticateMiddleware(config.UserHandler.JwtHandler, []byte(config.UserHandler.JwtSecret)))
	{
		protected.Handle("DELETE", "/users", middleware.CheckContentType(), middleware.CheckContentBody[dto.User](config.UserHandler.MaxAllowedSize), config.UserHandler.DeleteUserHandler)
		protected.Handle("POST", "/stories", config.UserHandler.StoryGenerationHandler)
	}

	// auth and register routes
	auth := g.Group("/auth")
	auth.Use(middleware.CheckContentType())
	{
		auth.Handle("POST", "/register", middleware.CheckContentBody[dto.RegisteredUser](config.UserHandler.MaxAllowedSize), middleware.AddCorrelationID(), config.UserHandler.RegisterHandler)
		auth.Handle("POST", "/verify", middleware.CheckContentBody[dto.RegisterVerify](config.UserHandler.MaxAllowedSize), config.UserHandler.VerificationHandler)
		auth.Handle("POST", "/login", middleware.CheckContentBody[dto.LoginUser](config.UserHandler.MaxAllowedSize), config.UserHandler.LoginHandler)
	}

	// public routes
	g.Handle("GET", "/home", config.UserHandler.HomePageHandler)
	g.Handle("GET", "/health", config.UserHandler.HealthHandler)

	return g

}
