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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		fmt.Printf("AuthMiddleware: Set user_id = %s\n", claims.UserID)

		c.Next()
	}
}

// RequireRole - Check if user has required role (by role name)
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
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
func RequirePermission(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
			c.Abort()
			return
		}

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

		// 2. Normalize path
		// Strip /api/v1, skip ID segments, ambil maks 3 segment pertama
		// Contoh:
		// /api/v1/transactions/procurement/execute        → /transactions/procurement/execute
		// /api/v1/transactions/procurement/execute/detail → /transactions/procurement/execute
		// /api/v1/assets/1/history                       → /assets/history
		// /api/v1/menus                                  → /menus
		normalizedPath := requestPath
		if after, ok := strings.CutPrefix(normalizedPath, "/api/v1"); ok {
			normalizedPath = after
		}

		// Filter out ID segments (numeric atau UUID)
		rawParts := strings.Split(strings.TrimRight(normalizedPath, "/"), "/")
		cleanParts := []string{""}
		for _, part := range rawParts {
			if part == "" {
				continue
			}
			if isIDSegment(part) {
				continue
			}
			cleanParts = append(cleanParts, part)
			// Maksimal 3 segment: /resource/sub/leaf
			if len(cleanParts) == 4 {
				break
			}
		}

		basePath := strings.Join(cleanParts, "/")

		fmt.Printf("RequirePermission: requestPath=%s, basePath=%s\n", requestPath, basePath)

		// 3. Cari menu — FIX: strict block kalau menu tidak ditemukan
		var menu models.Menu
		if err := config.DB.Where("route_path = ?", basePath).First(&menu).Error; err != nil {
			// Menu tidak terdaftar = akses ditolak
			// Semua endpoint wajib didaftarkan di tabel menus
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: route not registered in menu")
			c.Abort()
			return
		}

		// 4. Ambil role_menus untuk menu ini
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

		// 5. Merge permissions dari semua role (union)
		userPermissions := make(map[string]bool)
		for _, rm := range roleMenus {
			for _, p := range rm.Permissions {
				userPermissions[p] = true
			}
		}

		// 6. Check required permissions
		hasPermission := false
		for _, reqPerm := range requiredPermissions {
			if userPermissions[reqPerm] {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Insufficient permissions: required %v", requiredPermissions))
			c.Abort()
			return
		}

		c.Next()
	}
}

// isIDSegment deteksi apakah segment path adalah ID dinamis
// yang harus di-skip saat normalize path untuk cek menu
// Contoh ID: "123", "456", "uuid-xxxx-xxxx"
func isIDSegment(segment string) bool {
	// Cek apakah numeric
	allDigits := true
	for _, c := range segment {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if allDigits && len(segment) > 0 {
		return true
	}

	// Cek apakah UUID format (8-4-4-4-12 dengan hex dan dash)
	// Contoh: a2a42c8c-9e43-48e9-8629-9d059369207c
	if len(segment) == 36 {
		uuidParts := strings.Split(segment, "-")
		if len(uuidParts) == 5 &&
			len(uuidParts[0]) == 8 &&
			len(uuidParts[1]) == 4 &&
			len(uuidParts[2]) == 4 &&
			len(uuidParts[3]) == 4 &&
			len(uuidParts[4]) == 12 {
			return true
		}
	}

	return false
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
