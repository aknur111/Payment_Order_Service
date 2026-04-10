package usecase

import (
	"context"
	"order-service/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderUsecase struct {
	repo          OrderRepository
	paymentClient PaymentAuthorizer
}

func NewOrderUsecase(repo OrderRepository, paymentClient PaymentAuthorizer) *OrderUsecase {
	return &OrderUsecase{repo: repo, paymentClient: paymentClient}
}

func (u *OrderUsecase) CreateOrder(
	ctx context.Context,
	customerID,
	itemName string,
	amount int64,
	idempotencyKey string,
) (*domain.Order, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)

	if idempotencyKey != "" {
		existing, err := u.repo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	order := &domain.Order{
		ID:             uuid.NewString(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         domain.OrderStatusPending,
		CreatedAt:      time.Now().UTC(),
		IdempotencyKey: idempotencyKey,
	}

	if err := u.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	_, paymentStatus, err := u.paymentClient.Authorize(ctx, order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(ctx, order.ID, domain.OrderStatusFailed)
		order.Status = domain.OrderStatusFailed
		return order, domain.ErrPaymentUnavailable
	}

	if paymentStatus == "Authorized" {
		order.Status = domain.OrderStatusPaid
	} else {
		order.Status = domain.OrderStatusFailed
	}

	if err := u.repo.UpdateStatus(ctx, order.ID, order.Status); err != nil {
		return nil, err
	}

	return order, nil
}

func (u *OrderUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *OrderUsecase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status == domain.OrderStatusPaid {
		return nil, domain.ErrCannotCancelPaid
	}
	if order.Status != domain.OrderStatusPending {
		return nil, domain.ErrCannotCancelStatus
	}

	if err := u.repo.UpdateStatus(ctx, id, domain.OrderStatusCancelled); err != nil {
		return nil, err
	}
	order.Status = domain.OrderStatusCancelled
	return order, nil
}
