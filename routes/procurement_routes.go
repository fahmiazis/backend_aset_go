package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupProcurementRoutes(rg *gin.RouterGroup) {
	procurement := rg.Group("/transactions/procurement")
	procurement.Use(middleware.AuthMiddleware())
	{
		procurement.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateProcurement)

		procurement.GET("", controllers.GetAllProcurements)
		procurement.GET("/detail", controllers.GetProcurementByNumber)
		procurement.PUT("/", controllers.UpdateProcurement)
		procurement.DELETE("/", controllers.DeleteProcurement)
	}
}
