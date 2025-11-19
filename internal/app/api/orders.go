package api

import (
	"Lab1/internal/app/auth"
	"Lab1/internal/app/middleware"
	"Lab1/internal/app/models"
	"Lab1/internal/app/repository"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitOrderAPI(database *gorm.DB, r *gin.RouterGroup) {
	db = database
	repo = repository.NewRepositoryFromDB(db)
	registerOrderRoutes(r)
}

func registerOrderRoutes(r *gin.RouterGroup) {
	telescopeObservation := r.Group("/telescopeObservations")
	telescopeObservation.Use(middleware.AuthMiddleware())
	{
		telescopeObservation.GET("/cart", getTelescopeObservationInfo)
		telescopeObservation.GET("", getAllTelescopeObservations)
		telescopeObservation.GET("/:id", getTelescopeObservationByID)
		telescopeObservation.PUT("/:id", updateTelescopeObservationFields)
		telescopeObservation.PUT("/:id/submit", submitTelescopeObservation)

		telescopeObservation.PUT("/:id/complete", middleware.RequireModerator(), completeTelescopeObservation)
		telescopeObservation.DELETE("/:id", middleware.RequireModerator(), deleteTelescopeObservation)
	}
}

// @Summary Получить информацию о корзине пользователя
// @Description Возвращает черновик заявки и количество услуг в нём
// @Tags TelescopeObservations
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /telescopeObservations/cart [get]
// @Security BearerAuth
func getTelescopeObservationInfo(c *gin.Context) {
	userID := auth.CurrentUserID()

	order, err := repo.GetOrCreateDraftTelescopeObservation(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении черновика: " + err.Error()})
		return
	}

	var count int64
	if err := db.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ?", order.TelescopeObservationID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подсчёта услуг: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"telescope_observation_id": order.TelescopeObservationID,
		"count":                    count,
	})
}

// @Summary Получить все заявки
// @Description Возвращает список всех заявок (с фильтрацией по дате и статусу)
// @Tags TelescopeObservations
// @Produce json
// @Param from query string false "Дата начала (YYYY-MM-DD)"
// @Param to query string false "Дата конца (YYYY-MM-DD)"
// @Param status query string false "Статус заявки"
// @Success 200 {array} models.TelescopeObservation
// @Failure 500 {object} map[string]string
// @Router /telescopeObservations [get]
// @Security BearerAuth
func getAllTelescopeObservations(c *gin.Context) {
	var orders []models.TelescopeObservation

	isModAny, _ := c.Get("is_moderator")
	isModerator := isModAny.(bool)

	from := c.Query("from")
	to := c.Query("to")
	status := c.Query("status")

	query := db.Model(&models.TelescopeObservation{})

	if !isModerator {
		query = query.Where("moderator_id IS NULL")
	}

	if from != "" && to != "" {
		query = query.Where("formation_date BETWEEN ? AND ?", from, to)
	} else if from != "" {
		query = query.Where("formation_date >= ?", from)
	} else if to != "" {
		query = query.Where("formation_date <= ?", to)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status != ?", "удалён")
	}

	if err := query.Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения заявок: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// @Summary Получить заявку по ID
// @Description Возвращает данные конкретной заявки со связанными звёздами и пользователями
// @Tags TelescopeObservations
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} models.TelescopeObservation
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /telescopeObservations/{id} [get]
// @Security BearerAuth
func getTelescopeObservationByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var order models.TelescopeObservation
	if err := db.
		Preload("TelescopeObservationStars.Star").
		First(&order, "telescope_observation_id = ? AND status <> ?", id, "удалён").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	type StarResponse struct {
		StarID           int     `json:"starId"`
		StarName         string  `json:"starName"`
		ImageURL         string  `json:"imageUrl"`
		ShortDescription string  `json:"shortDescription"`
		Description      string  `json:"description"`
		RA               float64 `json:"ra"`
		Dec              float64 `json:"dec"`
		Quantity         int     `json:"quantity"`
		OrderNumber      int     `json:"orderNumber"`
	}

	type Response struct {
		Stars []StarResponse `json:"stars"`
	}

	var starsResponse []StarResponse
	for _, observationStar := range order.TelescopeObservationStars {
		starsResponse = append(starsResponse, StarResponse{
			StarID:           observationStar.Star.StarID,
			StarName:         observationStar.Star.StarName,
			ImageURL:         observationStar.Star.ImageURL,
			ShortDescription: observationStar.Star.ShortDescription,
			Description:      observationStar.Star.Description,
			RA:               observationStar.Star.RA,
			Dec:              observationStar.Star.Dec,
			Quantity:         observationStar.Quantity,
			OrderNumber:      observationStar.OrderNumber,
		})
	}

	c.JSON(http.StatusOK, Response{Stars: starsResponse})
}

// @Summary Обновить поля заявки
// @Description Обновляет произвольные поля заявки (кроме ID и связей)
// @Tags TelescopeObservations
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param input body map[string]interface{} true "Поля для обновления"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /telescopeObservations/{id} [put]
// @Security BearerAuth
func updateTelescopeObservationFields(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var payload map[string]interface{}
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}

	delete(payload, "telescope_observation_id")
	delete(payload, "creator_id")
	delete(payload, "moderator_id")

	if err := db.Model(&models.TelescopeObservation{}).
		Where("telescope_observation_id = ?", id).
		Updates(payload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка обновлена"})
}

// @Summary Сформировать заявку
// @Description Переводит заявку из состояния 'черновик' в 'сформирован'
// @Tags TelescopeObservations
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /telescopeObservations/{id}/submit [put]
// @Security BearerAuth
func submitTelescopeObservation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	order, err := repo.GetTelescopeObservation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if order.Status == "удалён" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя сформировать удалённую заявку"})
		return
	}
	if order.Status != "черновик" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно сформировать только черновик"})
		return
	}

	now := time.Now()
	order.Status = "сформирован"
	order.FormationDate = &now

	if err := repo.UpdateTelescopeObservation(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при формировании заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка успешно сформирована",
		"id":      order.TelescopeObservationID,
	})
}

// @Summary Завершить или отклонить заявку
// @Description Доступно только модератору. Завершает или отклоняет сформированную заявку
// @Tags TelescopeObservations
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param input body map[string]string true "Действие (action=reject или complete)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /telescopeObservations/{id}/complete [put]
// @Security BearerAuth
func completeTelescopeObservation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON"})
		return
	}

	order, err := repo.GetTelescopeObservation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if order.Status != "сформирован" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно завершить/отклонить только сформированную заявку"})
		return
	}

	moderatorID := 2
	now := time.Now()

	if req.Action == "reject" {
		order.Status = "отклонён"
		order.ModeratorID = &moderatorID
		order.CompletionDate = &now

		if err := repo.UpdateTelescopeObservation(order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
		return
	}

	var stars []models.TelescopeObservationStar
	if err := db.Preload("Star").
		Where("telescope_observation_id = ?", id).
		Find(&stars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки звёзд заявки: " + err.Error()})
		return
	}

	for _, s := range stars {
		value := math.Sqrt(math.Pow(s.Star.RA, 2) + math.Pow(s.Star.Dec, 2))
		result := math.Round(value*100) / 100

		if err := repo.UpdateTelescopeObservationStarResult(id, s.StarID, result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результата: " + err.Error()})
			return
		}
	}

	order.Status = "завершён"
	order.ModeratorID = &moderatorID
	order.CompletionDate = &now

	if err := repo.UpdateTelescopeObservation(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при завершении заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка завершена успешно"})
}

// @Summary Удалить заявку
// @Description Доступно только модератору. Помечает заявку как удалённую
// @Tags TelescopeObservations
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /telescopeObservations/{id} [delete]
// @Security BearerAuth
func deleteTelescopeObservation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	if err := repo.DeleteTelescopeObservation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка помечена как удалённая"})
}
