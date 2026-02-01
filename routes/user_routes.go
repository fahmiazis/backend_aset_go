package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware()) // All user routes require authentication
	{
		// Admin only routes
		adminRoutes := users.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.GET("", controllers.GetAllUsers)
			adminRoutes.POST("", controllers.CreateUser)
			adminRoutes.PUT("/:id", controllers.UpdateUser)
			adminRoutes.DELETE("/:id", controllers.DeleteUser)
			adminRoutes.POST("/:id/roles", controllers.AssignRoles)
		}

		// User can view their own profile (handled in auth routes /auth/me)
		// Or admin/manager can view specific user
		users.GET("/:id", middleware.RequireRole("admin", "manager"), controllers.GetUserByID)
	}
}
