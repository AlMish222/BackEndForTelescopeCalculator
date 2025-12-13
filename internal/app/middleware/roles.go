package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireModerator() gin.HandlerFunc {
	return func(c *gin.Context) {
		isModerator, exists := c.Get("is_moderator")
		if !exists || !isModerator.(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden — moderator access required",
			})
			return
		}
		c.Next()
	}
}
