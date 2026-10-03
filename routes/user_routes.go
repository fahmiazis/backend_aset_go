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

			// Upload Excel: mode=new | update (mass update)
			adminRoutes.GET("/import/template", controllers.UserImportTemplate)
			adminRoutes.POST("/import", controllers.ImportUsers)
			// unduh seluruh user dalam format template mass update
			adminRoutes.GET("/export", controllers.ExportUsers)
			adminRoutes.PUT("/:id", controllers.UpdateUser)
			adminRoutes.DELETE("/:id", controllers.DeleteUser)
			adminRoutes.POST("/:id/roles", controllers.AssignRoles)
			adminRoutes.POST("/:id/avatar", controllers.UploadUserAvatar)
			adminRoutes.DELETE("/:id/avatar", controllers.DeleteUserAvatar)
		}

		// User can view their own profile (handled in auth routes /auth/me)
		// Or admin/manager can view specific user
		users.GET("/:id", middleware.RequireRole("admin", "manager"), controllers.GetUserByID)

		// Foto profil boleh dilihat semua user yang login (navbar, profil, dsb)
		users.GET("/:id/avatar", controllers.ServeUserAvatar)
	}
}
