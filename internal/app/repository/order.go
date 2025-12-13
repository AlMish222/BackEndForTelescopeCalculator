package repository

import (
	"Lab1/internal/app/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Получение всех заявок (observations)
func (r *Repository) GetTelescopeObservations() ([]models.TelescopeObservation, error) {
	var telescopeObservations []models.TelescopeObservation

	err := r.DB.
		Preload("Stars").
		Preload("Creator").
		Preload("Moderator").
		Where("status <> ?", "удалён").
		Order("created_at DESC").
		Find(&telescopeObservations).Error

	if err != nil {
		return nil, err
	}
	return telescopeObservations, nil
}

// Получение корзины по ID
func (r *Repository) GetTelescopeObservation(id int) (*models.TelescopeObservation, error) {
	var telescopeObservation models.TelescopeObservation

	err := r.DB.
		Preload("TelescopeObservationStars.Star").
		Preload("Creator").
		Preload("Moderator").
		Where("telescope_observation_id = ? AND status <> ?", id, "удалён").
		First(&telescopeObservation).Error

	if err != nil {
		return nil, err
	}
	return &telescopeObservation, nil
}

// Получение корзин по статусу
func (r *Repository) GetTelescopeObservationsByStatus(status string) ([]models.TelescopeObservation, error) {
	var telescopeObservations []models.TelescopeObservation
	err := r.DB.Where("status = ?", status).Preload("Stars").Find(&telescopeObservations).Error
	if err != nil {
		return nil, err
	}
	return telescopeObservations, nil
}

// Создание новой корзины
func (r *Repository) CreateTelescopeObservation(telescopeObservation *models.TelescopeObservation) error {
	return r.DB.Create(telescopeObservation).Error
}

// Обновление корзины (например, статус или даты)
func (r *Repository) UpdateTelescopeObservation(telescopeObservation *models.TelescopeObservation) error {
	return r.DB.Save(telescopeObservation).Error
}

// Логическое удаление корзины
func (r *Repository) DeleteTelescopeObservation(id int) error {
	return r.DB.Exec(`UPDATE telescope_observations SET status = 'удалён' WHERE telescope_observation_id = ?`, id).Error
}

// Получение или создание черновика
func (r *Repository) GetOrCreateDraftTelescopeObservation(userID int) (*models.TelescopeObservation, error) {
	var telescopeObservation models.TelescopeObservation

	err := r.DB.Where("creator_id = ? AND status = ?", userID, "черновик").First(&telescopeObservation).Error
	if err == gorm.ErrRecordNotFound {
		now := time.Now()
		observationDate := now.AddDate(0, 0, 5)
		telescopeObservation = models.TelescopeObservation{
			CreatorID:         userID,
			Status:            "черновик",
			CreatedAt:         now,
			ObservationDate:   &observationDate,
			ObserverLatitude:  55.858196,
			ObserverLongitude: 37.800544,
		}
		if err := r.DB.Create(&telescopeObservation).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &telescopeObservation, nil
}

func (r *Repository) GetTelescopeObservationInfo(userID int) (hasDraft bool, draftID int, cartCount int64, err error) {
	var draft models.TelescopeObservation
	err = r.DB.Where("creator_id = ? AND status = 'черновик'", userID).First(&draft).Error
	if err != nil {
		return false, 0, 0, nil
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
func (r *Repository) AddStarToTelescopeObservation(telescopeObservationID, starID int) error {
	var existing models.TelescopeObservationStar

	// Проверяем, есть ли уже такая звезда в заказе
	err := r.DB.
		Where("telescope_observation_id = ? AND star_id = ?", telescopeObservationID, starID).
		First(&existing).Error

	if err == nil {
		// Уже существует — увеличиваем количество
		existing.Quantity += 1
		return r.DB.Save(&existing).Error
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Нет такой — добавляем новую
		link := models.TelescopeObservationStar{
			TelescopeObservationID: telescopeObservationID,
			StarID:                 starID,
			OrderNumber:            1,
			Quantity:               1,
		}
		return r.DB.Create(&link).Error
	}
	return err
}

func (r *Repository) UpdateTelescopeObservationStarResult(observationID, starID int, result float64) error {
	return r.DB.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ? AND star_id = ?", observationID, starID).
		Update("result_value", result).Error
}

// ------- M - M -------

// Удалить запись м-м по observation_id + star_id
func (r *Repository) DeleteTelescopeObservationStar(observationID, starID int) error {
	return r.DB.Exec(
		"DELETE FROM telescope_observation_stars WHERE telescope_observation_id = ? AND star_id = ?",
		observationID, starID).Error
}

// Обновить поля записи м-м (quantity, order_number, result_value, observer_latitude, observer_longitude)
func (r *Repository) UpdateTelescopeObservationStar(observationID, starID int, updates map[string]interface{}) error {
	return r.DB.Model(&models.TelescopeObservationStar{}).
		Where("telescope_observation_id = ? AND star_id = ?", observationID, starID).
		Updates(updates).Error
}
