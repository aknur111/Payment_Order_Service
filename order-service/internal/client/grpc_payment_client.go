package client

import (
	"context"
	"fmt"
	"log/slog"

	paymentv1 "github.com/aknur111/my-user-service-gen/service/payment/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCPaymentClient struct {
	client paymentv1.PaymentServiceClient
	conn   *grpc.ClientConn
}

func NewGRPCPaymentClient(addr string) (*GRPCPaymentClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial payment service at %s: %w", addr, err)
	}

	slog.Info("gRPC payment client connected", "addr", addr)

	return &GRPCPaymentClient{
		client: paymentv1.NewPaymentServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *GRPCPaymentClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCPaymentClient) Authorize(
	ctx context.Context,
	orderID string,
	amount int64,
) (transactionID string, paymentStatus string, err error) {
	resp, err := c.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return "", "", fmt.Errorf("payment validation: %s", st.Message())
			case codes.Unavailable, codes.DeadlineExceeded:
				return "", "", fmt.Errorf("payment service unavailable: %s", st.Message())
			}
		}
		return "", "", fmt.Errorf("payment service error: %w", err)
	}

	return resp.GetTransactionId(), resp.GetStatus(), nil
}
