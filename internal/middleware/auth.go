package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-055/internal/application"
	"github.com/wyw14/cry-055/internal/domain"
)

func LocalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		actorID := strings.TrimSpace(c.GetHeader("X-Actor-ID"))
		roles := splitRoles(c.GetHeader("X-Actor-Roles"))
		if actorID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "AUTH_REQUIRED", "message": "X-Actor-ID is required", "field_errors": []any{}, "request_id": GetRequestID(c)})
			return
		}
		metadata := application.RequestMetadata{RequestID: GetRequestID(c), Actor: domain.Actor{ID: domain.ID(actorID), Name: c.GetHeader("X-Actor-Name"), Roles: roles}}
		c.Request = c.Request.WithContext(application.WithRequestMetadata(c.Request.Context(), metadata))
		c.Next()
	}
}
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata := application.MetadataFromContext(c.Request.Context())
		if !metadata.Actor.HasRole(role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "ROLE_REQUIRED", "message": "required role: " + role, "field_errors": []any{}, "request_id": GetRequestID(c)})
			return
		}
		c.Next()
	}
}
func splitRoles(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
