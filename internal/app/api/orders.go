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

		orders.DELETE("/telescope-observation-stars", deleteObservationStar)
		orders.PUT("/telescope-observation-stars", putObservationStar)
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

func getAllOrders(c *gin.Context) {
	var orders []models.TelescopeObservation

	from := c.Query("from")
	to := c.Query("to")
	status := c.Query("status")

	query := db.Model(&models.TelescopeObservation{})

	if from != "" && to != "" {
		query = query.Where("formation_date BETWEEN ? AND ?", from, to)
	} else if from != "" {
		query = query.Where("formation_date >= ?", from)
	} else if to != "" {
		query = query.Where("formation_date <= ?", to)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

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

	var order models.TelescopeObservation
	if err := db.
		Preload("TelescopeObservationStars.Star").
		Preload("Creator").
		Preload("Moderator").
		First(&order, "telescope_observation_id = ? AND status <> ?", id, "удалён").Error; err != nil {
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

	now := time.Now()
	order.Status = "сформирован"
	order.FormationDate = &now

	if err := repo.UpdateOrder(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при формировании заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка успешно сформирована",
		"id":      order.TelescopeObservationID,
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
		Action string `json:"action"`
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

	moderatorID := 2
	now := time.Now()

	if req.Action == "reject" {
		order.Status = "отклонён"
		order.ModeratorID = &moderatorID
		order.CompletionDate = &now

		if err := repo.UpdateOrder(order); err != nil {
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

		if err := repo.UpdateObservationStarResult(id, s.StarID, result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результата: " + err.Error()})
			return
		}
	}

	order.Status = "завершён"
	order.ModeratorID = &moderatorID
	order.CompletionDate = &now

	if err := repo.UpdateOrder(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при завершении заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка завершена успешно"})
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

	c.JSON(http.StatusOK, gin.H{"message": "Заявка помечена как удалённая"})
}

// удаление услуги из заявки
func deleteObservationStar(c *gin.Context) {
	obsStr := c.Query("telescope_observation_id")
	starStr := c.Query("star_id")
	obsID, err := strconv.Atoi(obsStr)
	if err != nil || starStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужны telescope_observation_id и star_id"})
		return
	}
	starID, _ := strconv.Atoi(starStr)

	if err := repo.DeleteObservationStar(obsID, starID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления: " + err.Error()})
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
	oi, ok1 := req["telescope_observation_id"]
	si, ok2 := req["star_id"]
	if !ok1 || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нужны telescope_observation_id и star_id"})
		return
	}
	obsID := int(int64(oi.(float64)))
	starID := int(int64(si.(float64)))

	delete(req, "telescope_observation_id")
	delete(req, "star_id")

	allowed := map[string]bool{
		"order_number": true,
		"quantity":     true,
		"result_value": true,
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
