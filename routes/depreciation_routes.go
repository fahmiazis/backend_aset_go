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

		// Tombol "Run Depreciation" di halaman Asset. Hak aksesnya di menu
		// permission "Asset Run Depreciation" (route_path /depreciation/calculate).
		depreciation.POST("/calculate",
			middleware.RequirePermission("run_depreciation"),
			controllers.CalculateMonthlyDepreciation)

		// Untuk menyembunyikan tombol: sidebar tidak memuat menu bertipe
		// permission, jadi frontend bertanya langsung.
		depreciation.GET("/calculate/allowed", controllers.CanRunDepreciation)

		adminDepr := depreciation.Group("")
		adminDepr.Use(middleware.RequireRole("admin"))
		{
			adminDepr.POST("/calculations/lock", controllers.LockMonthlyDepreciation)
		}
	}
}
