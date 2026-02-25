package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAssetHistoryRoutes(rg *gin.RouterGroup) {
	routes := rg.Group("")
	routes.Use(middleware.AuthMiddleware())

	// Asset history by asset number
	routes.GET("/assets/:number/history", controllers.GetAssetHistory)

	// All asset histories with filters
	routes.GET("/asset-histories", controllers.GetAllAssetHistories)
}
