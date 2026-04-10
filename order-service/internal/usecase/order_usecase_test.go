package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
	"testing"
)

type mockRepo struct {
	order *domain.Order
}

func (m *mockRepo) Create(ctx context.Context, o *domain.Order) error {
	m.order = o
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	if m.order == nil {
		return nil, domain.ErrOrderNotFound
	}
	return m.order, nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	if m.order != nil {
		m.order.Status = status
	}
	return nil
}

func (m *mockRepo) EnsureSchema(ctx context.Context) error {
	return nil
}

func (m *mockRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	return nil, nil
}

type mockPayment struct {
	status string
	err    error
}

func (m *mockPayment) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	return "tx-id", m.status, nil
}

func TestCreateOrder_Success(t *testing.T) {
	repo := &mockRepo{}
	payment := &mockPayment{status: "Authorized"}

	uc := NewOrderUsecase(repo, payment)

	order, err := uc.CreateOrder(context.Background(), "1", "Book", 1000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Status != domain.OrderStatusPaid {
		t.Fatalf("expected Paid, got %s", order.Status)
	}
}

func TestCreateOrder_InvalidAmount(t *testing.T) {
	repo := &mockRepo{}
	payment := &mockPayment{}

	uc := NewOrderUsecase(repo, payment)

	_, err := uc.CreateOrder(context.Background(), "1", "Book", 0, "")
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("expected invalid amount error")
	}
}

func TestCreateOrder_PaymentFailed(t *testing.T) {
	repo := &mockRepo{}
	payment := &mockPayment{status: "Declined"}

	uc := NewOrderUsecase(repo, payment)

	order, err := uc.CreateOrder(context.Background(), "1", "Book", 200000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Status != domain.OrderStatusFailed {
		t.Fatalf("expected Failed, got %s", order.Status)
	}
}

func TestCancelOrder_Paid(t *testing.T) {
	repo := &mockRepo{
		order: &domain.Order{
			ID:     "1",
			Status: domain.OrderStatusPaid,
		},
	}

	payment := &mockPayment{}
	uc := NewOrderUsecase(repo, payment)

	_, err := uc.CancelOrder(context.Background(), "1")
	if !errors.Is(err, domain.ErrCannotCancelPaid) {
		t.Fatalf("expected cannot cancel paid error")
	}
}
