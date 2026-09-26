package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupStockOpnameFlowRoutes(rg *gin.RouterGroup) {
	stockOpname := rg.Group("/transactions/stock-opname")
	stockOpname.Use(middleware.AuthMiddleware())
	{
		// POST   /transactions/stock-opname                          → create draft
		// GET    /transactions/stock-opname                          → list semua stock opname user
		// GET    /transactions/stock-opname/detail?transaction_number → detail + items + stages
		stockOpname.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateStockOpnameDraft)

		stockOpname.GET("", controllers.GetAllStockOpnamesFlow)

		stockOpname.GET("/detail", controllers.GetStockOpnameFlowDetail)

		// GET /transactions/stock-opname/photo/:id/file → serve foto (inline), gak dibatasi stage
		stockOpname.GET("/photo/:id/file", controllers.ServeStockOpnameAssetPhoto)

		// GET /transactions/stock-opname/borrow-document/:id/file → serve dokumen peminjaman (inline), gak dibatasi stage
		stockOpname.GET("/borrow-document/:id/file", controllers.ServeStockOpnameBorrowDocument)

		// GET/PUT /transactions/stock-opname/config → lihat/ubah jendela tanggal submit
		// Belum ada pembatasan role/permission (nyusul) — cukup login.
		stockOpname.GET("/config", controllers.GetStockOpnameConfig)
		stockOpname.PUT("/config", controllers.UpdateStockOpnameConfig)

		stockOpnameDraft := stockOpname.Group("/draft")
		{
			// PUT /transactions/stock-opname/draft/update-finding?transaction_number → isi/update temuan fisik per asset
			// Asset list SEKARANG otomatis terisi semua asset di branch homebase
			// aktif si creator saat draft dibuat — tidak ada lagi add/remove manual.
			stockOpnameDraft.PUT("/update-finding",
				middleware.RequirePermission("create_transaction"),
				controllers.UpdateStockOpnameFinding)

			// PUT /transactions/stock-opname/draft/bulk-update-finding?transaction_number → autosave grid "Lengkapi Data"
			stockOpnameDraft.PUT("/bulk-update-finding",
				middleware.RequirePermission("create_transaction"),
				controllers.BulkUpdateStockOpnameFinding)

			// POST /transactions/stock-opname/draft/photo/upload?transaction_number → upload/replace foto bukti fisik 1 asset
			stockOpnameDraft.POST("/photo/upload",
				middleware.RequirePermission("create_transaction"),
				controllers.UploadStockOpnameAssetPhoto)

			// POST /transactions/stock-opname/draft/borrow-document/upload?transaction_number → upload/replace dokumen peminjaman (PDF) 1 asset
			stockOpnameDraft.POST("/borrow-document/upload",
				middleware.RequirePermission("create_transaction"),
				controllers.UploadStockOpnameBorrowDocument)

			// GET  /transactions/stock-opname/draft/template/download?transaction_number → download template excel
			// POST /transactions/stock-opname/draft/template/upload?transaction_number   → bulk update temuan dari excel
			stockOpnameDraft.GET("/template/download",
				middleware.RequirePermission("create_transaction"),
				controllers.DownloadStockOpnameTemplate)

			stockOpnameDraft.POST("/template/upload",
				middleware.RequirePermission("create_transaction"),
				controllers.UploadStockOpnameTemplate)

			// POST /transactions/stock-opname/draft/submit?transaction_number → DRAFT → APPROVAL
			stockOpnameDraft.POST("/submit",
				middleware.RequirePermission("create_transaction"),
				controllers.SubmitStockOpname)
		}

		stockOpnameApproval := stockOpname.Group("/approval")
		{
			// POST /transactions/stock-opname/approval/initiate?transaction_number → trigger approval
			stockOpnameApproval.POST("/initiate",
				middleware.RequirePermission("manage_approval"),
				controllers.InitiateStockOpnameApproval)

			// GET /transactions/stock-opname/approval/status?transaction_number → status approval
			stockOpnameApproval.GET("/status", controllers.GetStockOpnameApprovalStatus)
		}

		// POST /transactions/stock-opname/execute?transaction_number → EXECUTE_STOCK_OPNAME → FINISHED
		// Dilakukan oleh role berwenang (mis. PIC Asset) setelah semua approval step disetujui
		stockOpname.POST("/execute",
			middleware.RequirePermission("execute_stock_opname"),
			controllers.ExecuteStockOpname)

		// POST /transactions/stock-opname/reject?transaction_number → REJECTED
		stockOpname.POST("/reject",
			middleware.RequirePermission("reject_transaction"),
			controllers.RejectStockOpname)

		stockOpnameReport := stockOpname.Group("/report")
		{
			// GET /transactions/stock-opname/report/dashboard?month=&year=&branch_code=
			//   → stats card + data chart (status per grouping, fisik vs SAP, kondisi aset, status submit)
			stockOpnameReport.GET("/dashboard", controllers.GetStockOpnameReportDashboard)

			// GET /transactions/stock-opname/report/detail?month=&year=&branch_code=
			//   → tabel rekapitulasi (SAP=FISIK, SAP ADA FISIK TIDAK, dst) + top 10 cost center
			stockOpnameReport.GET("/detail", controllers.GetStockOpnameReportDetail)

			// GET /transactions/stock-opname/report/export?month=&year=&branch_code=
			//   → download file .xlsx (sheet SUMMARY + breakdown per kategori)
			stockOpnameReport.GET("/export", controllers.ExportStockOpnameReport)
		}
	}
}
