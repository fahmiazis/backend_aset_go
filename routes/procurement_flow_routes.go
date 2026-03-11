package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupProcurementFlowRoutes(rg *gin.RouterGroup) {
	procurement := rg.Group("/transactions/procurement")
	procurement.Use(middleware.AuthMiddleware())
	{
		// ============================================================
		// Detail with Stage + Approval Status + GR Status
		// GET /transactions/procurement/detail?transaction_number=xxx
		// ============================================================
		procurement.GET("/detail", controllers.GetProcurementDetailWithStage)
		procurement.GET("/approval-status", controllers.GetProcurementApprovalStatus)
		procurement.GET("/gr", controllers.GetProcurementGRStatus)

		// ============================================================
		// Flow Actions
		// Semua pakai query param: ?transaction_number=xxx
		// karena nomor transaksi mengandung "/" (contoh: TRX/JKT/2026/0001)
		// ============================================================

		// DRAFT → VERIFIKASI_ASET
		procurement.POST("/submit",
			middleware.RequirePermission("create_transaction"),
			controllers.SubmitProcurement)

		// VERIFIKASI_ASET → APPROVAL
		procurement.POST("/verify",
			middleware.RequirePermission("verify_asset"),
			controllers.VerifyProcurement)

		// APPROVAL
		procurement.POST("/initiate-approval",
			middleware.RequirePermission("manage_approval"),
			controllers.InitiateProcurementApproval)

		procurement.POST("/complete-approval",
			middleware.RequirePermission("manage_approval"),
			controllers.CompleteProcurementApproval)

		// PROSES_BUDGET → EKSEKUSI_ASET
		procurement.POST("/process-budget",
			middleware.RequirePermission("process_budget"),
			controllers.ProcessProcurementBudget)

		// EKSEKUSI_ASET → GR
		procurement.POST("/execute",
			middleware.RequirePermission("execute_asset"),
			controllers.ExecuteProcurementAsset)

		// GR per item
		procurement.POST("/gr",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateAssetGR)

		// REJECT
		procurement.POST("/reject",
			middleware.RequirePermission("reject_transaction"),
			controllers.RejectProcurement)

		// REVISI
		procurement.PUT("/revise",
			middleware.RequirePermission("update_transaction"),
			controllers.ReviseProcurement)
	}
}
