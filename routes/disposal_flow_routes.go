package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupDisposalFlowRoutes(rg *gin.RouterGroup) {
	disposal := rg.Group("/transactions/disposal")
	disposal.Use(middleware.AuthMiddleware())
	{
		// ============================================================
		// DRAFT MANAGEMENT
		// POST   /transactions/disposal                             → create draft
		// GET    /transactions/disposal                             → list semua disposal
		// GET    /transactions/disposal/detail?transaction_number   → detail + assets + stages
		// ============================================================

		disposal.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateDisposalDraft)

		disposal.GET("", controllers.GetAllDisposals)

		disposal.GET("/detail", controllers.GetDisposalDetail)

		disposalDraft := disposal.Group("/draft")
		{
			// POST   /transactions/disposal/draft/add-asset?transaction_number
			// DELETE /transactions/disposal/draft/remove-asset?transaction_number
			// POST   /transactions/disposal/draft/submit?transaction_number
			disposalDraft.POST("/add-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.AddAssetToDisposal)

			disposalDraft.DELETE("/remove-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.RemoveAssetFromDisposal)

			disposalDraft.POST("/submit",
				middleware.RequirePermission("create_transaction"),
				controllers.SubmitDisposal)
		}

		// ============================================================
		// PURCHASING (SELL only)
		// POST /transactions/disposal/purchasing/set-sale-values?transaction_number
		// ============================================================

		disposalPurchasing := disposal.Group("/purchasing")
		{
			disposalPurchasing.POST("/set-sale-values",
				middleware.RequirePermission("manage_purchasing"),
				controllers.SetDisposalSaleValues)
		}

		// ============================================================
		// APPROVAL REQUEST
		// POST /transactions/disposal/approval-request/initiate?transaction_number
		// GET  /transactions/disposal/approval-request/status?transaction_number
		// ============================================================

		disposalApprovalRequest := disposal.Group("/approval-request")
		{
			disposalApprovalRequest.POST("/initiate",
				middleware.RequirePermission("manage_approval"),
				controllers.InitiateDisposalApprovalRequest)

			disposalApprovalRequest.GET("/status",
				controllers.GetDisposalApprovalRequestStatus)
		}

		// ============================================================
		// APPROVAL AGREEMENT
		// POST /transactions/disposal/approval-agreement/initiate?transaction_number
		// GET  /transactions/disposal/approval-agreement/status?transaction_number
		// ============================================================

		disposalApprovalAgreement := disposal.Group("/approval-agreement")
		{
			disposalApprovalAgreement.POST("/initiate",
				middleware.RequirePermission("manage_approval"),
				controllers.InitiateDisposalApprovalAgreement)

			disposalApprovalAgreement.GET("/status",
				controllers.GetDisposalApprovalAgreementStatus)
		}

		// ============================================================
		// EXECUTE — creator upload dok penghapusan / hasil jual
		// POST /transactions/disposal/execute?transaction_number
		// ============================================================

		disposal.POST("/execute",
			middleware.RequirePermission("execute_disposal"),
			controllers.ExecuteDisposal)

		// ============================================================
		// FINANCE (SELL only) — validasi + upload
		// POST /transactions/disposal/finance/confirm?transaction_number
		// ============================================================

		disposalFinance := disposal.Group("/finance")
		{
			disposalFinance.POST("/confirm",
				middleware.RequirePermission("manage_finance"),
				controllers.ConfirmDisposalFinance)
		}

		// ============================================================
		// TAX (SELL only) — validasi + upload
		// POST /transactions/disposal/tax/confirm?transaction_number
		// ============================================================

		disposalTax := disposal.Group("/tax")
		{
			disposalTax.POST("/confirm",
				middleware.RequirePermission("manage_tax"),
				controllers.ConfirmDisposalTax)
		}

		// ============================================================
		// ASSET DELETION — tim asset hapus + generate doc number
		// POST /transactions/disposal/asset-deletion/confirm?transaction_number
		// ============================================================

		disposalAssetDeletion := disposal.Group("/asset-deletion")
		{
			disposalAssetDeletion.POST("/confirm",
				middleware.RequirePermission("execute_asset_deletion"),
				controllers.ConfirmDisposalAssetDeletion)
		}

		// ============================================================
		// REJECT
		// POST /transactions/disposal/reject?transaction_number
		// ============================================================

		disposal.POST("/reject",
			middleware.RequirePermission("reject_transaction"),
			controllers.RejectDisposal)

		// ============================================================
		// ATTACHMENT PER ASSET PER STAGE
		// POST /transactions/disposal/attachments/upload?transaction_number
		// PUT  /transactions/disposal/attachments/:id/review
		// GET  /transactions/disposal/attachments/status?transaction_number&stage
		// ============================================================

		disposal.POST("/attachments/upload",
			middleware.RequirePermission("upload_attachment"),
			controllers.UploadDisposalAttachment)

		disposal.PUT("/attachments/:id/review",
			middleware.RequirePermission("review_attachment"),
			controllers.ReviewDisposalAttachment)

		disposal.GET("/attachments/status",
			controllers.GetDisposalAttachmentStatus)
	}
}
