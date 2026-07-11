package http

import "github.com/gin-gonic/gin"

func SetupRouter(h *Handler) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api/v1")
	{
		products := api.Group("/products")
		{
			products.POST("/", h.CreateProduct)
			products.GET("/", h.ListProducts)
			products.GET("/:id", h.GetProductByID)
			products.PUT("/:id", h.UpdateProduct)
			products.DELETE("/:id", h.DeleteProduct)
		}

		orders := api.Group("/orders")
		{
			orders.POST("/", h.CreateOrder)
			orders.GET("/", h.ListOrders)
			orders.GET("/:id", h.GetOrder)
		}
	}

	return router
}
