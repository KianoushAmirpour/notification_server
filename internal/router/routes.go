package router

import (
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/handler"
	"github.com/KianoushAmirpour/notification_server/internal/middleware"
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
			config.UserHandler.Config.RataLimitCapacity,
			config.UserHandler.Config.RataLimitFillRate,
			config.UserHandler.Logger),
		middleware.AddRequestID(),
		middleware.LoggingRequestMiddleware(config.UserHandler.Logger),
		middleware.CheckContentType(),
	)

	// protected routes
	protected := g.Group("")
	protected.Use(middleware.AuthenticateMiddleware([]byte(config.UserHandler.Config.JwtSecret)))
	{
		protected.Handle("DELETE", "/users", middleware.CheckContentBody[domain.User](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.DeleteUserHandler)
		protected.Handle("POST", "/stories", config.UserHandler.StoryGenerationHandler)
	}

	// auth and register routes
	auth := g.Group("/auth")
	{
		auth.Handle("POST", "/register", middleware.CheckContentBody[domain.RegisteredUser](config.UserHandler.Config.MaxAllowedSize), middleware.AddCorrelationID(), config.UserHandler.RegisterHandler)
		auth.Handle("POST", "/verify", middleware.CheckContentBody[domain.RegisterVerify](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.VerificationHandler)
		auth.Handle("POST", "/login", middleware.CheckContentBody[domain.LoginUser](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.LoginHandler)
	}

	// public routes
	g.Handle("GET", "/home", config.UserHandler.HomePageHandler)
	g.Handle("GET", "/health", config.UserHandler.HealthHandler)

	return g

}
