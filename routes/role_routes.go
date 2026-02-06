package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoleRoutes(rg *gin.RouterGroup) {
	roles := rg.Group("/roles")
	roles.Use(middleware.AuthMiddleware())
	{
		// Admin and manager can view roles
		roles.GET("", middleware.RequireRole("admin", "manager"), controllers.GetAllRoles)
		roles.GET("/:id", middleware.RequireRole("admin", "manager"), controllers.GetRoleByID)

		// Admin only routes
		adminRoutes := roles.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.POST("", controllers.CreateRole)
			adminRoutes.PUT("/:id", controllers.UpdateRole)
			adminRoutes.DELETE("/:id", controllers.DeleteRole)
		}
	}
}
