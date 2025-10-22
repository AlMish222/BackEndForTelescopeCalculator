package main

import (
	"Lab1/internal/app/config"
	"Lab1/internal/app/handler"
	"Lab1/internal/app/repository"
	app "Lab1/internal/pkg"

	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// --- Загружаем конфиг ---
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	dsn := "host=127.0.0.1 user=alex password=password123 dbname=RIP port=5432 sslmode=disable"

	repo, err := repository.NewRepository(dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	config.InitMinio()

	h := handler.NewHandler(repo)
	h.MinioClient = config.MinioClient

	// --- Создаем Gin роутер ---
	router := gin.Default()
	application := app.NewApp(cfg, router, h)

	// --- Запуск ---
	log.Println("Сервер запущен на http://127.0.0.1:9005")
	application.RunApp()
}
