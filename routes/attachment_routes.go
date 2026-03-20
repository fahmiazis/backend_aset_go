package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAttachmentRoutes(rg *gin.RouterGroup) {
	// ============================================================
	// ATTACHMENT CONFIG (master)
	// ============================================================
	configs := rg.Group("/attachment-configs")
	configs.Use(middleware.AuthMiddleware())
	{
		configs.GET("", controllers.GetAttachmentConfigs)
		configs.GET("/:id", controllers.GetAttachmentConfigByID)
		configs.POST("",
			middleware.RequirePermission("manage_attachment_config"),
			controllers.CreateAttachmentConfig)
		configs.PUT("/:id",
			middleware.RequirePermission("manage_attachment_config"),
			controllers.UpdateAttachmentConfig)
		configs.DELETE("/:id",
			middleware.RequirePermission("manage_attachment_config"),
			controllers.DeleteAttachmentConfig)
	}

	// ============================================================
	// TRANSACTION ATTACHMENTS
	// ============================================================
	attachments := rg.Group("/attachments")
	attachments.Use(middleware.AuthMiddleware())
	{
		// GET /attachments?transaction_number=xxx&transaction_type=procurement&stage=VERIFIKASI_ASET
		attachments.GET("", controllers.GetTransactionAttachments)

		// GET /attachments/status?transaction_number=xxx&transaction_type=procurement&stage=xxx&branch_code=xxx
		attachments.GET("/status", controllers.GetAttachmentStatusSummary)

		// POST /attachments/upload?transaction_number=xxx&transaction_type=procurement&stage=xxx
		// multipart/form-data: file + attachment_config_id
		attachments.POST("/upload",
			middleware.RequirePermission("upload_attachment", "create_transaction"),
			controllers.UploadAttachment)

		// PUT /attachments/:id/review
		attachments.PUT("/:id/review",
			middleware.RequirePermission("review_attachment"),
			controllers.ReviewAttachment)
	}
}
