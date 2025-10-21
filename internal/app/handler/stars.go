package handler

import (
	"Lab1/internal/app/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Добавление звезды в корзину со статусом "черновик"
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
		OrderNumber:   1,
		Quantity:      1,
	}
	if err := h.Repository.DB.Create(&relation).Error; err != nil {
		logrus.Error("Ошибка добавления звезды в корзину: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка добавления звезды в корзину")
		return
	}
	// Возвращаем обратно на страницу звёзд (без редиректа на заявку)
	ctx.Redirect(http.StatusSeeOther, "/stars")
}

func (h *Handler) GetStars(ctx *gin.Context) {
	userID := 1 // временный ID пользователя

	query := ctx.Query("query") // <-- добавляем строку поиска

	var stars []models.Star
	var err error

	if query == "" {
		stars, err = h.Repository.GetStars()
	} else {
		stars, err = h.Repository.SearchStars(query) // <-- новый метод
	}

	if err != nil {
		logrus.Error("Ошибка получения звёзд: ", err)
		ctx.String(http.StatusInternalServerError, "Ошибка загрузки звёзд")
		return
	}

	hasDraft, draftID, cartCount, err := h.Repository.GetCartInfo(userID)
	if err != nil {
		logrus.Error("Ошибка получения информации о корзине: ", err)
	}

	ctx.HTML(http.StatusOK, "pageStars.html", gin.H{
		"stars":     stars,
		"query":     query,
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

// +++API+++

func (h *Handler) ApiGetStars(c *gin.Context) {
	stars, err := h.Repository.GetStars()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения списка звёзд"})
		return
	}
	c.JSON(http.StatusOK, stars)
}

func (h *Handler) ApiGetStarByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID"})
		return
	}
	star, err := h.Repository.GetStarByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Звезда не найдена"})
		return
	}
	c.JSON(http.StatusOK, star)
}
