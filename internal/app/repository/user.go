package repository

import (
	"Lab1/internal/app/models"
	"context"

	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) CreateUser(user *models.User) error {
	return r.DB.Create(user).Error
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.DB.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *Repository) GetUserByID(id int) (*models.User, error) {
	ctx := context.Background()

	// 1. Пробуем получить из Redis
	if r.Redis != nil {
		user, err := r.Redis.GetCachedUser(ctx, id)
		if err == nil {
			return user, nil
		}
	}

	// 2. Если в Redis нет — достаём из БД
	var user models.User
	err := r.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}

	// 3. Кладём в Redis
	if r.Redis != nil {
		_ = r.Redis.CacheUser(ctx, &user)
	}

	return &user, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
