package api

import (
	"Lab1/internal/app/middleware"
	"Lab1/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, repo *repository.Repository) {
	api := r.Group("/api")

	api.Use(func(c *gin.Context) {
		c.Set("db", repo.DB)
		c.Next()
	})

	middleware.InitAuth(repo)

	InitStarAPI(repo.DB, api)
	InitOrderAPI(repo.DB, api)
	InitUserAPI(repo, api)
	InitMMObservationStarsAPI(api, repo)
}
