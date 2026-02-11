package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupCustomApprovalRoutes(rg *gin.RouterGroup) {
	// All routes require authentication
	customApprovals := rg.Group("/custom-approvals")
	customApprovals.Use(middleware.AuthMiddleware())
	{
		// ========================================================================
		// USER ROUTES - Create & Manage Own Custom Approvals
		// ========================================================================

		// Get user's own custom approvals
		customApprovals.GET("/me", controllers.GetUserCustomApprovals)

		// Create custom approval
		customApprovals.POST("", controllers.CreateCustomApproval)

		// Get specific custom approval
		// customApprovals.GET("/:id", controllers.GetCustomApprovalByID)

		// Update custom approval (will reset to pending verification)
		customApprovals.PUT("/:id", controllers.UpdateCustomApproval)

		// Delete custom approval
		customApprovals.DELETE("/:id", controllers.DeleteCustomApproval)

		// ========================================================================
		// ADMIN/TIM ASSET ROUTES - Verification
		// ========================================================================

		// Get all pending verifications (admin only)
		customApprovals.GET("/pending-verifications",
			middleware.RequireRole("admin", "asset_team"),
			controllers.GetPendingVerifications)

		// Verify/Reject custom approval (admin only)
		customApprovals.POST("/:id/verify",
			middleware.RequireRole("admin", "asset_team"),
			controllers.VerifyCustomApproval)
	}

	// ========================================================================
	// APPROVAL FLOW ROUTES - Check Permission
	// ========================================================================
	flows := rg.Group("/approval-flows")
	flows.Use(middleware.AuthMiddleware())
	{
		// Check if user can customize a flow
		flows.GET("/:id/can-customize", controllers.CheckCanCustomizeFlow)
	}
}
