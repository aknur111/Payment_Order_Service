package httptransport

import (
	"errors"
	"log/slog"
	"net/http"

	"payment-service/internal/domain"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc *usecase.PaymentUsecase
}

type createPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"`
}

type errorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

func NewHandler(uc *usecase.PaymentUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/payments", h.CreatePayment)
	r.GET("/payments/:order_id", h.GetPaymentByOrderID)
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
}

func (h *Handler) respondError(c *gin.Context, status int, err error, code string) {
	requestID := c.GetString("request_id")

	slog.Error("request failed",
		"request_id", requestID,
		"code", code,
		"error", err,
	)

	c.JSON(status, errorResponse{
		Error:     err.Error(),
		Code:      code,
		RequestID: requestID,
	})
}

// @Summary Create payment
// @Description Authorize or decline payment for an order
// @Tags payments
// @Accept json
// @Produce json
// @Param request body createPaymentRequest true "payment request"
// @Success 201 {object} domain.Payment
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /payments [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	requestID := c.GetString("request_id")

	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, err, "INVALID_REQUEST")
		return
	}

	slog.Info("create payment request",
		"request_id", requestID,
		"order_id", req.OrderID,
		"amount", req.Amount,
	)

	payment, err := h.uc.CreatePayment(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAmount) {
			h.respondError(c, http.StatusBadRequest, err, "INVALID_AMOUNT")
			return
		}
		h.respondError(c, http.StatusInternalServerError, err, "INTERNAL_ERROR")
		return
	}

	slog.Info("payment created",
		"request_id", requestID,
		"payment_id", payment.ID,
		"status", payment.Status,
	)

	c.JSON(http.StatusCreated, payment)
}

// @Summary Get payment by order ID
// @Description Get payment status for a specific order
// @Tags payments
// @Produce json
// @Param order_id path string true "order id"
// @Success 200 {object} domain.Payment
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /payments/{order_id} [get]
func (h *Handler) GetPaymentByOrderID(c *gin.Context) {
	requestID := c.GetString("request_id")

	payment, err := h.uc.GetByOrderID(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			h.respondError(c, http.StatusNotFound, err, "PAYMENT_NOT_FOUND")
			return
		}
		h.respondError(c, http.StatusInternalServerError, err, "INTERNAL_ERROR")
		return
	}

	slog.Info("get payment",
		"request_id", requestID,
		"order_id", payment.OrderID,
	)

	c.JSON(http.StatusOK, payment)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *Handler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
