package api

import (
	"Lab1/internal/app/middleware"
	"Lab1/internal/app/models"
	"Lab1/internal/app/repository"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/sirupsen/logrus"
)

var userRepo *repository.Repository

const sessionTTL = 24 * time.Hour

func InitUserAPI(repo *repository.Repository, r *gin.RouterGroup) {
	userRepo = repo
	registerUserRoutes(r)
}

func registerUserRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.POST("/register", registerUser)
		users.POST("/login", loginUser)
		users.POST("/logout", logoutUser)
		users.GET("/me", middleware.AuthMiddleware(), getCurrentUser)
		users.PUT("/me", middleware.AuthMiddleware(), updateCurrentUser)
	}
}

// registerUser godoc
// @Summary Регистрация пользователя
// @Description Создает нового пользователя с логином, паролем и флагом модератора
// @Tags Users
// @Accept json
// @Produce json
// @Param user body object{Username=string,Password=string,IsModerator=bool} true "Данные пользователя"
// @Success 201 {object} map[string]string "user created"
// @Failure 400 {object} map[string]string "invalid json"
// @Failure 500 {object} map[string]string "Ошибка создания пользователя"
// @Router /users/register [post]
func registerUser(c *gin.Context) {
	var req struct {
		Username    string `json:"Username"`
		Password    string `json:"Password"`
		IsModerator bool   `json:"IsModerator"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	hash, _ := repository.HashPassword(req.Password)
	user := models.User{
		Username:     req.Username,
		PasswordHash: hash,
		IsModerator:  req.IsModerator,
	}

	if err := userRepo.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

// loginUser godoc
// @Summary Авторизация пользователя
// @Description Логин пользователя и получение токена авторизации
// @Tags Users
// @Accept json
// @Produce json
// @Param credentials body object{Username=string,Password=string} true "Данные для входа"
// @Success 200 {object} map[string]string "token"
// @Failure 400 {object} map[string]string "invalid json"
// @Failure 401 {object} map[string]string "invalid credentials"
// @Router /users/login [post]
func loginUser(c *gin.Context) {
	var req struct {
		Username string `json:"Username"`
		Password string `json:"Password"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	user, err := userRepo.GetUserByUsername(req.Username)
	if err != nil || !repository.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := uuid.NewString()

	// сохраняем токен в Redis (token:<uuid> -> userID)
	if userRepo != nil && userRepo.Redis != nil {
		if err := userRepo.Redis.SetUserToken(c.Request.Context(), token, user.UserID, sessionTTL); err != nil {
			// логирование ошибки, но не фатал
			fmt.Println("ERROR saving session to redis:", err)
		}
	} else {
		fmt.Println("Redis не инициализирован при логине")
	}

	// возвращаем токен в теле (для Postman/Authorization header)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// logoutUser godoc
// @Summary Выход пользователя
// @Description Удаляет текущую сессию пользователя и очищает cookie
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "logged out"
// @Router /users/logout [post]
func logoutUser(c *gin.Context) {
	ctx := c.Request.Context()

	// пробуем удалить токен из Authorization header, если прислали
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " && userRepo != nil && userRepo.Redis != nil {
		token := auth[7:]
		_ = userRepo.Redis.DeleteToken(ctx, token)
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// getCurrentUser godoc
// @Summary Получить информацию о текущем пользователе
// @Description Возвращает ID, имя пользователя и флаг модератора для авторизованного пользователя
// @Tags Users
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Информация о пользователе"
// @Failure 500 {object} map[string]string "Ошибка получения пользователя"
// @Router /users/me [get]
func getCurrentUser(c *gin.Context) {
	uid := auth.CurrentUserID()

	user, err := userRepo.GetUserByID(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":      user.UserID,
		"username":     user.Username,
		"is_moderator": user.IsModerator,
	})
}

// updateCurrentUser godoc
// @Summary Обновление информации о текущем пользователе
// @Description Позволяет обновить имя пользователя или пароль. Флаги модератора и ID недоступны для изменения
// @Tags Users
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param user body object true "Поля для обновления пользователя"
// @Success 200 {object} map[string]string "user updated"
// @Failure 400 {object} map[string]string "invalid json"
// @Failure 500 {object} map[string]string "Ошибка обновления пользователя"
// @Router /users/me [put]
func updateCurrentUser(c *gin.Context) {
	uid := auth.CurrentUserID()
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	delete(req, "user_id")
	delete(req, "is_moderator")
	//delete(req, "password_hash")

	if pw, ok := req["password"]; ok {
		hash, _ := repository.HashPassword(pw.(string))
		req["password_hash"] = hash
		delete(req, "password")
	}

	if err := db.Model(&models.User{}).Where("user_id = ?", uid).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}
