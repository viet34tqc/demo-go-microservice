package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewReverseProxy builds a Gin handler that forwards the current request to
// the target service. The returned handler is registered on gateway routes in
// main.go and only runs when Gin matches one of those routes.
func NewReverseProxy(target string) gin.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	proxy := newReverseProxy(targetURL, nil)

	return func(c *gin.Context) {
		// Bridge Gin's response/request objects back to the standard net/http
		// reverse proxy implementation.
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func newReverseProxy(targetURL *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Transport: transport,
	}

	// Rewrite receives both the inbound request from the client and the outbound
	// request that will be sent to the upstream service.
	proxy.Rewrite = func(req *httputil.ProxyRequest) {
		// SetURL changes the outbound request scheme, host, and base path to the
		// target service URL while preserving the inbound query string.
		req.SetURL(targetURL)

		// Rewrite the request path to remove the "/api" prefix before forwarding it to the internal service.
		// Gateway public path: /api/...
		// Internal service path: /...
		// Example:
		// /api/auth/login -> /auth/login
		// /api/users/me   -> /users/me
		req.Out.URL.Path = strings.TrimPrefix(req.In.URL.Path, "/api")
		req.Out.URL.RawPath = ""
		if req.Out.URL.Path == "" {
			req.Out.URL.Path = "/"
		}

		req.SetXForwarded()
	}

	// Return a consistent JSON response when the upstream service cannot be
	// reached instead of exposing the default reverse proxy error body.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("gateway proxy error for %s %s: %v", r.Method, r.URL.Path, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"bad_gateway","message":"upstream service unavailable"}`))
	}

	return proxy
}
