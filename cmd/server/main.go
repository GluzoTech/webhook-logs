package main

import (
	"log"
	"net/http"

	"github.com/GluzoTech/webhook-logs/config"
	"github.com/GluzoTech/webhook-logs/internal/webhook"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	repo, err := webhook.NewFileRepository(cfg.LogDir, cfg.MaxBodyBytes)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	svc := webhook.NewService(repo, cfg.MaxBodyBytes, cfg.PageSize)

	handler, err := webhook.NewHandler(svc)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	webhook.RegisterRoutes(router, handler, cfg.ViewAPIKey)

	log.Printf("webhook-logs listening on :%s, writing to %s", cfg.Port, cfg.LogDir)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
