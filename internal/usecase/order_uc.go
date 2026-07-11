package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/dto"
)

type OrderUseCase struct {
	orderRepo   OrderRepository
	productRepo ProductRepository
	txManager   TransactionManager
	outboxRepo  OutboxRepository
}

func NewOrderUseCase(or OrderRepository, pr ProductRepository, tm TransactionManager, outboxRepo OutboxRepository) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:   or,
		productRepo: pr,
		txManager:   tm,
		outboxRepo:  outboxRepo,
	}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, input dto.CreateOrderInput) (*domain.Order, error) {
	if len(input.Items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}

	var createdOrder *domain.Order

	// --- Transactional Business Logic ---
	// using the callback pattern provided by our TransactionManager.
	err := uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Get all product IDs from the input to fetch them in one query.
		productIDs := make([]int64, len(input.Items))
		itemMap := make(map[int64]dto.CreateOrderItemInput)
		for i, item := range input.Items {
			if item.Quantity <= 0 {
				return errors.New("item quantity must be positive")
			}
			productIDs[i] = item.ProductID
			itemMap[item.ProductID] = item
		}

		// Fetch all required products from the database and lock their rows.
		products, err := uc.productRepo.FindManyByIDsForUpdate(txCtx, productIDs)
		if err != nil {
			return err
		}
		if len(products) != len(productIDs) {
			return errors.New("one or more products not found")
		}

		var orderItems []domain.OrderItem
		var productsToUpdate []*domain.Product

		// Validate stock and prepare domain objects.
		for _, p := range products {
			itemInput := itemMap[p.ID]

			if !p.IsStockAvailable(itemInput.Quantity) {
				return domain.ErrInsufficientStock
			}

			if err := p.DecreaseStock(itemInput.Quantity); err != nil {
				return err
			}

			orderItems = append(orderItems, domain.OrderItem{
				Product:      p,
				Quantity:     itemInput.Quantity,
				PriceAtOrder: p.Price,
			})

			productToUpdate := p
			productsToUpdate = append(productsToUpdate, &productToUpdate)
		}

		// Create the main Order domain object.
		createdOrder, err = domain.NewOrder(input.UserID, orderItems)
		if err != nil {
			return err
		}

		// Persist the order and its items to the database.
		if err := uc.orderRepo.Save(txCtx, createdOrder); err != nil {
			return err
		}

		// Persist the updated product stock for all affected products.
		for _, p := range productsToUpdate {
			if err := uc.productRepo.Update(txCtx, p); err != nil {
				return err
			}
		}

		// Create the outbox event payload inside the same SQL transaction.
		eventPayload, err := json.Marshal(map[string]interface{}{
			"order_id": createdOrder.ID,
			"user_id":  createdOrder.UserID,
		})
		if err != nil {
			return err
		}

		outboxEvent := &domain.OutboxEvent{
			EventType: "orders.created",
			Payload:   eventPayload,
			Status:    domain.OutboxStatusPending,
		}

		if err := uc.outboxRepo.Save(txCtx, outboxEvent); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdOrder, nil
}

