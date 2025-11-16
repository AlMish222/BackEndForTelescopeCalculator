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
	rou.GET("/", h.GetTelescopeObservations)
	rou.GET("/stars", h.GetStars)
	rou.GET("/stars/:id", h.GetStarByID)
	rou.POST("/telescopeObservation/:id/delete", h.DeleteTelescopeObservation)

	rou.GET("/telescopeObservation/:id", h.GetTelescopeObservation)
	rou.POST("/telescopeObservation", h.CreateTelescopeObservation)

	rou.POST("/telescopeObservation/:id/update", h.UpdateTelescopeObservation)

	rou.POST("/star/:id/add", h.AddStarToDraftOrder)

	api.RegisterRoutes(rou, h.Repository)
}

func (h *Handler) RegisterStatic(rou *gin.Engine) {
	absPath, _ := filepath.Abs("templates/*")
	rou.LoadHTMLGlob(absPath)
	rou.Static("/styles", "./resources/styles")
}
