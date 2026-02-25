package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupDisposalRoutes(rg *gin.RouterGroup) {
	disposal := rg.Group("/transactions/disposal")
	disposal.Use(middleware.AuthMiddleware())
	{
		disposal.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateDisposal)

		disposal.GET("", controllers.GetAllDisposals)
		disposal.GET("/:number", controllers.GetDisposalByNumber)
		disposal.PUT("/:number", controllers.UpdateDisposal)
		disposal.DELETE("/:number", controllers.DeleteDisposal)
	}
}
