package grpctransport

import (
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"order-service/internal/domain"

	orderv1 "github.com/aknur111/my-user-service-gen/service/order/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGRPCHandler struct {
	orderv1.UnimplementedOrderServiceServer
	db             *sql.DB
	pollIntervalMs int
}

func NewOrderGRPCHandler(db *sql.DB, pollIntervalMs int) *OrderGRPCHandler {
	return &OrderGRPCHandler{db: db, pollIntervalMs: pollIntervalMs}
}

func (h *OrderGRPCHandler) SubscribeToOrderUpdates(
	req *orderv1.OrderRequest,
	stream orderv1.OrderService_SubscribeToOrderUpdatesServer,
) error {
	ctx := stream.Context()
	orderID := req.GetOrderId()

	slog.Info("gRPC SubscribeToOrderUpdates started", "order_id", orderID)

	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	queryStatus := func() (domain.OrderStatus, error) {
		var s string
		err := h.db.QueryRowContext(ctx,
			`SELECT status FROM orders WHERE id = $1`, orderID,
		).Scan(&s)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", status.Errorf(codes.NotFound, "order %s not found", orderID)
			}
			return "", status.Errorf(codes.Internal, "db query error: %v", err)
		}
		return domain.OrderStatus(s), nil
	}

	currentStatus, err := queryStatus()
	if err != nil {
		return err
	}

	if err := stream.Send(&orderv1.OrderStatusUpdate{
		OrderId:   orderID,
		Status:    string(currentStatus),
		UpdatedAt: timestamppb.New(time.Now().UTC()),
	}); err != nil {
		return err
	}

	if isTerminal(currentStatus) {
		slog.Info("order already terminal, closing stream", "order_id", orderID, "status", currentStatus)
		return nil
	}

	ticker := time.NewTicker(time.Duration(h.pollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	lastKnown := currentStatus

	for {
		select {
		case <-ctx.Done():
			slog.Info("stream client disconnected", "order_id", orderID)
			return nil

		case <-ticker.C:
			newStatus, err := queryStatus()
			if err != nil {
				slog.Warn("could not query order during stream", "order_id", orderID, "error", err)
				return err
			}

			if newStatus != lastKnown {
				slog.Info("status changed — pushing to subscriber",
					"order_id", orderID,
					"old", lastKnown,
					"new", newStatus,
				)
				if err := stream.Send(&orderv1.OrderStatusUpdate{
					OrderId:   orderID,
					Status:    string(newStatus),
					UpdatedAt: timestamppb.New(time.Now().UTC()),
				}); err != nil {
					return err
				}
				lastKnown = newStatus
				if isTerminal(newStatus) {
					slog.Info("order reached terminal state, closing stream", "order_id", orderID)
					return nil
				}
			}
		}
	}
}

func isTerminal(s domain.OrderStatus) bool {
	return s == domain.OrderStatusPaid ||
		s == domain.OrderStatusFailed ||
		s == domain.OrderStatusCancelled
}
