package app

import (
	"fmt"

	"Lab1/internal/app/config"
	"Lab1/internal/app/handler"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Application struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.Handler
}

func NewApp(cfg *config.Config, r *gin.Engine, h *handler.Handler) *Application {
	return &Application{
		Config:  cfg,
		Router:  r,
		Handler: h,
	}
}

func (a *Application) RunApp() {
	log.Info("Server start up")

	// Регистрируем маршруты и статику
	a.Handler.RegisterHandler(a.Router)
	a.Handler.RegisterStatic(a.Router)

	// Формируем адрес из конфига
	serverAddress := fmt.Sprintf("%s:%d", a.Config.ServiceHost, a.Config.ServicePort)
	if err := a.Router.Run(serverAddress); err != nil {
		log.Fatal(err)
	}
	log.Info("Server down")
}
