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

		// Katalog permission — dipakai picker hak akses role
		menus.GET("/permissions", controllers.GetPermissionCatalog)

		// Admin only routes
		adminRoutes := menus.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.GET("", controllers.GetAllMenus)
			adminRoutes.POST("", controllers.CreateMenu)

			// Didaftarkan sebelum /:id supaya "reorder" tidak tertangkap sebagai ID
			adminRoutes.PUT("/reorder", controllers.ReorderMenus)

			// Tambah hak akses baru ke master (mode development)
			adminRoutes.POST("/permissions", controllers.AddPermission)

			adminRoutes.GET("/:id", controllers.GetMenuByID)
			adminRoutes.PUT("/:id", controllers.UpdateMenu)
			adminRoutes.DELETE("/:id", controllers.DeleteMenu)

			// Atur hak akses mana yang relevan untuk sebuah menu
			adminRoutes.PUT("/:id/permissions", controllers.SetMenuPermissions)
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
