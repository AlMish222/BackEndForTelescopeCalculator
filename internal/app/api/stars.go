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
		stars.POST("/:id/image", uploadStarImage)
		stars.POST("/:id/add", addStarToDraftOrder)
	}
}

// getStars godoc
// @Summary Получить список звёзд
// @Description Возвращает список звёзд, доступных для наблюдения. Можно указать фильтр по названию.
// @Tags Stars
// @Accept json
// @Produce json
// @Param star_name query string false "Фильтр по названию звезды"
// @Success 200 {array} models.Star
// @Failure 500 {object} map[string]string "Ошибка получения звёзд"
// @Router /stars [get]
func getStars(c *gin.Context) {
	var stars []models.Star

	starName := c.Query("star_name")
	query := db.Model(&models.Star{})

	if starName != "" {
		query = query.Where("star_name ILIKE ?", "%"+starName+"%")
	}

	if err := query.Find(&stars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения звёзд: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, stars)
}

// getStarByID godoc
// @Summary Получить звезду по ID
// @Description Возвращает информацию о конкретной звезде по её ID
// @Tags Stars
// @Accept json
// @Produce json
// @Param id path int true "ID звезды"
// @Success 200 {object} models.Star
// @Failure 404 {object} map[string]string "Звезда не найдена"
// @Router /stars/{id} [get]
func getStarByID(c *gin.Context) {
	id := c.Param("id")
	var star models.Star
	if err := db.First(&star, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Звезда не найдена"})
		return
	}
	c.JSON(http.StatusOK, star)
}

// createStar godoc
// @Summary Добавить новую звезду
// @Description Создаёт новую запись звезды в базе данных
// @Tags Stars
// @Accept json
// @Produce json
// @Param star body models.Star true "Данные звезды"
// @Success 201 {object} map[string]interface{} "Звезда успешно добавлена"
// @Failure 400 {object} map[string]string "Некорректный JSON или отсутствует название"
// @Failure 500 {object} map[string]string "Ошибка при сохранении"
// @Router /stars [post]
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

// updateStar godoc
// @Summary Обновить данные звезды
// @Description Обновляет информацию о звезде по ID
// @Tags Stars
// @Accept json
// @Produce json
// @Param id path int true "ID звезды"
// @Param star body models.Star true "Обновлённые данные звезды"
// @Success 200 {object} map[string]interface{} "Звезда успешно обновлена"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 404 {object} map[string]string "Звезда не найдена"
// @Failure 500 {object} map[string]string "Ошибка при обновлении"
// @Router /stars/{id} [put]
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

// deleteStar godoc
// @Summary Удалить звезду
// @Description Удаляет звезду по ID
// @Tags Stars
// @Accept json
// @Produce json
// @Param id path int true "ID звезды"
// @Success 200 {object} map[string]string "Звезда успешно удалена"
// @Failure 400 {object} map[string]string "Некорректный ID"
// @Failure 404 {object} map[string]string "Звезда не найдена"
// @Failure 500 {object} map[string]string "Ошибка при удалении"
// @Router /stars/{id} [delete]
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

// uploadStarImage godoc
// @Summary Загрузить изображение звезды
// @Description Загружает изображение звезды в MinIO и сохраняет путь в БД
// @Tags Stars
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID звезды"
// @Param image formData file true "Файл изображения"
// @Success 200 {object} map[string]string "Изображение успешно загружено"
// @Failure 400 {object} map[string]string "Некорректный ID или файл"
// @Failure 500 {object} map[string]string "Ошибка при загрузке или сохранении"
// @Router /stars/{id}/image [post]
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

// addStarToDraftOrder godoc
// @Summary Добавить звезду в черновик заявки
// @Description Добавляет звезду в текущую заявку пользователя (черновик). Если уже есть — увеличивает количество.
// @Tags Stars
// @Accept json
// @Produce json
// @Param id path int true "ID звезды"
// @Success 200 {object} map[string]string "Звезда успешно добавлена или количество увеличено"
// @Failure 400 {object} map[string]string "Некорректный ID"
// @Failure 500 {object} map[string]string "Ошибка при работе с заявкой"
// @Router /stars/{id}/add [post]
func addStarToDraftOrder(c *gin.Context) {
	userID := auth.CurrentUserID()
	starID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID звезды"})
		return
	}

	order, err := repo.GetOrCreateDraftOrder(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения или создания черновика: " + err.Error()})
		return
	}

	var relation models.TelescopeObservationStar
	err = db.Where("telescope_observation_id = ? AND star_id = ?", order.TelescopeObservationID, starID).
		First(&relation).Error

	if err == nil {
		relation.Quantity += 1
		if err := db.Save(&relation).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления количества: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Количество звезды увеличено"})
		return
	}

	relation = models.TelescopeObservationStar{
		TelescopeObservationID: order.TelescopeObservationID,
		StarID:                 starID,
		OrderNumber:            1,
		Quantity:               1,
	}

	if err := db.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка добавления звезды в заявку: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Звезда успешно добавлена в черновик заявки"})
}
