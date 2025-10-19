package handler

import (
	"Lab1/internal/app/models"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Получение списка заявок
func (h *Handler) GetOrders(ctx *gin.Context) {
	var orders []models.Observation
	var err error

	// query параметр — фильтр по статусу
	searchStatus := ctx.Query("status")

	if searchStatus == "" {
		orders, err = h.Repository.GetOrders()
		if err != nil {
			logrus.Error("Ошибка получения всех заявок: ", err)
			ctx.String(http.StatusInternalServerError, "Ошибка получения заявок")
			return
		}
	} else {
		orders, err = h.Repository.GetOrdersByStatus(searchStatus)
		if err != nil {
			logrus.Error("Ошибка поиска заявок по статусу: ", err)
			ctx.String(http.StatusInternalServerError, "Ошибка поиска заявок")
			return
		}
	}

	ctx.HTML(http.StatusOK, "pageOrders.html", gin.H{
		"time":   time.Now().Format("15:04:05"),
		"orders": orders,
		"status": searchStatus,
	})
}

// Получение одной заявки по ID
func (h *Handler) GetOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error("Некорректный ID заявки: ", err)
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	// Загружаем заявку с привязкой звёзд через ObservationStars
	var order models.Observation
	err = h.Repository.DB.
		Preload("ObservationStars.Star").
		Preload("Creator").
		Preload("Moderator").
		First(&order, id).Error
	if err != nil {
		logrus.Error("Ошибка получения заявки: ", err)
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}

	if order.Status == "удалён" {
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}

	ctx.HTML(http.StatusOK, "shoppingCartPageWithApplications.html", gin.H{
		"order": order,
	})
}

// Создание новой заявки (черновик)
func (h *Handler) CreateOrder(ctx *gin.Context) {
	var newOrder models.Observation

	if err := ctx.ShouldBind(&newOrder); err != nil {
		ctx.String(http.StatusBadRequest, "Ошибка данных формы")
		return
	}

	newOrder.Status = "черновик"
	newOrder.CreatedAt = time.Now()

	if err := h.Repository.CreateOrder(&newOrder); err != nil {
		logrus.Error("Ошибка создания заявки: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка создания заявки")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// Обновление заявки (например, смена статуса)
func (h *Handler) UpdateOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	order, err := h.Repository.GetOrder(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}

	var input struct {
		Status string `form:"status"`
	}
	if err := ctx.ShouldBind(&input); err != nil {
		ctx.String(http.StatusBadRequest, "Ошибка формы")
		return
	}

	order.Status = input.Status
	if err := h.Repository.UpdateOrder(order); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка обновления заявки")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// Логическое удаление заявки (через SQL)
func (h *Handler) DeleteOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error("Некорректный ID заявки: ", err)
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	if err := h.Repository.DeleteOrder(id); err != nil {
		logrus.Error("Ошибка при логическом удалении заявки: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка при удалении заявки")
		return
	}

	logrus.Infof("Заявка #%d успешно логически удалена", id)
	ctx.Redirect(http.StatusSeeOther, "/stars")
}

func (h *Handler) CompleteOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	order, err := h.Repository.GetOrder(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}

	if order.Status != "черновик" {
		ctx.String(http.StatusBadRequest, "Можно завершить только черновик")
		return
	}

	// вычисляем результат для каждой звезды
	for _, obsStar := range order.ObservationStars {
		result := h.calculateResult(order, obsStar.Star)
		err := h.Repository.UpdateObservationStarResult(order.ObservationID, obsStar.Star.StarID, result)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Ошибка сохранения результата")
			return
		}
	}

	// обновляем статус заявки
	order.Status = "завершён"
	now := time.Now()
	order.CompletionDate = &now

	if err := h.Repository.UpdateOrder(order); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка завершения заявки")
		return
	}

	ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/order/%d", order.ObservationID))
}

func (h *Handler) calculateResult(order *models.Observation, star models.Star) float64 {
	// Пример формулы:
	// "Результат" = √(RA² + Dec²) * (1 + |широта - долгота| / 180)
	delta := math.Abs(order.ObserverLatitude - order.ObserverLongitude)
	value := math.Sqrt(math.Pow(star.RA, 2)+math.Pow(star.Dec, 2)) * (1 + delta/180)
	return math.Round(value*100) / 100 // округляем
}
