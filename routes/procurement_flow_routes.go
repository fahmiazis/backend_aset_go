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
		// ============================================================
		procurement.GET("/:transaction_number/detail",
			controllers.GetProcurementDetailWithStage)

		procurement.GET("/:transaction_number/approval-status",
			controllers.GetProcurementApprovalStatus)

		procurement.GET("/:transaction_number/gr",
			controllers.GetProcurementGRStatus)

		// ============================================================
		// Flow Actions
		// ============================================================

		// DRAFT → VERIFIKASI_ASET
		procurement.POST("/:transaction_number/submit",
			middleware.RequirePermission("create_transaction"),
			controllers.SubmitProcurement)

		// VERIFIKASI_ASET → APPROVAL
		procurement.POST("/:transaction_number/verify",
			middleware.RequirePermission("verify_asset"),
			controllers.VerifyProcurement)

		// APPROVAL
		procurement.POST("/:transaction_number/initiate-approval",
			middleware.RequirePermission("manage_approval"),
			controllers.InitiateProcurementApproval)

		procurement.POST("/:transaction_number/complete-approval",
			middleware.RequirePermission("manage_approval"),
			controllers.CompleteProcurementApproval)

		// PROSES_BUDGET → EKSEKUSI_ASET
		procurement.POST("/:transaction_number/process-budget",
			middleware.RequirePermission("process_budget"),
			controllers.ProcessProcurementBudget)

		// EKSEKUSI_ASET → GR
		procurement.POST("/:transaction_number/execute",
			middleware.RequirePermission("execute_asset"),
			controllers.ExecuteProcurementAsset)

		// GR per item
		procurement.POST("/:transaction_number/gr",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateAssetGR)

		// REJECT
		procurement.POST("/:transaction_number/reject",
			middleware.RequirePermission("reject_transaction"),
			controllers.RejectProcurement)

		// REVISI
		procurement.PUT("/:transaction_number/revise",
			middleware.RequirePermission("update_transaction"),
			controllers.ReviseProcurement)
	}
}
