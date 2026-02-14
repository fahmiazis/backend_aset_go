package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupApprovalRoutes(rg *gin.RouterGroup) {
	// All approval routes require authentication
	approvalRoutes := rg.Group("")
	approvalRoutes.Use(middleware.AuthMiddleware())

	// ========================================================================
	// APPROVAL FLOW ROUTES (Admin only)
	// ========================================================================
	flows := approvalRoutes.Group("/approval-flows")
	flows.Use(middleware.RequireRole("admin"))
	{
		flows.GET("", controllers.GetAllApprovalFlows)
		flows.GET("/:id", controllers.GetApprovalFlowByID)
		flows.GET("/code/:code", controllers.GetApprovalFlowByCode)
		flows.POST("", controllers.CreateApprovalFlow)
		flows.PUT("/:id", controllers.UpdateApprovalFlow)
		flows.DELETE("/:id", controllers.DeleteApprovalFlow)
	}

	// ========================================================================
	// APPROVAL FLOW STEP ROUTES (Admin only)
	// ========================================================================
	steps := approvalRoutes.Group("/approval-flow-steps")
	steps.Use(middleware.RequireRole("admin"))
	{
		steps.POST("", controllers.CreateApprovalFlowStep)
		steps.PUT("/:id", controllers.UpdateApprovalFlowStep)
		steps.DELETE("/:id", controllers.DeleteApprovalFlowStep)
		steps.PUT("/step-order-change/:id", controllers.UpdateBulkStepOrderFlowStep)
	}

	// ========================================================================
	// TRANSACTION APPROVAL ROUTES
	// ========================================================================
	transactions := approvalRoutes.Group("/transaction-approvals")
	{
		// Admin can initiate approval for any transaction
		transactions.POST("/initiate",
			middleware.RequirePermission("create_transaction"), // ← Lebih flexible!
			controllers.InitiateTransactionApproval)

		// All authenticated users can approve/reject their assigned approvals
		transactions.POST("/approve", controllers.ApproveTransaction)
		transactions.POST("/reject", controllers.RejectTransaction)

		// Get status of a specific transaction
		transactions.GET("/status/:transaction_number/:transaction_type", controllers.GetTransactionApprovalStatus)

		// Get current user's pending approvals
		transactions.GET("/pending", controllers.GetUserPendingApprovals)
	}
}
