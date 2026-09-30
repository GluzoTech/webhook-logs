package webhook

import (
	"github.com/GluzoTech/webhook-logs/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the webhook logger and its viewer.
// The viewer is guarded by viewKey, passed as ?key=... on the request.
func RegisterRoutes(r gin.IRouter, h *Handler, viewKey string) {
	api := r.Group("/api/v1/webhook")
	{
		api.POST("/log", h.Log)
		api.GET("/view", middleware.APIKey(viewKey), h.View)
	}
}
