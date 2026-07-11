package http

import (
	"errors"
	"net/http"
	"strconv"

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

// GetOrder handles GET /api/v1/orders/:id
func (h *Handler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	order, err := h.orderUseCase.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order: " + err.Error()})
		return
	}

	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders handles GET /api/v1/orders
func (h *Handler) ListOrders(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	orders, err := h.orderUseCase.ListOrders(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list orders: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
