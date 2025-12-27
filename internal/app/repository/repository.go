package repository

import (
	"Lab1/internal/app/models"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &Repository{DB: db}, nil
}

func NewRepositoryFromDB(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetDraftOrder(userID int) (*models.TelescopeObservation, error) {
	var order models.TelescopeObservation
	err := r.DB.Where("creator_id = ? AND status = ?", userID, "черновик").First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}
