package api

import (
	"Lab1/internal/app/models"
	"Lab1/internal/app/repository"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var userRepo *repository.Repository
var sessions = map[string]int{} // token -> userID

func InitUserAPI(database *gorm.DB, r *gin.RouterGroup) {
	db = database
	userRepo = repository.NewRepositoryFromDB(db)
	registerUserRoutes(r)
}

func registerUserRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.POST("/register", registerUser)                 // POST /api/users/register
		users.POST("/login", loginUser)                       // POST /api/users/login
		users.POST("/logout", logoutUser)                     // POST /api/users/logout
		users.GET("/me", authMiddleware(), getCurrentUser)    // GET /api/users/me
		users.PUT("/me", authMiddleware(), updateCurrentUser) // PUT /api/users/me
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			token := auth[7:]
			if uid, ok := sessions[token]; ok {
				c.Set("user_id", uid)
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

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
	sessions[token] = user.UserID

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func logoutUser(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token := auth[7:]
		delete(sessions, token)
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func getCurrentUser(c *gin.Context) {
	uid := c.GetInt("user_id")
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

func updateCurrentUser(c *gin.Context) {
	uid := c.GetInt("user_id")
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	delete(req, "user_id")
	delete(req, "is_moderator")
	delete(req, "password_hash")

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
