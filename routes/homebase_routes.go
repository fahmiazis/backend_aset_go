package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupHomebaseRoutes(rg *gin.RouterGroup) {
	// All routes require authentication
	routes := rg.Group("")
	routes.Use(middleware.AuthMiddleware())

	// ========================================================================
	// HOMEBASE MANAGEMENT
	// ========================================================================
	homebases := routes.Group("/user")
	{
		// Get user's homebases
		homebases.GET("/homebases", controllers.GetUserHomebases)

		// Set active homebase (for login selection)
		homebases.POST("/homebase/set-active", controllers.SetActiveHomebase)
	}

	// ========================================================================
	// TRANSACTION NUMBER MANAGEMENT
	// ========================================================================
	txNumber := routes.Group("/transaction-number")
	{
		// Generate new transaction number
		txNumber.POST("/generate", controllers.GenerateTransactionNumber)

		// Mark transaction as used (submitted)
		txNumber.POST("/mark-used", controllers.MarkTransactionUsed)

		// Mark transaction as expired (cancelled)
		txNumber.POST("/mark-expired", controllers.MarkTransactionExpired)

		// Get transaction status
		txNumber.GET("/status/:number", controllers.GetTransactionStatus)
	}
}
