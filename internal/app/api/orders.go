package api

import (
	"Lab1/internal/app/auth"
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
	orders := r.Group("/orders")
	{
		orders.GET("/cart", getCartInfo)

		orders.GET("", getAllOrders)
		orders.GET("/:id", getOrderByID)
		orders.PUT("/:id", updateOrderFields)
		orders.PUT("/:id/submit", submitOrder) // ✅ сформировать
		orders.PUT("/:id/complete", completeOrder)
		orders.DELETE("/:id", deleteOrder)

		orders.DELETE("/observation-stars", deleteObservationStar)
		orders.PUT("/observation-stars", putObservationStar)
	}
}

func getCartInfo(c *gin.Context) {
	userID := auth.CurrentUserID()

	order, err := repo.GetOrCreateDraftOrder(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении черновика: " + err.Error()})
		return
	}

	var count int64
	if err := db.Model(&models.ObservationStar{}).
		Where("observation_id = ?", order.ObservationID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подсчёта услуг: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": order.ObservationID,
		"count":    count,
	})
}

func getAllOrders(c *gin.Context) {
	var orders []models.Observation

	// Параметры фильтра из URL
	from := c.Query("from")
	to := c.Query("to")
	status := c.Query("status")

	query := db.Model(&models.Observation{})

	// Фильтр по дате формирования
	if from != "" && to != "" {
		query = query.Where("formation_date BETWEEN ? AND ?", from, to)
	} else if from != "" {
		query = query.Where("formation_date >= ?", from)
	} else if to != "" {
		query = query.Where("formation_date <= ?", to)
	}

	// Фильтр по статусу
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Исключаем удалённые заявки
	query = query.Where("status != ?", "удалён")

	if err := query.Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения заявок: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func getOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var order models.Observation
	if err := db.
		Preload("ObservationStars.Star"). // предзагрузим связи м-м и звёзды с image_url
		Preload("Creator").
		Preload("Moderator").
		First(&order, "observation_id = ? AND status <> ?", id, "удалён").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func updateOrderFields(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	// Принимаем JSON с полями по предметной области. Системные поля (id, status, creator, moderator, даты) не должны изменяться.
	var payload map[string]interface{}
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}

	// Удаляем системные ключи, если они случайно пришли
	delete(payload, "observation_id")
	delete(payload, "creator_id")
	delete(payload, "moderator_id")
	//delete(payload, "status")
	//delete(payload, "created_at")
	//delete(payload, "formation_date")
	//delete(payload, "completion_date")

	if err := db.Model(&models.Observation{}).
		Where("observation_id = ?", id).
		Updates(payload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка обновлена"})
}

func submitOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	order, err := repo.GetOrder(id)
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

	formation := time.Now()
	order.Status = "сформирован"
	order.FormationDate = &formation

	if err := repo.UpdateOrder(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при формировании заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка успешно сформирована",
		"orderID": order.ObservationID,
	})
}

func completeOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	var req struct {
		Action string `json:"action"` // "complete" или "reject"
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON"})
		return
	}

	order, err := repo.GetOrder(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if order.Status != "сформирован" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно завершить/отклонить только сформированную заявку"})
		return
	}

	moderatorID := 2 // временно
	completion := time.Now()

	// Если модератор отклоняет заявку
	if req.Action == "reject" {
		order.Status = "отклонён"
		order.ModeratorID = &moderatorID
		order.CompletionDate = &completion

		if err := repo.UpdateOrder(order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
		return
	}

	// Загружаем все звёзды заявки
	var obsStars []models.ObservationStar
	if err := db.Preload("Star").
		Where("observation_id = ?", id).
		Find(&obsStars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки звёзд заявки: " + err.Error()})
		return
	}

	// Расчёт результата (аналогично handler.calculateResult)
	for _, os := range obsStars {
		delta := math.Abs(os.ObserverLatitude - os.ObserverLongitude)
		value := math.Sqrt(math.Pow(os.Star.RA, 2)+math.Pow(os.Star.Dec, 2)) * (1 + delta/180)
		result := math.Round(value*100) / 100

		if err := repo.UpdateObservationStarResult(id, os.StarID, result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результата: " + err.Error()})
			return
		}
	}

	// Обновляем саму заявку
	order.Status = "завершён"
	order.ModeratorID = &moderatorID
	order.CompletionDate = &completion

	if err := repo.UpdateOrder(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при завершении заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка завершена успешно",
		"orderID": order.ObservationID,
	})
}

func deleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}

	if err := repo.DeleteOrder(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка помечена как удалённая",
		"orderID": id,
	})
}

// удаление услуги из заявки
func deleteObservationStar(c *gin.Context) {
	obsStr := c.Query("observation_id")
	starStr := c.Query("star_id")
	obsID, err := strconv.Atoi(obsStr)
	if err != nil || starStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "обязательно observation_id и star_id"})
		return
	}
	starID, _ := strconv.Atoi(starStr)

	if err := repo.DeleteObservationStar(obsID, starID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления записи: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Услуга удалена из заявки"})
}

// PUT /api/orders/observation-stars
// Body JSON: { "observation_id":1, "star_id":2, "quantity":3, "order_number":1, "result_value":12.34 }
func putObservationStar(c *gin.Context) {
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный JSON: " + err.Error()})
		return
	}
	oi, ok1 := req["observation_id"]
	si, ok2 := req["star_id"]
	if !ok1 || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нужны observation_id и star_id"})
		return
	}
	obsID := int(int64(oi.(float64))) // JSON numbers -> float64
	starID := int(int64(si.(float64)))

	// Удаляем ключи, которые не относятcя к полям м-м
	delete(req, "observation_id")
	delete(req, "star_id")

	// Разрешённые поля: order_number, quantity, result_value, observer_latitude, observer_longitude
	allowed := map[string]bool{
		"order_number":       true,
		"quantity":           true,
		"result_value":       true,
		"observer_latitude":  true,
		"observer_longitude": true,
	}
	updates := map[string]interface{}{}
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нет полей для обновления"})
		return
	}

	if err := repo.UpdateObservationStar(obsID, starID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "М-М запись обновлена"})
}
