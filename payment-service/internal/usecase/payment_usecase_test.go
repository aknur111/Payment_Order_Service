package usecase

import (
	"context"
	"errors"
	"payment-service/internal/domain"
	"testing"
)

type mockRepo struct {
	payment *domain.Payment
}

func (m *mockRepo) Create(ctx context.Context, p *domain.Payment) error {
	m.payment = p
	return nil
}

func (m *mockRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	if m.payment == nil {
		return nil, domain.ErrPaymentNotFound
	}
	return m.payment, nil
}

func (m *mockRepo) EnsureSchema(ctx context.Context) error {
	return nil
}

func TestCreatePayment_Authorized(t *testing.T) {
	repo := &mockRepo{}
	uc := NewPaymentUsecase(repo)

	p, err := uc.CreatePayment(context.Background(), "order-1", 50000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Status != domain.PaymentStatusAuthorized {
		t.Fatalf("expected Authorized, got %s", p.Status)
	}
}

func TestCreatePayment_Declined(t *testing.T) {
	repo := &mockRepo{}
	uc := NewPaymentUsecase(repo)

	p, err := uc.CreatePayment(context.Background(), "order-1", 200000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Status != domain.PaymentStatusDeclined {
		t.Fatalf("expected Declined, got %s", p.Status)
	}
}

func TestCreatePayment_InvalidAmount(t *testing.T) {
	repo := &mockRepo{}
	uc := NewPaymentUsecase(repo)

	_, err := uc.CreatePayment(context.Background(), "order-1", 0)
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("expected invalid amount error")
	}
}

func TestGetPayment_Success(t *testing.T) {
	repo := &mockRepo{
		payment: &domain.Payment{
			ID:            "1",
			OrderID:       "order-1",
			TransactionID: "tx-1",
			Amount:        50000,
			Status:        domain.PaymentStatusAuthorized,
		},
	}

	uc := NewPaymentUsecase(repo)

	p, err := uc.GetByOrderID(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.OrderID != "order-1" {
		t.Fatalf("wrong payment returned")
	}
}

func TestGetPayment_NotFound(t *testing.T) {
	repo := &mockRepo{}
	uc := NewPaymentUsecase(repo)

	_, err := uc.GetByOrderID(context.Background(), "order-1")
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected not found error")
	}
}