package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupDepreciationRoutes(rg *gin.RouterGroup) {
	routes := rg.Group("")
	routes.Use(middleware.AuthMiddleware())

	// Depreciation Settings
	settings := routes.Group("/depreciation-settings")
	{
		settings.GET("", controllers.GetAllDepreciationSettings)
		settings.GET("/:id", controllers.GetDepreciationSettingByID)

		adminSettings := settings.Group("")
		adminSettings.Use(middleware.RequireRole("admin"))
		{
			adminSettings.POST("", controllers.CreateDepreciationSetting)
			adminSettings.PUT("/:id", controllers.UpdateDepreciationSetting)
			adminSettings.DELETE("/:id", controllers.DeleteDepreciationSetting)
		}
	}

	// Monthly Depreciation Calculations
	depreciation := routes.Group("/depreciation")
	{
		depreciation.GET("/monthly", controllers.GetMonthlyDepreciationCalculations)

		adminDepr := depreciation.Group("")
		adminDepr.Use(middleware.RequireRole("admin"))
		{
			adminDepr.POST("/calculate", controllers.CalculateMonthlyDepreciation)
		}
	}
}
