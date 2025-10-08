package main

import (
	"Lab1/internal/app/config"
	"Lab1/internal/app/handler"
	"Lab1/internal/app/repository"
	app "Lab1/internal/pkg"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// --- Загружаем конфиг ---
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	// --- Подключаемся к БД ---
	db, err := sqlx.Connect(
		"postgres",
		"host=127.0.0.1 port=5432 user=alex password=password123 dbname=RIP sslmode=disable",
	)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// --- Создаем репозиторий ---
	repo, _ := repository.NewRepository(db)

	// --- Создаем handler ---
	h := handler.NewHandler(repo)

	// --- Создаем Gin роутер ---
	router := gin.Default()

	// --- Создаем приложение в стиле методички ---
	application := app.NewApp(cfg, router, h)

	// --- Запуск ---
	application.RunApp()
}
