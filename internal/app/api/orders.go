package api

import (
	"Lab1/internal/app/auth"
	"Lab1/internal/app/middleware"
	"Lab1/internal/app/models"
	"Lab1/internal/app/repository"
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
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

	accuracyResults := r.Group("/telescopeObservations")
	{
		accuracyResults.POST("/:id/accuracy-results", receiveAccuracyResults)
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

	query := db.Model(&models.TelescopeObservation{}).
		Preload("TelescopeObservationStars")

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

	// === ДОБАВЛЯЕМ ПОЛЕ completed_stars_count ===
	type ObservationWithCount struct {
		models.TelescopeObservation
		CompletedStarsCount int `json:"completed_stars_count"`
		TotalStars          int `json:"total_stars"` // ← НОВОЕ ПОЛЕ
	}

	response := make([]ObservationWithCount, len(orders))

	for i, order := range orders {
		// Считаем количество звёзд с заполненным result_value
		var completedCount int64
		db.Model(&models.TelescopeObservationStar{}).
			Where("telescope_observation_id = ? AND result_value IS NOT NULL",
				order.TelescopeObservationID).
			Count(&completedCount)

		// Общее количество звёзд в заявке
		totalStars := len(order.TelescopeObservationStars)

		response[i] = ObservationWithCount{
			TelescopeObservation: order,
			CompletedStarsCount:  int(completedCount),
			TotalStars:           totalStars, // ← Добавляем
		}
	}

	c.JSON(http.StatusOK, response)
}

type ObservationWithCount struct {
	models.TelescopeObservation
	CompletedStarsCount int `json:"completed_stars_count"`
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

	// === ДОБАВЛЯЕМ: Подсчёт completed_stars_count ===
	var completedCount int64
	err = db.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ? AND result_value IS NOT NULL", id).
		Count(&completedCount).Error

	if err != nil {
		// Логируем ошибку, но не прерываем выполнение
		log.Printf("Error counting completed stars: %v", err)
		completedCount = 0
	}

	type StarResponse struct {
		StarID           int      `json:"starId"`
		StarName         string   `json:"starName"`
		ImageURL         string   `json:"imageUrl"`
		ShortDescription string   `json:"shortDescription"`
		Description      string   `json:"description"`
		RA               float64  `json:"ra"`
		Dec              float64  `json:"dec"`
		Quantity         int      `json:"quantity"`
		OrderNumber      int      `json:"orderNumber"`
		ResultValue      *float64 `json:"resultValue"` // Изменяем на указатель
	}

	type Response struct {
		Stars                  []StarResponse `json:"stars"`
		ObserverLatitude       float64        `json:"observerLatitude"`
		ObserverLongitude      float64        `json:"observerLongitude"`
		ObservationDate        *time.Time     `json:"observationDate"`
		CompletedStarsCount    int            `json:"completedStarsCount"` // НОВОЕ ПОЛЕ
		TotalStars             int            `json:"totalStars"`
		TelescopeObservationID int            `json:"telescopeObservationId"`
		Status                 string         `json:"status"`
		CreatedAt              time.Time      `json:"createdAt"`
	}

	var starsResponse []StarResponse
	for _, observationStar := range order.TelescopeObservationStars {
		// Используем реальное значение ResultValue из БД
		var resultValue *float64

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
			ResultValue:      resultValue, // Реальное значение или nil
		})
	}

	response := Response{
		Stars:                  starsResponse,
		ObserverLatitude:       order.ObserverLatitude,
		ObserverLongitude:      order.ObserverLongitude,
		ObservationDate:        order.ObservationDate,
		CompletedStarsCount:    int(completedCount), // Добавляем счётчик
		TotalStars:             len(order.TelescopeObservationStars),
		TelescopeObservationID: order.TelescopeObservationID,
		Status:                 order.Status,
		CreatedAt:              order.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
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
	delete(payload, "status")
	delete(payload, "created_at")
	delete(payload, "formation_date")
	delete(payload, "completion_date")

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

	if order.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Только создатель может сформировать заявку"})
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

	if order.CreatorID == userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Создатель не может выступать модератором для своей заявки"})
		return
	}

	now := time.Now()

	if req.Action == "reject" {
		order.Status = "отклонён"
		order.ModeratorID = &userID
		order.CompletionDate = &now

		if err := repo.UpdateTelescopeObservation(order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонения: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
		return
	}

	// === ЗАПУСК АСИНХРОННОГО РАСЧЁТА ===

	// Просто берём токен из заголовка
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Токен не найден"})
		return
	}

	authToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Вызываем асинхронный сервис Django
	asyncSuccess := callAsyncService(id, authToken)
	if !asyncSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось запустить асинхронный расчёт"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Запущен асинхронный расчёт точности наблюдения",
		"status":         "processing",
		"estimated_time": "5-10 секунд",
		"observation_id": id,
		"current_status": "сформирован (расчёт в процессе)",
		"next_step":      "Результаты будут сохранены автоматически",
	})
}

func callAsyncService(observationID int, authToken string) bool {
	// URL Django сервиса
	djangoURL := "http://localhost:9010/api/calculate/"

	payload := map[string]interface{}{
		"observation_id": observationID,
		"auth_token":     authToken,           // UUID токен пользователя из Redis
		"async_token":    "async_secret_2024", // Дополнительный токен для Django->Go коммуникации
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling async request: %v", err)
		return false
	}

	// Добавляем таймаут
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(djangoURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error calling async service: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Логируем для отладки
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Async service error %d: %s", resp.StatusCode, body)
		return false
	}

	log.Printf("Async calculation started for observation %d", observationID)
	return true
}

func receiveAccuracyResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	// Проверяем существование заявки
	order, err := repo.GetTelescopeObservation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	var req struct {
		AuthToken string `json:"auth_token"`
		Results   []struct {
			StarID      int     `json:"star_id"`
			ResultValue float64 `json:"result_value"`
		} `json:"results"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}

	// Проверка токена (псевдо-авторизация по ТЗ)
	const asyncToken = "async_secret_2024"
	if req.AuthToken != asyncToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен асинхронного сервиса"})
		return
	}

	// Обновляем result_value для каждой звезды
	successCount := 0
	for _, result := range req.Results {
		err := repo.UpdateTelescopeObservationStarResult(id, result.StarID, result.ResultValue)
		if err != nil {
			log.Printf("Error updating star %d result: %v", result.StarID, err)
			// Продолжаем для остальных звёзд
			continue
		}
		successCount++
	}

	// Меняем статус заявки на "завершён" если она ещё не завершена
	if successCount > 0 && order.Status == "сформирован" {
		moderatorID := 2 // ID системного модератора
		now := time.Now()

		order.Status = "завершён"
		order.ModeratorID = &moderatorID
		order.CompletionDate = &now

		if err := repo.UpdateTelescopeObservation(order); err != nil {
			log.Printf("Error updating observation status: %v", err)
		} else {
			log.Printf("Observation %d status changed to 'завершён'", id)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Результаты успешно сохранены",
		"stars_updated":  successCount,
		"total_stars":    len(req.Results),
		"observation_id": id,
		"new_status":     "завершён",
	})
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
