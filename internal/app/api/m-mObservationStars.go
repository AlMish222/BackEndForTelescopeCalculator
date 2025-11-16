package api

import (
	"Lab1/internal/app/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func InitMMObservationStarsAPI(r *gin.RouterGroup, repo *repository.Repository) {
	group := r.Group("/telescope-observation-stars")
	{
		group.DELETE("", func(c *gin.Context) { deleteObservationStar(c, repo) })
		group.PUT("", func(c *gin.Context) { putObservationStar(c, repo) })
	}
}

// @Summary Удалить услугу из заявки
// @Description Удаляет связь звезды с наблюдением (услугу)
// @Tags TelescopeObservationStars
// @Produce json
// @Param telescope_observation_id query int true "ID заявки"
// @Param star_id query int true "ID звезды"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /telescope-observation-stars [delete]
// @Security BearerAuth
func deleteObservationStar(c *gin.Context, repo *repository.Repository) {
	obsStr := c.Query("telescope_observation_id")
	starStr := c.Query("star_id")
	obsID, err := strconv.Atoi(obsStr)
	if err != nil || starStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужны telescope_observation_id и star_id"})
		return
	}
	starID, _ := strconv.Atoi(starStr)

	if err := repo.DeleteTelescopeObservationStar(obsID, starID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Услуга удалена из заявки"})
}

// / @Summary Обновить услугу заявки
// @Description Обновляет поля услуги в заявке (order_number, quantity, result_value)
// @Tags TelescopeObservationStars
// @Accept json
// @Produce json
// @Param input body map[string]interface{} true "Поля для обновления"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /telescope-observation-stars [put]
// @Security BearerAuth
func putObservationStar(c *gin.Context, repo *repository.Repository) {
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

	if err := repo.UpdateTelescopeObservationStar(obsID, starID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "М-М запись обновлена"})
}
