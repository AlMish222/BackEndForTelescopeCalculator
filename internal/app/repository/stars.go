package repository

import "Lab1/internal/app/models"

func (r *Repository) GetStars() ([]models.Star, error) {
	var stars []models.Star
	err := r.DB.Find(&stars).Error

	if err != nil {
		return nil, err
	}

	return stars, nil
}

func (r *Repository) GetStarByID(id int) (*models.Star, error) {
	var star models.Star
	if err := r.DB.First(&star, id).Error; err != nil {
		return nil, err
	}
	return &star, nil
}
