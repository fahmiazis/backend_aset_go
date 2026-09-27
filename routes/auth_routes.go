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
			// Profil sendiri — tanpa permission, selalu milik user yang login
			authenticated.PUT("/me/password", controllers.ChangeMyPassword)
			authenticated.POST("/me/avatar", controllers.UploadMyAvatar)
			authenticated.DELETE("/me/avatar", controllers.DeleteMyAvatar)
			authenticated.POST("/logout", controllers.Logout)
			authenticated.POST("/logout-all", controllers.LogoutAllDevices)
		}
	}
}
