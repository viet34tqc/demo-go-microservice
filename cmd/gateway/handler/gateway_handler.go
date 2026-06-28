package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	userServiceURL string
	todoServiceURL string
}

func NewGatewayHandler(userServiceURL string, todoServiceURL string) *GatewayHandler {
	return &GatewayHandler{
		userServiceURL: userServiceURL,
		todoServiceURL: todoServiceURL,
	}
}

func (h *GatewayHandler) forward(c *gin.Context, targetBaseURL string) {
	targetURL := targetBaseURL + internalServicePath(c.Request.URL.Path)

	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequest(
		c.Request.Method,
		targetURL,
		c.Request.Body,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
		})
		return
	}

	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to forward request",
		})
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Status(resp.StatusCode)

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to copy response",
		})
		return
	}
}

func (h *GatewayHandler) ForwardToUserService(c *gin.Context) {
	h.forward(c, h.userServiceURL)
}

func (h *GatewayHandler) ForwardToTodoService(c *gin.Context) {
	h.forward(c, h.todoServiceURL)
}

func internalServicePath(path string) string {
	internalPath := strings.TrimPrefix(path, "/api")
	if internalPath == "" {
		return "/"
	}

	return internalPath
}
