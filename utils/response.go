package utils

import "github.com/gin-gonic/gin"

func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"status":  "error",
		"message": message,
	})
}

func ValidationErrorResponse(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"status":  "error",
		"message": "Validation failed",
		"errors":  err.Error(),
	})
}
