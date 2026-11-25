package main

import (
	"log"
	"net/http"

	"github.com/amitc1996/tenant-provisioning/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "trading-provisioning",
		})
	})

	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
