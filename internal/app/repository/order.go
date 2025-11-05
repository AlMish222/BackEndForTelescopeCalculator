package repository

import (
	"Lab1/internal/app/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Получение всех заявок (observations)
func (r *Repository) GetOrders() ([]models.TelescopeObservation, error) {
	var orders []models.TelescopeObservation

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
func (r *Repository) GetOrder(id int) (*models.TelescopeObservation, error) {
	var order models.TelescopeObservation

	err := r.DB.
		Preload("TelescopeObservationStars.Star").
		Preload("Creator").
		Preload("Moderator").
		Where("telescope_observation_id = ? AND status <> ?", id, "удалён").
		First(&order).Error

	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Получение корзин по статусу
func (r *Repository) GetOrdersByStatus(status string) ([]models.TelescopeObservation, error) {
	var orders []models.TelescopeObservation
	err := r.DB.Where("status = ?", status).Preload("Stars").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Создание новой корзины
func (r *Repository) CreateOrder(order *models.TelescopeObservation) error {
	return r.DB.Create(order).Error
}

// Обновление корзины (например, статус или даты)
func (r *Repository) UpdateOrder(order *models.TelescopeObservation) error {
	return r.DB.Save(order).Error
}

// Логическое удаление корзины
func (r *Repository) DeleteOrder(id int) error {
	return r.DB.Exec(`UPDATE telescope_observations SET status = 'удалён' WHERE telescope_observation_id = ?`, id).Error
}

// Получение или создание черновика
func (r *Repository) GetOrCreateDraftOrder(userID int) (*models.TelescopeObservation, error) {
	var order models.TelescopeObservation

	err := r.DB.Where("creator_id = ? AND status = ?", userID, "черновик").First(&order).Error
	if err == gorm.ErrRecordNotFound {
		now := time.Now()
		order = models.TelescopeObservation{
			CreatorID:         userID,
			Status:            "черновик",
			CreatedAt:         now,
			ObservationDate:   &now,
			ObserverLatitude:  0.0,
			ObserverLongitude: 0.0,
		}
		if err := r.DB.Create(&order).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *Repository) GetObservationInfo(userID int) (hasDraft bool, draftID int, cartCount int64, err error) {
	var draft models.TelescopeObservation
	err = r.DB.Where("creator_id = ? AND status = 'черновик'", userID).First(&draft).Error
	if err != nil {
		return false, 0, 0, nil // черновика нет
	}

	// Суммируем количество звёзд (Quantity)
	err = r.DB.Model(&models.TelescopeObservationStar{}).
		Select("COALESCE(SUM(quantity), 0)").
		Where("telescope_observation_id = ?", draft.TelescopeObservationID).
		Scan(&cartCount).Error
	if err != nil {
		return true, draft.TelescopeObservationID, 0, err
	}

	return true, draft.TelescopeObservationID, cartCount, nil
}

// Добавление звезды в корзину
func (r *Repository) AddStarToOrder(orderID, starID int) error {
	var existing models.TelescopeObservationStar

	// Проверяем, есть ли уже такая звезда в заказе
	err := r.DB.
		Where("telescope_observation_id = ? AND star_id = ?", orderID, starID).
		First(&existing).Error

	if err == nil {
		// Уже существует — увеличиваем количество
		existing.Quantity += 1
		if err := r.DB.Save(&existing).Error; err != nil {
			return err
		}
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Нет такой — добавляем новую
		link := models.TelescopeObservationStar{
			TelescopeObservationID: orderID,
			StarID:                 starID,
			OrderNumber:            1,
			Quantity:               1,
		}
		if err := r.DB.Create(&link).Error; err != nil {
			return err
		}
		return nil
	}

	return err
}

func (r *Repository) UpdateObservationStarResult(observationID, starID int, result float64) error {
	return r.DB.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ? AND star_id = ?", observationID, starID).
		Update("result_value", result).Error
}

// Удалить запись м-м по observation_id + star_id
func (r *Repository) DeleteObservationStar(observationID, starID int) error {
	return r.DB.Exec("DELETE FROM telescope_observation_stars WHERE telescope_observation_id = ? AND star_id = ?", observationID, starID).Error
}

// Обновить поля записи м-м (quantity, order_number, result_value, observer_latitude, observer_longitude)
func (r *Repository) UpdateObservationStar(observationID, starID int, updates map[string]interface{}) error {
	return r.DB.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ? AND star_id = ?", observationID, starID).
		Updates(updates).Error
}
