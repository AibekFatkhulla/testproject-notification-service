package handler

import (
	"context"
	"notification-service/internal/service"
	"notification-service/internal/domain"
	"encoding/json"
)

type PurchaseHandler struct {
	notificationService service.NotificationServiceInterface
}

func NewPurchaseHandler(notificationService service.NotificationServiceInterface) *PurchaseHandler {
	return &PurchaseHandler{notificationService: notificationService}
}

func (h *PurchaseHandler) HandleMessage(ctx context.Context, message []byte) error {
	var purchase domain.PurchaseInfo
	if err := json.Unmarshal(message, &purchase); err != nil {
		return err
	}
	return h.notificationService.ProcessPurchase(ctx, purchase)
}