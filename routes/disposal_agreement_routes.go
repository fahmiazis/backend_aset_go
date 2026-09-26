package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// SetupDisposalAgreementRoutes — kesepakatan disposal (agreement).
//
// Path-nya sengaja /transactions/disposal-agreements (bukan di bawah
// /transactions/disposal) supaya normalisasi RequirePermission menghasilkan
// menu sendiri dan hak aksesnya bisa dipisah dari disposal biasa.
func SetupDisposalAgreementRoutes(rg *gin.RouterGroup) {
	agreements := rg.Group("/transactions/disposal-agreements")
	agreements.Use(middleware.AuthMiddleware())
	{
		// Transaksi yang sudah lolos approval request dan belum masuk
		// agreement aktif — jadi pilihan saat membuat agreement.
		agreements.GET("/eligible", controllers.GetEligibleDisposalsForAgreement)

		agreements.GET("", controllers.GetAllDisposalAgreements)
		agreements.GET("/detail", controllers.GetDisposalAgreementDetail)
		agreements.GET("/approval-status", controllers.GetDisposalAgreementApprovalStatus)

		agreements.POST("",
			middleware.RequirePermission("manage_disposal_agreement"),
			controllers.CreateDisposalAgreement)

		// Revisi oleh approver step berjalan. Tanpa RequirePermission,
		// mengikuti revisi disposal/procurement/mutasi: otorisasinya melekat
		// pada agreement ("apakah kamu approver step ini") dan diperiksa di
		// service.
		agreements.POST("/revise", controllers.ReviseDisposalAgreement)
	}
}
