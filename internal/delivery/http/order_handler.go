package http

import (
	"errors"
	"net/http"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/dto"
	"github.com/gin-gonic/gin"
)

type createOrderRequest struct {
	UserID int64              `json:"user_id" binding:"required"`
	Items  []orderItemRequest `json:"items" binding:"required,min=1"`
}

type orderItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required,gt=0"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	usecaseItems := make([]dto.CreateOrderItemInput, len(req.Items))
	for i, item := range req.Items {
		usecaseItems[i] = dto.CreateOrderItemInput{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	input := dto.CreateOrderInput{
		UserID: req.UserID,
		Items:  usecaseItems,
	}

	createdOrder, err := h.orderUseCase.CreateOrder(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()}) // 409 Conflict is a good choice for stock issues
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdOrder)
}
