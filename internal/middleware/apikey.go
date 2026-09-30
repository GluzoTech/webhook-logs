package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKey guards a route with a shared secret, read from the `key` query
// parameter or, as a convenience, the X-API-Key header.
func APIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "viewer is not configured: VIEW_API_KEY is unset",
			})
			return
		}

		provided := c.Query("key")
		if provided == "" {
			provided = c.GetHeader("X-API-Key")
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		c.Next()
	}
}
