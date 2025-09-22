package api

import (
	"Lab1/internal/app/handler"
	"Lab1/internal/app/repository"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func StartServer() {
	log.Println("Server start up")

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Error("ошибка инициализации репозитория")
	}

	handler := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/hello", handler.GetOrders)
	r.GET("/order/:id", handler.GetOrder)
	r.GET("/cart", handler.GetCart)

	r.Run() // (for windows "localhost:8080")
	log.Println("Server down")
}
