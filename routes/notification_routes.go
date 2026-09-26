package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// Tanpa RequirePermission: isinya sudah difilter per user dengan aturan yang
// sama dengan hak akses tiap stage.
func SetupNotificationRoutes(rg *gin.RouterGroup) {
	notifications := rg.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware())
	{
		notifications.GET("/waiting", controllers.GetWaitingNotifications)
	}
}
