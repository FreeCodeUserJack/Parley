package services

import (
	"context"
	"html"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

type NotificationServiceInterface interface {
	GetUserNotifications(context.Context, string) ([]domain.Notification, rest_errors.RestError)
}

type notificationService struct {
	NotificationRepository repository.NotificationRepositoryInterface
}

func NewNotificationService(notificationRepo repository.NotificationRepositoryInterface) NotificationServiceInterface {
	return &notificationService{
		NotificationRepository: notificationRepo,
	}
}

func (n notificationService) GetUserNotifications(ctx context.Context, userId string) ([]domain.Notification, rest_errors.RestError) {
	logger.Info("notification service GetUserNotifications start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize userId
	userId = html.EscapeString(userId)

	logger.Info("notification service GetUserNotifications finish", context_utils.GetTraceAndClientIds(ctx)...)
	return n.NotificationRepository.GetUserNotifications(ctx, userId)
}
