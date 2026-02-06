package middleware

import (
	"backend-go/config"
	"backend-go/models"
	"backend-go/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware - Validate JWT access token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		fmt.Printf("AuthMiddleware: Set user_id = %s\n", claims.UserID)

		c.Next()
	}
}

// RequireRole - Check if user has required role (by role name)
// Usage: middleware.RequireRole("admin", "manager")
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context
		rolesInterface, exists := c.Get("roles")
		if !exists {
			utils.ErrorResponse(c, http.StatusForbidden, "No roles found")
			c.Abort()
			return
		}

		userRoles, ok := rolesInterface.([]string)
		if !ok {
			utils.ErrorResponse(c, http.StatusForbidden, "Invalid roles format")
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			utils.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission - Check if user has required permission on current route
// Logic: ambil path dari request → cari menu dengan route_path yang match → cek role_menus
// Usage: middleware.RequirePermission("write", "delete")
func RequirePermission(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by AuthMiddleware)
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
			c.Abort()
			return
		}

		// Get current request path (full path dengan /api/v1)
		requestPath := c.Request.URL.Path

		// 1. Ambil user roles
		var userRoles []models.UserRole
		if err := config.DB.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch user roles")
			c.Abort()
			return
		}

		if len(userRoles) == 0 {
			utils.ErrorResponse(c, http.StatusForbidden, "User has no roles")
			c.Abort()
			return
		}

		roleIDs := make([]string, len(userRoles))
		for i, ur := range userRoles {
			roleIDs[i] = ur.RoleID
		}

		// 2. Cari menu yang route_path-nya exact match dengan request path
		// Untuk route dinamis (misal /api/v1/users/:id), kita perlu strip ID-nya
		// Simple approach: ambil base path aja (split by "/" dan ambil s.d 3 segment)
		// Misal: /api/v1/users/123 → /api/v1/users
		pathParts := strings.Split(requestPath, "/")
		basePath := requestPath
		if len(pathParts) > 4 { // /api/v1/resource/...
			basePath = strings.Join(pathParts[:4], "/") // ambil /api/v1/resource
		}

		var menu models.Menu
		if err := config.DB.Where("route_path = ?", basePath).First(&menu).Error; err != nil {
			// Menu tidak ditemukan, skip permission check
			// Bisa juga di-block tergantung policy
			c.Next()
			return
		}

		// 3. Ambil role_menus untuk menu ini dari role-role user
		var roleMenus []models.RoleMenu
		if err := config.DB.Where("role_id IN ? AND menu_id = ?", roleIDs, menu.ID).Find(&roleMenus).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch permissions")
			c.Abort()
			return
		}

		if len(roleMenus) == 0 {
			utils.ErrorResponse(c, http.StatusForbidden, "No permissions for this menu")
			c.Abort()
			return
		}

		// 4. Merge permissions dari semua role_menus (union)
		userPermissions := make(map[string]bool)
		for _, rm := range roleMenus {
			for _, p := range rm.Permissions {
				userPermissions[p] = true
			}
		}

		// 5. Check if user has any of the required permissions
		hasPermission := false
		for _, reqPerm := range requiredPermissions {
			if userPermissions[reqPerm] {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			utils.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions for this action")
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth - Similar to AuthMiddleware but doesn't abort if no token
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}
