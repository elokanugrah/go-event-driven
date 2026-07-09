package http

import (
	"github.com/elokanugrah/go-event-driven/internal/usecase"
)

type Handler struct {
	productUseCase *usecase.ProductUseCase
	orderUseCase   *usecase.OrderUseCase
}

func NewHandler(puc *usecase.ProductUseCase, ouc *usecase.OrderUseCase) *Handler {
	return &Handler{
		productUseCase: puc,
		orderUseCase:   ouc,
	}
}
