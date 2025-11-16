package repository

import (
	"Lab1/internal/app/models"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) CacheUser(ctx context.Context, user *models.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%d", user.UserID)
	fmt.Println("CACHING USER IN REDIS:", key)
	return r.client.Set(ctx, key, data, 10*time.Minute).Err()
}

func (r *RedisRepository) GetCachedUser(ctx context.Context, userID int) (*models.User, error) {

	key := fmt.Sprintf("user:%d", userID)
	fmt.Println(" TRYING TO GET FROM REDIS:", key)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// SetSession сохраняет session:<token> = userID (строка) с TTL
func (r *RedisRepository) SetSession(ctx context.Context, token string, userID int, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", token)
	val := strconv.Itoa(userID)
	// лог для отладки
	fmt.Println("SET REDIS SESSION:", key, "->", val)
	return r.client.Set(ctx, key, val, ttl).Err()
}

// GetSession возвращает userID по токену
func (r *RedisRepository) GetSession(ctx context.Context, token string) (int, error) {
	key := fmt.Sprintf("session:%s", token)
	// лог для отладки
	fmt.Println("GET REDIS SESSION:", key)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *RedisRepository) DeleteSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	// лог для отладки
	fmt.Println("DEL REDIS SESSION:", key)
	return r.client.Del(ctx, key).Err()
}
