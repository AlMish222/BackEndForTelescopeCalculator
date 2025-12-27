// @title Calculator Observations Stars API
// @version 1.0
// @description Система позволяет пользователям создавать заявки на получение данных для расчёта наведения телескопа,
// @description а модераторам — управлять этими заявками и добавлять новые звёзды.
//
// @description ## 🔐 Аутентификация
// @description - Используются **сессии и cookie**, хранящиеся в **Redis**.
// @description - Без авторизации доступны только методы **чтения (GET)**.
// @description - Для авторизованных запросов браузер автоматически отправляет cookie.
//
// @description ## 👥 Роли и права доступа
// @description - **Гость:** только GET-запросы.
// @description - **Пользователь:** управление своими заявками + чтение данных.
// @description - **Модератор:** полный доступ ко всем ресурсам.
//
// @host 127.0.0.1:9005
// @BasePath /api
//// @schemes http
//
// @securityDefinitions.apikey SessionAuth
// @type apiKey
// @in cookie
// @name session_token

package main

import (
	"Lab1/internal/app/config"
	"Lab1/internal/app/handler"
	"Lab1/internal/app/redisdb"
	"Lab1/internal/app/repository"
	app "Lab1/internal/pkg"

	"log"

	_ "Lab1/cmd/docs"
	"Lab1/internal/app/middleware"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func main() {
	// --- Загружаем конфиг ---
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	rdb := redisdb.NewRedisClient("localhost:6379", "password")
	defer rdb.Close()

	dsn := "host=127.0.0.1 user=alex password=password123 dbname=RIP port=5432 sslmode=disable"
	redisRepo := repository.NewRedisRepository(rdb)

	repo, err := repository.NewRepository(dsn, redisRepo)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	config.InitMinio()

	h := handler.NewHandler(repo)
	h.MinioClient = config.MinioClient
	h.RedisClient = rdb

	// --- Создаем Gin роутер ---
	router := gin.Default()

	router.Use(middleware.CORSMiddleware())

	//router.Use(cors.New(cors.Config{
	//	AllowOrigins: []string{
	//		"http://localhost:3000",
	//		"http://127.0.0.1:3000",
	//		"http://192.168.1.51:3000",
	//
	//		// Tauri
	//		"tauri://localhost",
	//		"http://tauri.localhost",
	//
	//		// GitHub Pages (на всякий случай)
	//		"https://almish222.github.io",
	//	},
	//	AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	//	AllowHeaders: []string{
	//		"Origin",
	//		"Content-Type",
	//		"Accept",
	//		"Authorization",
	//	},
	//	ExposeHeaders:    []string{"Content-Length"},
	//	AllowCredentials: true,
	//}))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	application := app.NewApp(cfg, router, h)

	// --- Запуск ---
	log.Println("Сервер запущен на http://192.168.1.51:9005")
	application.RunApp()
}
