package api

import (
	"Lab1/internal/app/auth"
	"Lab1/internal/app/config"
	"Lab1/internal/app/models"
	"Lab1/internal/app/repository"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

var db *gorm.DB
var repo *repository.Repository

func InitStarAPI(database *gorm.DB, r *gin.RouterGroup) {
	db = database
	repo = repository.NewRepositoryFromDB(db)
	registerStarRoutes(r)
}

func registerStarRoutes(r *gin.RouterGroup) {
	stars := r.Group("/stars")
	{
		stars.GET("", getStars)
		stars.GET("/:id", getStarByID)
		stars.POST("", createStar)

		stars.PUT("/:id", updateStar)
		stars.DELETE("/:id", deleteStar)
		stars.POST("/:id/image", uploadStarImage) //реализую позже
		stars.POST("/:id/add", addStarToDraftOrder)
	}
}

func getStars(c *gin.Context) {
	var stars []models.Star
	if err := db.Find(&stars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения звёзд: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, stars)
}

func getStarByID(c *gin.Context) {
	id := c.Param("id")
	var star models.Star
	if err := db.First(&star, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Звезда не найдена"})
		return
	}
	c.JSON(http.StatusOK, star)
}

func createStar(c *gin.Context) {
	var input models.Star

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}

	if input.StarName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название звезды обязательно"})
		return
	}

	if err := db.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Звезда успешно добавлена",
		"star":    input,
	})
}

func updateStar(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var existing models.Star
	if err := db.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Звезда не найдена"})
		return
	}

	var input models.Star
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}

	// Обновляем поля
	existing.StarName = input.StarName
	existing.ShortDescription = input.ShortDescription
	existing.Description = input.Description
	existing.ImageURL = input.ImageURL
	existing.IsActive = input.IsActive
	existing.RA = input.RA
	existing.Dec = input.Dec

	if err := db.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Звезда успешно обновлена",
		"star":    existing,
	})
}

func deleteStar(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	// пробуем удалить запись
	result := db.Delete(&models.Star{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении звезды"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Звезда не найдена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Звезда успешно удалена"})
}

func uploadStarImage(c *gin.Context) {
	idStr := c.Param("id")
	starID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	// Получаем файл из запроса
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не получен"})
		return
	}

	// Генерируем уникальное имя файла (латиница)
	filename := fmt.Sprintf("star_%d_%s", starID, file.Filename)

	// Загружаем файл в MinIO
	ctx := context.Background()
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка открытия файла: " + err.Error()})
		return
	}
	defer src.Close()

	_, err = config.MinioClient.PutObject(ctx, "test", filename, src, file.Size, minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки в MinIO: " + err.Error()})
		return
	}

	// Обновляем поле ImageURL в БД
	if err := db.Model(&models.Star{}).Where("star_id = ?", starID).Update("image_url", filename).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения пути изображения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Изображение успешно загружено",
		"filename": filename,
	})
}

func addStarToDraftOrder(c *gin.Context) {
	userID := auth.CurrentUserID()
	starIDStr := c.Param("id")

	starID, err := strconv.Atoi(starIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID звезды"})
		return
	}

	// Получаем или создаём черновик
	order, err := repo.GetOrCreateDraftOrder(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения или создания черновика: " + err.Error()})
		return
	}

	// Проверяем, не добавлена ли уже звезда
	var existing models.TelescopeObservationStar
	if err := db.Where("observation_id = ? AND star_id = ?", order.TelescopeObservationID, starID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Звезда уже добавлена в заявку"})
		return
	}

	// Добавляем новую запись
	relation := models.TelescopeObservationStar{
		TelescopeObservationID: order.TelescopeObservationID,
		StarID:                 starID,
		OrderNumber:            1,
		Quantity:               1,
	}

	if err := db.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка добавления звезды в заявку: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Звезда успешно добавлена в черновик заявки",
		"orderID": order.TelescopeObservationID,
	})
}
