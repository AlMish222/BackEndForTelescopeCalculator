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
func (h *Handler) GetTelescopeObservations(ctx *gin.Context) {
	var telescopeObservations []models.TelescopeObservation
	var err error

	// query параметр — фильтр по статусу
	searchStatus := ctx.Query("status")

	if searchStatus == "" {
		telescopeObservations, err = h.Repository.GetTelescopeObservations()
		if err != nil {
			logrus.Error("Ошибка получения всех заявок: ", err)
			ctx.String(http.StatusInternalServerError, "Ошибка получения заявок")
			return
		}
	} else {
		telescopeObservations, err = h.Repository.GetTelescopeObservationsByStatus(searchStatus)
		if err != nil {
			logrus.Error("Ошибка поиска заявок по статусу: ", err)
			ctx.String(http.StatusInternalServerError, "Ошибка поиска заявок")
			return
		}
	}

	ctx.HTML(http.StatusOK, "pageOrders.html", gin.H{
		"time":                  time.Now().Format("15:04:05"),
		"telescopeObservations": telescopeObservations,
		"status":                searchStatus,
	})
}

// Получение одной корзины по ID
func (h *Handler) GetTelescopeObservation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error("Некорректный ID корзины: ", err)
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	// Загружаем заявку с привязкой звёзд через TelescopeObservationStars
	var telescopeObservation models.TelescopeObservation
	err = h.Repository.DB.
		Preload("TelescopeObservationStars.Star").
		Preload("Creator").
		Preload("Moderator").
		First(&telescopeObservation, id).Error
	if err != nil {
		logrus.Error("Ошибка получения корзины: ", err)
		ctx.String(http.StatusNotFound, "Корзина не найдена")
		return
	}

	if telescopeObservation.Status == "удалён" {
		ctx.String(http.StatusNotFound, "Корзина не найдена")
		return
	}

	ctx.HTML(http.StatusOK, "shoppingCartPageWithApplications.html", gin.H{
		"telescopeObservation": telescopeObservation,
	})
}

// Создание новой заявки (черновик)
func (h *Handler) CreateTelescopeObservation(ctx *gin.Context) {
	var newTelescopeObservation models.TelescopeObservation

	if err := ctx.ShouldBind(&newTelescopeObservation); err != nil {
		ctx.String(http.StatusBadRequest, "Ошибка данных формы")
		return
	}

	newTelescopeObservation.Status = "черновик"
	newTelescopeObservation.CreatedAt = time.Now()

	if err := h.Repository.CreateTelescopeObservation(&newTelescopeObservation); err != nil {
		logrus.Error("Ошибка создания корзины: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка создания корзины")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// Обновление заявки (например, смена статуса)
func (h *Handler) UpdateTelescopeObservation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	telescopeObservation, err := h.Repository.GetTelescopeObservation(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Корзина не найдена")
		return
	}

	var input struct {
		Status string `form:"status"`
	}
	if err := ctx.ShouldBind(&input); err != nil {
		ctx.String(http.StatusBadRequest, "Ошибка формы")
		return
	}

	telescopeObservation.Status = input.Status
	if err := h.Repository.UpdateTelescopeObservation(telescopeObservation); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка обновления корзины")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// Логическое удаление заявки (через SQL)
func (h *Handler) DeleteTelescopeObservation(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		logrus.Error("Некорректный ID корзины: ", err)
		ctx.String(http.StatusBadRequest, "Неверный ID")
		return
	}

	if err := h.Repository.DeleteTelescopeObservation(id); err != nil {
		logrus.Error("Ошибка при логическом удалении корзины: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка при удалении корзины")
		return
	}

	logrus.Infof("Корзина #%d успешно логически удалена", id)
	ctx.Redirect(http.StatusSeeOther, "/stars")
}

func (h *Handler) CompleteTelescopeObservation(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "Неверный ID корзины")
		return
	}

	telescopeObservation, err := h.Repository.GetTelescopeObservation(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Корзина не найдена")
		return
	}

	if telescopeObservation.Status != "черновик" {
		ctx.String(http.StatusBadRequest, "Можно завершить только черновик")
		return
	}

	// вычисляем результат для каждой звезды
	for _, obsStar := range telescopeObservation.TelescopeObservationStars {
		result := h.calculateResult(&obsStar, obsStar.Star)
		err := h.Repository.UpdateTelescopeObservationStarResult(telescopeObservation.TelescopeObservationID, obsStar.Star.StarID, result)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Ошибка сохранения результата")
			return
		}
	}

	// обновляем статус заявки
	telescopeObservation.Status = "завершён"
	now := time.Now()
	telescopeObservation.CompletionDate = &now

	if err := h.Repository.UpdateTelescopeObservation(telescopeObservation); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка завершения корзины")
		return
	}

	ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("/telescopeObservation/%d", telescopeObservation.TelescopeObservationID))
}

// расчёт индекса сложности наведения
func (h *Handler) calculateResult(obsStar *models.TelescopeObservationStar, star models.Star) float64 {
	// Берем широту и долготу из родительской заявки
	obs := obsStar.TelescopeObservation
	delta := math.Abs(obs.ObserverLatitude - obs.ObserverLongitude)

	// Пример формулы:
	// "Индекс сложности наведения" = √(RA² + Dec²) * (1 + |широта - долгота| / 180)
	value := math.Sqrt(math.Pow(star.RA, 2)+math.Pow(star.Dec, 2)) * (1 + delta/180)

	return math.Round(value*100) / 100 // округляем до сотых
}
