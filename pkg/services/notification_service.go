package services

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/dto"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

type NotificationServiceInterface interface {
	GetUserNotifications(context.Context, string, string, string) ([]domain.Notification, rest_errors.RestError)
	MarkNotificationRead(context.Context, string) (string, rest_errors.RestError)
	MarkAllNotificationRead(context.Context, dto.UuidsRequest) (string, rest_errors.RestError)
}

type notificationService struct {
	NotificationRepository repository.NotificationRepositoryInterface
}

func NewNotificationService(notificationRepo repository.NotificationRepositoryInterface) NotificationServiceInterface {
	return &notificationService{
		NotificationRepository: notificationRepo,
	}
}

func (n notificationService) GetUserNotifications(ctx context.Context, userId, queryKey, queryVal string) ([]domain.Notification, rest_errors.RestError) {
	logger.Info("notification service GetUserNotifications start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize userId
	userId = strings.TrimSpace(html.EscapeString(userId))
	queryKey = strings.TrimSpace(html.EscapeString(queryKey))
	queryVal = strings.TrimSpace(html.EscapeString(queryVal))

	if userId == "" || queryKey != "status" || queryVal != "old" && queryVal != "new" && queryVal != "all" {
		logger.Error(fmt.Sprintf("notiication service CloseAgreement - id, searchKey, searchVal improper: %s %s %s", userId, queryKey, queryVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("improper key/val: " + queryKey + "/" + queryVal)
	}

	logger.Info("notification service GetUserNotifications finish", context_utils.GetTraceAndClientIds(ctx)...)
	return n.NotificationRepository.GetUserNotifications(ctx, userId, queryVal)
}

func (n notificationService) MarkNotificationRead(ctx context.Context, id string) (string, rest_errors.RestError) {
	logger.Info("notification service MarkNotificationRead start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize id
	id = strings.TrimSpace(html.EscapeString(id))

	logger.Info("notification service MarkNotificationRead finish", context_utils.GetTraceAndClientIds(ctx)...)
	return n.NotificationRepository.MarkNotificationRead(ctx, id)
}

func (n notificationService) MarkAllNotificationRead(ctx context.Context, uuids dto.UuidsRequest) (string, rest_errors.RestError) {
	logger.Info("notification service MarkAllNotificationRead start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize uuids
	uuids.Payload = domain.SanitizeStringSlice(uuids.Payload)

	logger.Info("notification service MarkAllNotificationRead finish", context_utils.GetTraceAndClientIds(ctx)...)
	return n.NotificationRepository.MarkAllNotificationRead(ctx, uuids)
}
