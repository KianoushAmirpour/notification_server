package router

import (
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/handler"
	"github.com/KianoushAmirpour/notification_server/internal/middelware"
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
	}), middelware.AddRequestID())

	protectedGroup := g.Group("/api", middelware.AuthenticateMiddleware([]byte(config.UserHandler.Config.JwtSecret)))
	{
		protectedGroup.Handle("DELETE", "/users", middelware.CheckContentType(), middelware.RegisterMiddelware[domain.User](), config.UserHandler.DeleteUserHandler)
		protectedGroup.Handle("POST", "/stories/generate", middelware.RateLimiterMiddelware(config.UserHandler.IpRateLimiter,
			config.UserHandler.Config.RataLimitCapacity, config.UserHandler.Config.RataLimitFillRate),
			config.UserHandler.ImageGenerationHandler)
	}

	api := g.Group("/api", middelware.CheckContentType(), middelware.RateLimiterMiddelware(config.UserHandler.IpRateLimiter,
		config.UserHandler.Config.RataLimitCapacity, config.UserHandler.Config.RataLimitFillRate))
	{

		api.Handle("POST", "/users/register", middelware.RegisterMiddelware[domain.RegisteredUser](), middelware.AddCorrelationID(), config.UserHandler.RegisterHandler)
		api.Handle("POST", "/users/verify", middelware.RegisterMiddelware[domain.RegisterVerify](), config.UserHandler.VerificationHandler)
		api.Handle("POST", "/users/login", middelware.RegisterMiddelware[domain.LoginUser](), config.UserHandler.LoginHandler)
	}

	g.Handle("GET", "/api/home", config.UserHandler.HomePageHandler)

	return g

}
