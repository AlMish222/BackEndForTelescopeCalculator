package handler

import (
	"Lab1/internal/app/auth"
	"Lab1/internal/app/models"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Добавление звезды в корзину со статусом "черновик"
func (h *Handler) AddStarToDraftOrder(ctx *gin.Context) {
	starIDStr := ctx.Param("id")
	userID := auth.CurrentUserID()

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
	var existing models.TelescopeObservationStar
	err = h.Repository.DB.
		Where("telescope_observation_id = ? AND star_id = ?", order.TelescopeObservationID, starID).
		First(&existing).Error

	if err == nil {
		// Есть такая звезда в заказе — увеличиваем количество
		existing.Quantity += 1
		if saveErr := h.Repository.DB.Save(&existing).Error; saveErr != nil {
			logrus.Error("Ошибка при обновлении количества звезды в корзине: ", saveErr)
			ctx.String(http.StatusInternalServerError, "Ошибка при обновлении количества")
			return
		}

		var cartCount int64
		if countErr := h.Repository.DB.Model(&models.TelescopeObservationStar{}).
			Where("telescope_observation_id = ?", order.TelescopeObservationID).
			Count(&cartCount).Error; countErr != nil {
			logrus.Error("Ошибка пересчёта корзины: ", countErr)
		}

		ctx.Redirect(http.StatusFound, "/stars")
		return
	}

	// Добавляем новую связь
	if errors.Is(err, gorm.ErrRecordNotFound) {
		relation := models.TelescopeObservationStar{
			TelescopeObservationID: order.TelescopeObservationID,
			StarID:                 starID,
			OrderNumber:            1,
			Quantity:               1,
		}

		if createErr := h.Repository.DB.Create(&relation).Error; createErr != nil {
			logrus.Error("Ошибка добавления звезды в корзину: ", createErr)
			ctx.String(http.StatusInternalServerError, "Ошибка добавления звезды в корзину")
			return
		}

		var cartCount int64
		if countErr := h.Repository.DB.Model(&models.TelescopeObservationStar{}).
			Where("telescope_observation_id = ?", order.TelescopeObservationID).
			Count(&cartCount).Error; countErr != nil {
			logrus.Error("Ошибка пересчёта корзины: ", countErr)
		}

		ctx.Redirect(http.StatusSeeOther, "/stars")
		return
	}

	// Любая другая ошибка
	logrus.Error("Ошибка проверки звезды в корзине: ", err)
	ctx.String(http.StatusInternalServerError, "Ошибка при проверке звезды в корзине")
}

func (h *Handler) GetStars(ctx *gin.Context) {
	userID := auth.CurrentUserID()

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

	hasDraft, draftID, cartCount, err := h.Repository.GetObservationInfo(userID)
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
	userID := auth.CurrentUserID()
	hasDraft, draftID, cartCount, err := h.Repository.GetObservationInfo(userID)
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
