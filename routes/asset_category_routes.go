package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAssetCategoryRoutes(rg *gin.RouterGroup) {
	categories := rg.Group("/asset-categories")
	categories.Use(middleware.AuthMiddleware())
	{
		// Public - anyone can view
		categories.GET("", controllers.GetAllAssetCategories)
		categories.GET("/:id", controllers.GetAssetCategoryByID)

		// Admin only - manage categories
		adminCategories := categories.Group("")
		adminCategories.Use(middleware.RequireRole("admin"))
		{
			adminCategories.POST("", controllers.CreateAssetCategory)
			adminCategories.PUT("/:id", controllers.UpdateAssetCategory)
			adminCategories.DELETE("/:id", controllers.DeleteAssetCategory)
		}
	}
}
