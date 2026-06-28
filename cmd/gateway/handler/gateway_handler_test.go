package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGatewayHandlerStripsAPIPrefixWhenForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalTransport := http.DefaultClient.Transport
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}

		if r.URL.Host != "user-service:8081" {
			t.Errorf("expected upstream host user-service:8081, got %s", r.URL.Host)
		}

		if r.URL.Path != "/auth/register" {
			t.Errorf("expected upstream path /auth/register, got %s", r.URL.Path)
		}

		if r.URL.RawQuery != "source=curl" {
			t.Errorf("expected query source=curl, got %s", r.URL.RawQuery)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected content type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read upstream body: %v", err)
		}

		if string(body) != `{"email":"john@example.com"}` {
			t.Errorf("unexpected body: %s", string(body))
		}

		return &http.Response{
			StatusCode: http.StatusCreated,
			Header: http.Header{
				"X-Upstream-Service": []string{"user-service"},
			},
			Body:    io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request: r,
		}, nil
	})

	router := gin.New()
	gatewayHandler := NewGatewayHandler("http://user-service:8081", "http://todo-service:8082")
	router.POST("/api/auth/register", gatewayHandler.ForwardToUserService)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/register?source=curl",
		strings.NewReader(`{"email":"john@example.com"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	if rec.Body.String() != `{"ok":true}` {
		t.Errorf("expected upstream body, got %s", rec.Body.String())
	}

	if rec.Header().Get("X-Upstream-Service") != "user-service" {
		t.Errorf("expected upstream response header to be copied")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestInternalServicePath(t *testing.T) {
	tests := map[string]string{
		"/api":               "/",
		"/api/auth/register": "/auth/register",
		"/api/todos/123":     "/todos/123",
		"/auth/register":     "/auth/register",
		"/":                  "/",
		"":                   "/",
	}

	for path, expected := range tests {
		t.Run(path, func(t *testing.T) {
			if got := internalServicePath(path); got != expected {
				t.Fatalf("expected %s, got %s", expected, got)
			}
		})
	}
}
