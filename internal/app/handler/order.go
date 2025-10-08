package handler

import (
	"Lab1/internal/app/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ==============================
// Получение списка заявок
// ==============================
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

// ==============================
// Получение одной заявки по ID
// ==============================
func (h *Handler) GetOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error("Некорректный ID заявки: ", err)
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	order, err := h.Repository.GetOrder(id)
	if err != nil {
		logrus.Error("Ошибка получения заявки: ", err)
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}

	ctx.HTML(http.StatusOK, "pageOrderDetail.html", gin.H{
		"order": order,
	})
}

// ==============================
// Создание новой заявки (черновик)
// ==============================
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

// ==============================
// Обновление заявки (например, смена статуса)
// ==============================
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

// ==============================
// Логическое удаление заявки (через SQL)
// ==============================
func (h *Handler) DeleteOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	// логическое удаление через SQL UPDATE, не ORM
	sql := `UPDATE observations SET status = 'удалён' WHERE id = $1`
	if err := h.Repository.DB.Exec(sql, id).Error; err != nil {
		logrus.Error("Ошибка логического удаления: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка удаления")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}
