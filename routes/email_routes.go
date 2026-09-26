package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupEmailRoutes(rg *gin.RouterGroup) {
	// ============================================================
	// EMAIL TEMPLATE (master) — sama seperti attachment-configs, tulis khusus
	// admin
	// ============================================================
	templates := rg.Group("/email-templates")
	templates.Use(middleware.AuthMiddleware())
	{
		templates.GET("", controllers.GetEmailTemplates)
		templates.GET("/:id", controllers.GetEmailTemplateByID)
		templates.POST("", middleware.RequireRole("admin"), controllers.CreateEmailTemplate)
		templates.PUT("/:id", middleware.RequireRole("admin"), controllers.UpdateEmailTemplate)
		templates.DELETE("/:id", middleware.RequireRole("admin"), controllers.DeleteEmailTemplate)
	}

	// ============================================================
	// EMAIL TRANSAKSI
	// Tanpa RequirePermission, mengikuti endpoint revisi/batal: yang berhak
	// adalah user yang memproses transaksi itu, dan itu diperiksa di service
	// (userActedRecently), bukan lewat hak akses menu.
	// ============================================================
	emails := rg.Group("/transaction-emails")
	emails.Use(middleware.AuthMiddleware())
	{
		emails.GET("/preview", controllers.PreviewTransactionEmail)
		emails.POST("/send", controllers.SendTransactionEmail)
		// selain admin hanya melihat & mengirim ulang kirimannya sendiri
		emails.GET("/logs", controllers.GetTransactionEmailLogs)
		emails.POST("/logs/:id/resend", controllers.ResendTransactionEmail)
	}
}
