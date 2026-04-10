package usecase

import (
	"context"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo PaymentRepository
}

func NewPaymentUsecase(repo PaymentRepository) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
}

func (u *PaymentUsecase) CreatePayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	status := domain.PaymentStatusAuthorized
	if amount > 100000 {
		status = domain.PaymentStatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		TransactionID: uuid.NewString(),
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(ctx, payment); err != nil {
		return nil, err
	}
	return payment, nil
}

func (u *PaymentUsecase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(ctx, orderID)
}
