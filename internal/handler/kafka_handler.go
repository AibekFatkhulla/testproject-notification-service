package handler

import (
	"context"
	"encoding/json"
	"notification-service/internal/domain"
)

// NotificationService defines the interface for notification business logic
type NotificationService interface {
	ProcessPurchase(ctx context.Context, purchase domain.PurchaseInfo) error
	ProcessRefund(ctx context.Context, refund domain.RefundInfo) error
}

type purchaseHandler struct {
	notificationService NotificationService
}

type refundHandler struct {
	notificationService NotificationService
}

func NewPurchaseHandler(notificationService NotificationService) *purchaseHandler {
	return &purchaseHandler{notificationService: notificationService}
}

func NewRefundHandler(notificationService NotificationService) *refundHandler {
	return &refundHandler{notificationService: notificationService}
}

func (h *purchaseHandler) HandleMessage(ctx context.Context, message []byte) error {
	var purchase domain.PurchaseInfo
	if err := json.Unmarshal(message, &purchase); err != nil {
		return err
	}
	return h.notificationService.ProcessPurchase(ctx, purchase)
}

func (h *refundHandler) HandleMessage(ctx context.Context, message []byte) error {
	var refund domain.RefundInfo
	if err := json.Unmarshal(message, &refund); err != nil {
		return err
	}
	return h.notificationService.ProcessRefund(ctx, refund)
}
