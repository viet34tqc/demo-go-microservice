package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuthAcceptsNumericUserIDClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-secret"
	token := signedToken(t, secret, jwt.MapClaims{
		"user_id": float64(5),
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})

	router := gin.New()
	router.Use(NewJWTMiddleware(secret).RequireAuth())
	router.GET("/api/users/me", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			t.Fatal("expected user_id in gin context")
		}

		if userID != "5" {
			t.Fatalf("expected user_id 5, got %v", userID)
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id_header": c.GetHeader("X-User-ID"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	if rec.Body.String() != `{"user_id_header":"5"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestRequireAuthRejectsMissingUserIDClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-secret"
	token := signedToken(t, secret, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	router := gin.New()
	router.Use(NewJWTMiddleware(secret).RequireAuth())
	router.GET("/api/users/me", func(c *gin.Context) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	if rec.Body.String() != `{"error":"missing user_id claim"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func signedToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return token
}
