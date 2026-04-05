package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupMutationRoutes(rg *gin.RouterGroup) {
	mutation := rg.Group("/mutation")
	mutation.Use(middleware.AuthMiddleware())
	{
		mutation.POST("",
			middleware.RequirePermission("create_transaction"),
			controllers.CreateMutation)

		mutation.GET("", controllers.GetAllMutations)
		mutation.GET("/:number", controllers.GetMutationByNumber)
		mutation.PUT("/:number", controllers.UpdateMutation)
		mutation.DELETE("/:number", controllers.DeleteMutation)
	}
}
