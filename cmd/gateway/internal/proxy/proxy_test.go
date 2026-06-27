package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewReverseProxyRewritesAPIPrefix(t *testing.T) {
	targetURL, err := url.Parse("http://user-service:8081")
	if err != nil {
		t.Fatal(err)
	}

	proxy := newReverseProxy(targetURL, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/auth/register" {
			t.Errorf("expected upstream path /auth/register, got %s", r.URL.Path)
		}

		if r.URL.Host != "user-service:8081" {
			t.Errorf("expected upstream host user-service:8081, got %s", r.URL.Host)
		}

		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    r,
		}, nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
