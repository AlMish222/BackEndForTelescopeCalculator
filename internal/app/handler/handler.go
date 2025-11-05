//goland:noinspection ALL
package handler

import (
	"Lab1/internal/app/api"
	"Lab1/internal/app/repository"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	Repository  *repository.Repository
	MinioClient *minio.Client
	RedisClient *redis.Client
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repository: r}
}

func (h *Handler) RegisterHandler(rou *gin.Engine) {
	rou.GET("/", h.GetOrders)
	rou.GET("/stars", h.GetStars)
	rou.GET("/stars/:id", h.GetStarByID)
	rou.POST("/order/:id/delete", h.DeleteOrder)

	rou.GET("/order/:id", h.GetOrder)
	rou.POST("/order", h.CreateOrder)

	rou.POST("/order/:id/update", h.UpdateOrder)

	rou.POST("/star/:id/add", h.AddStarToDraftOrder)

	api.RegisterRoutes(rou, h.Repository)
}

func (h *Handler) RegisterStatic(rou *gin.Engine) {
	absPath, _ := filepath.Abs("templates/*")
	rou.LoadHTMLGlob(absPath)
	rou.Static("/styles", "./resources/styles")
}
