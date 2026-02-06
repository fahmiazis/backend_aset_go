package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupMenuRoutes(rg *gin.RouterGroup) {
	menus := rg.Group("/menus")
	menus.Use(middleware.AuthMiddleware())
	{
		// Sidebar - accessible by all authenticated users
		menus.GET("/sidebar", controllers.GetSidebarMenus)

		// Admin only routes
		adminRoutes := menus.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.GET("", controllers.GetAllMenus)
			adminRoutes.POST("", controllers.CreateMenu)
			adminRoutes.GET("/:id", controllers.GetMenuByID)
			adminRoutes.PUT("/:id", controllers.UpdateMenu)
			adminRoutes.DELETE("/:id", controllers.DeleteMenu)
		}
	}

	// Role-Menu assignment routes (under /roles)
	roles := rg.Group("/roles")
	roles.Use(middleware.AuthMiddleware())
	{
		// Accessible by admin and manager
		roles.GET("/:id/menus", middleware.RequireRole("admin", "manager"), controllers.GetRoleMenus)

		// Admin only
		adminRoleRoutes := roles.Group("")
		adminRoleRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoleRoutes.POST("/:id/menus", controllers.AssignMenusToRole)
		}
	}
}
