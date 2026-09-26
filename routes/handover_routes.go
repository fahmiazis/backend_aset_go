package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// SetupHandoverRoutes — serah terima aset.
//
// Dokumen (BAST) memakai endpoint attachment generik (/attachments) dengan
// transaction_type=handover, jadi tidak ada route dokumen di sini.
func SetupHandoverRoutes(rg *gin.RouterGroup) {
	handover := rg.Group("/transactions/handover")
	handover.Use(middleware.AuthMiddleware())
	{
		handover.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateHandoverDraft)
		handover.GET("", controllers.GetAllHandovers)
		handover.GET("/detail", controllers.GetHandoverDetail)

		// pilihan di form: aset di homebase peminta & user penerima
		handover.GET("/eligible-assets", controllers.GetHandoverEligibleAssets)
		handover.GET("/recipients", controllers.GetHandoverRecipients)

		draft := handover.Group("/draft")
		{
			draft.PUT("/update",
				middleware.RequirePermission("create_transaction"),
				controllers.UpdateHandoverDraft)
			// DRAFT → APPROVAL (approval langsung dibentuk)
			draft.POST("/submit",
				middleware.RequirePermission("create_transaction"),
				controllers.SubmitHandover)
		}

		handover.GET("/approval/status", controllers.GetHandoverApprovalStatus)

		// Tanpa RequirePermission — otorisasinya melekat pada transaksi dan
		// diperiksa di service:
		//   revise            : approver step berjalan
		//   cancel            : pembuat ajuan
		//   confirm/reject-receiving : penerima (HANDOVER) atau pemegang
		//                       confirm_receiving di cabang itu (RETURN)
		handover.POST("/approval/revise", controllers.ReturnHandoverForRevision)
		handover.POST("/cancel", controllers.CancelHandover)
		handover.POST("/confirm-receiving", controllers.ConfirmHandoverReceiving)
		handover.POST("/reject-receiving", controllers.RejectHandoverReceiving)
	}
}
