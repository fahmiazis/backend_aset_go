package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupTransactionHeaderRoutes(rg *gin.RouterGroup) {
	routes := rg.Group("/transactions")
	routes.Use(middleware.AuthMiddleware())
	{
		// Get all transactions (with filters)
		routes.GET("", controllers.GetAllTransactions)

		// Get my transactions
		routes.GET("/my", controllers.GetMyTransactions)

		// Get specific transaction by number
		routes.GET("/:number", controllers.GetTransactionByNumber)
	}
}
