package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupStockOpnameRoutes(rg *gin.RouterGroup) {
	stockOpname := rg.Group("/transactions/stock-opname")
	stockOpname.Use(middleware.AuthMiddleware())
	{
		stockOpname.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateStockOpname)

		stockOpname.GET("", controllers.GetAllStockOpnames)
		stockOpname.GET("/:number", controllers.GetStockOpnameByNumber)
		stockOpname.PUT("/:number", controllers.UpdateStockOpname)
		stockOpname.DELETE("/:number", controllers.DeleteStockOpname)
	}
}
