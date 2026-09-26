package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// Tanpa RequirePermission: halaman dashboard terbuka untuk semua user login,
// isinya sudah dibatasi cabang user di service.
func SetupDashboardRoutes(rg *gin.RouterGroup) {
	dashboard := rg.Group("/dashboard")
	dashboard.Use(middleware.AuthMiddleware())
	{
		dashboard.GET("/summary", controllers.GetDashboardSummary)
	}
}
