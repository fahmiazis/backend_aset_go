package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// Report transaksi, dibatasi cabang user (lihat transaction_report_service.go).
// route_path menu: /reports/procurement, /reports/mutation, /reports/disposal.
// Tambah ?format=xlsx untuk mengunduh excel dengan filter yang sama.
func SetupReportRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/reports")
	reports.Use(middleware.AuthMiddleware())
	{
		reports.GET("/procurement", middleware.RequirePermission("read"), controllers.GetProcurementReport)
		reports.GET("/mutation", middleware.RequirePermission("read"), controllers.GetMutationReport)
		reports.GET("/disposal", middleware.RequirePermission("read"), controllers.GetDisposalReport)
	}
}
