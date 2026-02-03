package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupCabangRoutes(rg *gin.RouterGroup) {
	// Cabang routes
	cabangs := rg.Group("/cabangs")
	cabangs.Use(middleware.AuthMiddleware()) // All cabang routes require authentication
	{
		// Routes accessible by all authenticated users
		cabangs.GET("", controllers.GetAllCabangs)
		cabangs.GET("/:id", controllers.GetCabangByID)
		cabangs.GET("/:id/users", controllers.GetCabangUsers)

		// Admin only routes
		adminRoutes := cabangs.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.POST("", controllers.CreateCabang)
			adminRoutes.PUT("/:id", controllers.UpdateCabang)
			adminRoutes.DELETE("/:id", controllers.DeleteCabang)
		}
	}

	// User-Cabang assignment routes (under /users)
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		// Routes accessible by all authenticated users
		users.GET("/:id/cabangs", controllers.GetUserCabangs)

		// Admin only routes
		adminUserRoutes := users.Group("")
		adminUserRoutes.Use(middleware.RequireRole("admin"))
		{
			adminUserRoutes.POST("/:id/cabangs", controllers.AssignCabangsToUser)
			adminUserRoutes.DELETE("/:id/cabangs/:cabang_id", controllers.RemoveCabangFromUser)
		}
	}
}
