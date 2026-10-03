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

		// Upload Excel: mode=new (nomor aset otomatis) | update (mass update).
		// Hak akses di menu permission "Asset Upload" (route_path /assets/import).
		assets.POST("/import", middleware.RequirePermission("import_asset"), controllers.ImportAssets)
		// template & cek hak akses cukup login; isi template dibatasi cabang user
		assets.GET("/import/template", controllers.AssetImportTemplate)
		assets.GET("/import/allowed", controllers.CanImportAssets)
		// unduh aset (format template mass update), dibatasi cabang user —
		// datanya sama dengan yang sudah terlihat di GET /assets
		assets.GET("/export", controllers.ExportAssets)
		assets.GET("/:number", controllers.GetAssetByNumber)
		assets.GET("/:number/value-history", controllers.GetAssetValueHistory)
	}
}
