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

		stockOpnameDraft := stockOpname.Group("/draft")
		{
			// POST   /transactions/stock-opname/draft/add-asset?transaction_number      → tambah asset ke draft
			// PUT    /transactions/stock-opname/draft/update-finding?transaction_number → isi/update temuan fisik per asset
			// DELETE /transactions/stock-opname/draft/remove-asset?transaction_number   → hapus asset dari draft
			stockOpnameDraft.POST("/add-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.AddAssetToStockOpname)

			stockOpnameDraft.PUT("/update-finding",
				middleware.RequirePermission("create_transaction"),
				controllers.UpdateStockOpnameFinding)

			stockOpnameDraft.DELETE("/remove-asset",
				middleware.RequirePermission("create_transaction"),
				controllers.RemoveAssetFromStockOpname)

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
	}
}
