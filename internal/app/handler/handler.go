package handler

import (
	"Lab1/internal/app/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) GetOrders(ctx *gin.Context) {
	var orders []repository.Order
	var err error

	searchQuery := ctx.Query("query")
	if searchQuery == "" {
		orders, err = h.Repository.GetOrders()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		orders, err = h.Repository.GetOrdersByTitle(searchQuery)
		if err != nil {
			logrus.Error(err)
		}
	}

	cart, _ := h.Repository.GetCart()

	ctx.HTML(http.StatusOK, "pageStars.html", gin.H{
		"time":      time.Now().Format("15:04:05"),
		"orders":    orders,
		"query":     searchQuery,
		"cartCount": cart.TotalQuantity(),
	})
}

func (h *Handler) GetOrder(ctx *gin.Context) {
	idStr := ctx.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	order, err := h.Repository.GetOrder(id)
	if err != nil {
		logrus.Error(err)
	}

	cart, _ := h.Repository.GetCart()

	ctx.HTML(http.StatusOK, "calculationPage.html", gin.H{
		"order":     order,
		"cartCount": cart.TotalQuantity(),
	})
}

func (h *Handler) GetCart(c *gin.Context) {
	cart, err := h.Repository.GetCart()
	if err != nil {
		c.String(500, "Ошибка получения корзины")
		return
	}

	c.HTML(http.StatusOK, "shoppingCartPageWithApplications.html", gin.H{
		"cart":      cart,
		"cartCount": cart.TotalQuantity(),
	})
}

//func (h *Handler) AddToCart(ctx *gin.Context) {
//	idStr := ctx.Param("id")
//	id, err := strconv.Atoi(idStr)
//	if err != nil {
//		ctx.String(400, "Неверный ID")
//		return
//	}
//
//	order, err := h.Repository.GetOrder(id)
//	if err != nil {
//		ctx.String(404, "Заказ не найден")
//		return
//	}
//
//	h.Repository.AddToCart(order)
//	ctx.Status(200)
//}
