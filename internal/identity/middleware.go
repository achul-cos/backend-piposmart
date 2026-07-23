package identity

import (
	"net/http"

	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "current_user"

func AuthMiddleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := bearerToken(c.GetHeader("Authorization"))
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Token akses wajib dikirim", nil)
			return
		}
		user, err := service.UserFromAccessToken(c.Request.Context(), token)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Token akses tidak valid", nil)
			return
		}
		c.Set(currentUserKey, user)
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok || !hasPermission(user, permission) {
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "Kamu tidak memiliki akses ke fitur ini", nil)
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (User, bool) {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return User{}, false
	}
	user, ok := value.(User)
	return user, ok
}
