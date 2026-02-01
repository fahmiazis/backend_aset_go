package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		// Public routes
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/refresh", controllers.RefreshToken)

		// Protected routes (require authentication)
		authenticated := auth.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			authenticated.GET("/me", controllers.GetProfile)
			authenticated.POST("/logout", controllers.Logout)
			authenticated.POST("/logout-all", controllers.LogoutAllDevices)
		}
	}
}
