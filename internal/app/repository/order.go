package repository

import (
	"Lab1/internal/app/models"
	"time"

	"gorm.io/gorm"
)

// Получение всех заявок (observations)
func (r *Repository) GetOrders() ([]models.Observation, error) {
	var orders []models.Observation

	err := r.DB.
		Preload("Stars").
		Preload("Creator").
		Preload("Moderator").
		Where("status <> ?", "удалён").
		Order("created_at DESC").
		Find(&orders).Error

	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Получение корзины по ID
func (r *Repository) GetOrder(id int) (*models.Observation, error) {
	var order models.Observation

	err := r.DB.
		Preload("Stars").
		Preload("Creator").
		Preload("Moderator").
		Where("observation_id = ? AND status <> ?", id, "удалён").
		First(&order).Error

	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Получение корзин по статусу
func (r *Repository) GetOrdersByStatus(status string) ([]models.Observation, error) {
	var orders []models.Observation
	err := r.DB.Where("status = ?", status).Preload("Stars").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Создание новой корзины
func (r *Repository) CreateOrder(order *models.Observation) error {
	return r.DB.Create(order).Error
}

// Обновление корзины (например, статус или даты)
func (r *Repository) UpdateOrder(order *models.Observation) error {
	return r.DB.Save(order).Error
}

// Логическое удаление корзины
func (r *Repository) DeleteOrder(id int) error {
	return r.DB.Exec(`UPDATE observations SET status = 'удалён' WHERE observation_id = ?`, id).Error
}

// Получение или создание черновика
func (r *Repository) GetOrCreateDraftOrder(userID int) (*models.Observation, error) {
	var order models.Observation
	err := r.DB.Where("creator_id = ? AND status = ?", userID, "черновик").First(&order).Error
	if err == gorm.ErrRecordNotFound {
		order = models.Observation{
			CreatorID: userID,
			Status:    "черновик",
			CreatedAt: time.Now(),
		}
		if err := r.DB.Create(&order).Error; err != nil {
			return nil, err
		}
	}
	return &order, nil
}

func (r *Repository) GetCartInfo(userID int) (hasDraft bool, draftID int, cartCount int64, err error) {
	var draft models.Observation
	err = r.DB.Where("creator_id = ? AND status = 'черновик'", userID).First(&draft).Error
	if err != nil {
		return false, 0, 0, nil // черновика нет
	}

	// Считаем количество звёзд в корзине
	err = r.DB.Model(&models.ObservationStar{}).
		Where("observation_id = ?", draft.ObservationID).
		Count(&cartCount).Error
	if err != nil {
		return true, draft.ObservationID, 0, err
	}

	return true, draft.ObservationID, cartCount, nil
}

// Добавление звезды в корзину
func (r *Repository) AddStarToOrder(orderID, starID int) error {
	link := models.ObservationStar{
		ObservationID: orderID,
		StarID:        starID,
		OrderNumber:   1,
		Quantity:      1,
	}
	return r.DB.Create(&link).Error
}

func (r *Repository) UpdateObservationStarResult(observationID, starID int, result float64) error {
	return r.DB.Model(&models.ObservationStar{}).
		Where("observation_id = ? AND star_id = ?", observationID, starID).
		Update("result_value", result).Error
}

// Удалить запись м-м по observation_id + star_id
func (r *Repository) DeleteObservationStar(observationID, starID int) error {
	return r.DB.Exec("DELETE FROM observation_stars WHERE observation_id = ? AND star_id = ?", observationID, starID).Error
}

// Обновить поля записи м-м (quantity, order_number, result_value, observer_latitude, observer_longitude)
func (r *Repository) UpdateObservationStar(observationID, starID int, updates map[string]interface{}) error {
	return r.DB.Model(&models.ObservationStar{}).
		Where("observation_id = ? AND star_id = ?", observationID, starID).
		Updates(updates).Error
}
