package handler

import (
	"Lab1/internal/app/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Добавление звезды в заявку со статусом "черновик"
func (h *Handler) AddStarToDraftOrder(ctx *gin.Context) {
	starIDStr := ctx.Param("id")
	userID := 1 // временный ID пользователя
	starID, err := strconv.Atoi(starIDStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Некорректный ID звезды")
		return
	}

	// Получаем или создаём черновик (корзину)
	order, err := h.Repository.GetOrCreateDraftOrder(userID)
	if err != nil {
		logrus.Error("Ошибка получения или создания черновика: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка работы с черновиком")
		return
	}

	// Проверяем, не добавлена ли звезда уже
	var existing models.ObservationStar
	err = h.Repository.DB.
		Where("observation_id = ? AND star_id = ?", order.ObservationID, starID).
		First(&existing).Error

	if err == nil {
		// Звезда уже есть в заказе
		ctx.Redirect(http.StatusSeeOther, "/stars")
		return
	}

	// Добавляем новую связь
	relation := models.ObservationStar{
		ObservationID: order.ObservationID,
		StarID:        starID,
		IsMain:        false,
		OrderNumber:   1,
		Quantity:      1,
	}

	if err := h.Repository.DB.Create(&relation).Error; err != nil {
		logrus.Error("Ошибка добавления звезды в заявку: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка добавления звезды в заявку")
		return
	}

	// Возвращаем обратно на страницу звёзд (без редиректа на заявку)
	ctx.Redirect(http.StatusSeeOther, "/stars")
}

func (h *Handler) GetStars(ctx *gin.Context) {
	userID := 1 // временный ID пользователя

	stars, err := h.Repository.GetStars()
	if err != nil {
		logrus.Error("Ошибка получения звёзд: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка загрузки звёзд")
		return
	}

	// Получаем информацию о корзине
	hasDraft, draftID, cartCount, err := h.Repository.GetCartInfo(userID)
	if err != nil {
		logrus.Error("Ошибка получения информации о корзине: ", err)
	}

	ctx.HTML(http.StatusOK, "pageStars.html", gin.H{
		"stars":     stars,
		"hasDraft":  hasDraft,
		"draftID":   draftID,
		"cartCount": cartCount,
	})
}

func (h *Handler) GetStarByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	starID, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Некорректный ID звезды")
		return
	}

	// Загружаем звезду по ID
	var star models.Star
	err = h.Repository.DB.
		Preload("Observations").
		First(&star, "star_id = ?", starID).Error
	if err != nil {
		logrus.Error("Ошибка при получении звезды: ", err)
		ctx.String(http.StatusNotFound, "Звезда не найдена")
		return
	}

	// Получаем данные корзины (черновик + количество элементов)
	userID := 1 // временный ID пользователя
	hasDraft, draftID, cartCount, err := h.Repository.GetCartInfo(userID)
	if err != nil {
		logrus.Error("Ошибка получения информации о корзине: ", err)
	}

	// Отображаем страницу звезды
	ctx.HTML(http.StatusOK, "pageStarDetail.html", gin.H{
		"star":      star,
		"hasDraft":  hasDraft,
		"draftID":   draftID,
		"cartCount": cartCount, // тут общее количество звёзд в корзине
	})
}
