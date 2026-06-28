package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMiddleware struct {
	secret string
}

func NewJWTMiddleware(secret string) *JWTMiddleware {
	return &JWTMiddleware{
		secret: secret,
	}
}

func (m *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// 2. Check format Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		// 3. Parse the token and validate it
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}

			return []byte(m.secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// 4. Extract claims and extract user_id from claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}

		// jwt.MapClaims decodes JSON numbers into float64, even though the
		// user-service creates user_id from a Go uint.
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok || userIDFloat <= 0 || userIDFloat != float64(uint64(userIDFloat)) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing user_id claim",
			})
			c.Abort()
			return
		}
		userID := strconv.FormatUint(uint64(userIDFloat), 10)

		// 5. Set user_id in context before fowarding the request to the next handler
		c.Set("user_id", userID)
		c.Request.Header.Set("X-User-ID", userID)

		c.Next()
	}
}
