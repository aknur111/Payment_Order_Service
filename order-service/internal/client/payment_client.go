package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PaymentClient struct {
	baseURL string
	client  *http.Client
}

type authorizeRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type authorizeResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func NewPaymentClient(baseURL string, client *http.Client) *PaymentClient {
	return &PaymentClient{baseURL: baseURL, client: client}
}

func (p *PaymentClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	payload, err := json.Marshal(authorizeRequest{OrderID: orderID, Amount: amount})
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/payments", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", "", fmt.Errorf("payment service status: %d", resp.StatusCode)
	}

	var out authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.TransactionID, out.Status, nil
}
