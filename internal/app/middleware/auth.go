package middleware

import (
	"Lab1/internal/app/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

var userRepo *repository.Repository

func InitAuth(repo *repository.Repository) {
	userRepo = repo
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var uid int
		var found bool

		// Проверка Bearer
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " && userRepo != nil && userRepo.Redis != nil {
			token := auth[7:]
			if id, err := userRepo.Redis.GetUserByToken(ctx, token); err == nil {
				uid = id
				found = true
			}
		}

		// Ошибка — не найден пользователь
		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := userRepo.GetUserByID(uid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set("user_id", uid)
		c.Set("is_moderator", user.IsModerator)
		c.Next()
	}
}
