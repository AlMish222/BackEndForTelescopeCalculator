package api

import (
	//"Lab1/internal/app/handler"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	api := r.Group("/api")

	api.Use(func(c *gin.Context) {
		c.Set("db", db)
		//c.Set("handler", h)
		c.Next()
	})

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	InitStarAPI(db, api)
}
