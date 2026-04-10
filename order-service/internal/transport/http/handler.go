package httptransport

import (
	"errors"
	"log/slog"
	"net/http"

	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc *usecase.OrderUsecase
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount"`
}

type errorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

func NewHandler(uc *usecase.OrderUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrder)

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


// @Summary Create order
// @Description Create a new order
// @Tags orders
// @Accept json
// @Produce json
// @Param request body createOrderRequest true "order request"
// @Success 201 {object} domain.Order
// @Failure 400 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /orders [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	requestID := c.GetString("request_id")

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, err, "INVALID_REQUEST")
		return
	}

	slog.Info("create order request",
		"request_id", requestID,
		"customer_id", req.CustomerID,
		"amount", req.Amount,
	)

	idempotencyKey := c.GetHeader("Idempotency-Key")

	order, err := h.uc.CreateOrder(
		c.Request.Context(),
		req.CustomerID,
		req.ItemName,
		req.Amount,
		idempotencyKey,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidAmount):
			h.respondError(c, http.StatusBadRequest, err, "INVALID_AMOUNT")

		case errors.Is(err, domain.ErrPaymentUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":      err.Error(),
				"code":       "PAYMENT_UNAVAILABLE",
				"order":      order,
				"request_id": requestID,
			})

		default:
			h.respondError(c, http.StatusInternalServerError, err, "INTERNAL_ERROR")
		}
		return
	}

	slog.Info("order created", "order_id", order.ID)

	c.JSON(http.StatusCreated, order)
}

// @Summary Get order
// @Tags orders
// @Param id path string true "order id"
// @Success 200 {object} domain.Order
// @Failure 404 {object} errorResponse
// @Router /orders/{id} [get]
func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.uc.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			h.respondError(c, http.StatusNotFound, err, "ORDER_NOT_FOUND")
			return
		}
		h.respondError(c, http.StatusInternalServerError, err, "INTERNAL_ERROR")
		return
	}

	c.JSON(http.StatusOK, order)
}

// @Summary Cancel order
// @Tags orders
// @Param id path string true "order id"
// @Success 200 {object} domain.Order
// @Failure 409 {object} errorResponse
// @Router /orders/{id}/cancel [patch]
func (h *Handler) CancelOrder(c *gin.Context) {
	order, err := h.uc.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderNotFound):
			h.respondError(c, http.StatusNotFound, err, "ORDER_NOT_FOUND")

		case errors.Is(err, domain.ErrCannotCancelPaid):
			h.respondError(c, http.StatusConflict, err, "CANNOT_CANCEL_PAID")

		case errors.Is(err, domain.ErrCannotCancelStatus):
			h.respondError(c, http.StatusConflict, err, "INVALID_STATUS")

		default:
			h.respondError(c, http.StatusInternalServerError, err, "INTERNAL_ERROR")
		}
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}