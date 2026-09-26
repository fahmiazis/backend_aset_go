package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAssetMasterRoutes(rg *gin.RouterGroup) {
	assets := rg.Group("/assets")
	assets.Use(middleware.AuthMiddleware())
	{
		assets.GET("", controllers.GetAllAssets)
		// sebelum /:number — rute statis diprioritaskan gin
		assets.GET("/my-branches", controllers.GetViewableAssetBranches)
		assets.GET("/:number", controllers.GetAssetByNumber)
		assets.GET("/:number/value-history", controllers.GetAssetValueHistory)
	}
}
