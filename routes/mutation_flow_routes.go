package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupMutationFlowRoutes(rg *gin.RouterGroup) {
	mutation := rg.Group("/transactions/mutation")
	mutation.Use(middleware.AuthMiddleware())
	{
		// ============================================================
		// DRAFT MANAGEMENT
		// ============================================================

		// POST   /transactions/mutation                          → create draft
		// GET    /transactions/mutation                          → list semua mutasi user
		// GET    /transactions/mutation/detail?transaction_number → detail + assets + stages
		mutation.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateMutationDraft)

		mutation.GET("", controllers.GetAllMutations)

		mutation.GET("/detail", controllers.GetMutationDetail)

		mutationDraft := mutation.Group("/draft")
		{
			// POST /transactions/mutation/draft/add-asset?transaction_number → tambah asset ke draft
			// DELETE /transactions/mutation/draft/remove-asset?transaction_number → hapus asset dari draft
			mutationDraft.POST("/add-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.AddAssetToMutation)

			mutationDraft.DELETE("/remove-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.RemoveAssetFromMutation)

			// POST /transactions/mutation/submit?transaction_number → DRAFT → APPROVAL
			mutationDraft.POST("/submit",
				middleware.RequirePermission("create_transaction"),
				controllers.SubmitMutation)

		}

		// ============================================================
		// FLOW ACTIONS
		// ============================================================

		mutationApproval := mutation.Group("/approval")
		{
			// POST /transactions/mutation/approval/initiate-approval?transaction_number → trigger approval
			mutationApproval.POST("/initiate",
				middleware.RequirePermission("manage_approval"),
				controllers.InitiateMutationApproval)

			// GET  /transactions/mutation/approval/status?transaction_number → status approval
			mutationApproval.GET("/status", controllers.GetMutationApprovalStatus)
		}

		// POST /transactions/mutation/confirm-receiving?transaction_number → APPROVAL → MUTATION_RECEIVING → EXECUTE_MUTATION
		// Dilakukan oleh user homebase branch tujuan, upload dok serah terima dulu
		mutation.POST("/confirm-receiving",
			middleware.RequirePermission("confirm_receiving", "create_transaction"),
			controllers.ConfirmMutationReceiving)

		// POST /transactions/mutation/execute?transaction_number → EXECUTE_MUTATION → FINISHED
		// Dilakukan oleh PIC Asset setelah receiving confirmed
		mutation.POST("/execute",
			middleware.RequirePermission("execute_mutation"),
			controllers.ExecuteMutation)

		// POST /transactions/mutation/reject?transaction_number → REJECTED
		mutation.POST("/reject",
			middleware.RequirePermission("reject_transaction"),
			controllers.RejectMutation)

		// ============================================================
		// ATTACHMENT PER ASSET
		// ============================================================

		// POST   /transactions/mutation/attachments/upload?transaction_number → upload per asset
		// PUT    /transactions/mutation/attachments/:id/review → approve/reject
		// GET    /transactions/mutation/attachments/status?transaction_number → status semua asset
		mutation.POST("/attachments/upload",
			middleware.RequirePermission("upload_attachment", "create_transaction"),
			controllers.UploadMutationAttachment)

		mutation.PUT("/attachments/:id/review",
			middleware.RequirePermission("review_attachment"),
			controllers.ReviewMutationAttachment)

		mutation.GET("/attachments/status", controllers.GetMutationAttachmentStatus)
	}
}
