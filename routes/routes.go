package routes

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Setup all route modules
		SetupAuthRoutes(v1)
		SetupUserRoutes(v1)
	}

	// Health check endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Server is running",
		})
	})
}
