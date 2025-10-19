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
	g.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}), middleware.AddRequestID())

	protectedGroup := g.Group("/api", middleware.AuthenticateMiddleware([]byte(config.UserHandler.Config.JwtSecret)))
	{
		protectedGroup.Handle("DELETE", "/users", middleware.CheckContentType(), middleware.RegisterMiddelware[domain.User](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.DeleteUserHandler)
		protectedGroup.Handle("POST", "/stories/generate", middleware.RateLimiterMiddelware(config.UserHandler.IpRateLimiter,
			config.UserHandler.Config.RataLimitCapacity, config.UserHandler.Config.RataLimitFillRate),
			config.UserHandler.ImageGenerationHandler)
	}

	api := g.Group("/api", middleware.CheckContentType(), middleware.RateLimiterMiddelware(config.UserHandler.IpRateLimiter,
		config.UserHandler.Config.RataLimitCapacity, config.UserHandler.Config.RataLimitFillRate))
	{

		api.Handle("POST", "/users/register", middleware.RegisterMiddelware[domain.RegisteredUser](config.UserHandler.Config.MaxAllowedSize), middleware.AddCorrelationID(), config.UserHandler.RegisterHandler)
		api.Handle("POST", "/users/verify", middleware.RegisterMiddelware[domain.RegisterVerify](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.VerificationHandler)
		api.Handle("POST", "/users/login", middleware.RegisterMiddelware[domain.LoginUser](config.UserHandler.Config.MaxAllowedSize), config.UserHandler.LoginHandler)
	}

	g.Handle("GET", "/api/home", config.UserHandler.HomePageHandler)

	return g

}
