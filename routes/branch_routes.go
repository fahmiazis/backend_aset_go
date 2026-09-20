package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func SetupBranchRoutes(rg *gin.RouterGroup) {
	// Branch routes
	branchs := rg.Group("/branchs")
	branchs.Use(middleware.AuthMiddleware()) // All branch routes require authentication
	{
		// Routes accessible by all authenticated users
		branchs.GET("", controllers.GetAllBranchs)
		branchs.GET("/:id", controllers.GetBranchByID)
		branchs.GET("/:id/users", controllers.GetBranchUsers)

		// Anggota homebase cabang ini
		branchs.GET("/:id/homebase-users", controllers.GetBranchHomebaseUsers)

		// User yang di-assign ke cabang ini (akses tambahan, bukan homebase)
		branchs.GET("/:id/assigned-users", controllers.GetBranchAssignedUsers)

		// Admin only routes
		adminRoutes := branchs.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.POST("", controllers.CreateBranch)
			adminRoutes.PUT("/:id", controllers.UpdateBranch)
			adminRoutes.DELETE("/:id", controllers.DeleteBranch)

			// Set banyak user sekaligus jadi homebase di cabang ini
			adminRoutes.POST("/:id/homebase-users", controllers.AssignHomebaseUsers)
			adminRoutes.DELETE("/:id/homebase-users/:user_id", controllers.RemoveHomebaseUser)

			// Beri banyak user sekaligus akses ke cabang ini
			adminRoutes.POST("/:id/assigned-users", controllers.AssignBranchUsers)
			adminRoutes.DELETE("/:id/assigned-users/:user_id", controllers.RemoveBranchUser)
		}
	}

	// User-Branch assignment routes (under /users)
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		// Routes accessible by all authenticated users
		users.GET("/:id/branchs", controllers.GetUserBranchs)

		// Admin only routes
		adminUserRoutes := users.Group("")
		adminUserRoutes.Use(middleware.RequireRole("admin"))
		{
			adminUserRoutes.POST("/:id/branchs", controllers.AssignBranchsToUser)
			adminUserRoutes.DELETE("/:id/branchs/:branch_id", controllers.RemoveBranchFromUser)
		}
	}
}
