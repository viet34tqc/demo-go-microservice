package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RequireUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing X-User-ID header",
			})
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(userIDHeader, 10, 64)
		if err != nil || userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid X-User-ID header",
			})
			c.Abort()
			return
		}

		c.Set("userID", uint(userID))
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	value, exists := c.Get("userID")
	if !exists {
		return 0
	}

	userID, ok := value.(uint)
	if !ok {
		return 0
	}

	return userID
}
