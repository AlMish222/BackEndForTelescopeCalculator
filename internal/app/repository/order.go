package repository

import (
	"Lab1/internal/app/models"
)

// ============================
// Получение всех заявок (observations)
// ============================
func (r *Repository) GetOrders() ([]models.Observation, error) {
	var orders []models.Observation
	err := r.DB.Preload("Stars").Preload("Creator").Preload("Moderator").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// ============================
// Получение заявки по ID
// ============================
func (r *Repository) GetOrder(id int) (*models.Observation, error) {
	var order models.Observation
	err := r.DB.Preload("Stars").Preload("Creator").Preload("Moderator").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// ============================
// Получение заявок по статусу
// ============================
func (r *Repository) GetOrdersByStatus(status string) ([]models.Observation, error) {
	var orders []models.Observation
	err := r.DB.Where("status = ?", status).Preload("Stars").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// ============================
// Создание новой заявки
// ============================
func (r *Repository) CreateOrder(order *models.Observation) error {
	return r.DB.Create(order).Error
}

// ============================
// Обновление заявки (например, статус или даты)
// ============================
func (r *Repository) UpdateOrder(order *models.Observation) error {
	return r.DB.Save(order).Error
}

// ============================
// Удаление заявки
// ============================
func (r *Repository) DeleteOrder(id int) error {
	return r.DB.Delete(&models.Observation{}, id).Error
}
