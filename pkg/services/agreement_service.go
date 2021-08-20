package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/google/uuid"
)

type AgreementServiceInterface interface {
	NewAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	DeleteAgreement(context.Context, string) (string, rest_errors.RestError)
	UpdateAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	GetAgreement(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	SearchAgreements(context.Context, string, string) ([]domain.Agreement, rest_errors.RestError)
	AddUserToAgreement(context.Context, string, string) (string, rest_errors.RestError)
	RemoveUserFromAgreement(context.Context, string, string) (string, rest_errors.RestError)
	SetDeadline(context.Context, string, domain.Deadline) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadline(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	ActionAndNotification(context.Context, string, domain.Notification) (*domain.Notification, rest_errors.RestError)
}

type agreementService struct {
	AgreementRepository        repository.AgreementRepositoryInterface
	AgreementArchiveRepository repository.AgreementArchiveRepositoryInterface
	NotificationRepository     repository.NotificationRepositoryInterface
}

func NewAgreementService(agreementRepo repository.AgreementRepositoryInterface, agreementArchiveRepo repository.AgreementArchiveRepositoryInterface, notificationRepo repository.NotificationRepositoryInterface) AgreementServiceInterface {
	return &agreementService{
		AgreementRepository:        agreementRepo,
		AgreementArchiveRepository: agreementArchiveRepo,
		NotificationRepository:     notificationRepo,
	}
}

func (a agreementService) NewAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service NewAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	if !agreement.Validate() {
		logger.Error("agreement failed validation", errors.New("request agreement is not valid"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("req failed validation")
	}

	// Sanitize Data

	// Add UUID
	uuid := uuid.NewString()
	agreement.Id = uuid

	// Add CreateTime/UpdateTime
	currTime := time.Now().UTC()
	agreement.CreateDateTime = currTime
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	if agreement.AgreementDeadline.NotifyDateTime.IsZero() {
		agreement.AgreementDeadline.NotifyDateTime = agreement.AgreementDeadline.DeadlineDateTime.Add(time.Hour * -24).UTC()
	}

	// Initialize 4 slices
	agreement.InvitedParticipants = []string{}
	agreement.RequestedParticipants = []string{}
	agreement.PendingRemovalParticipants = []string{}
	agreement.PendingLeaveParticipants = []string{}

	logger.Info("agreement service NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.NewAgreement(ctx, agreement)
}

func (a agreementService) DeleteAgreement(ctx context.Context, id string) (string, rest_errors.RestError) {
	logger.Info("agreement service DeleteAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	//Sanitize the id string

	// Archive Agreement
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, id, "deleted", "agreement was deleted", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service DeleteAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.DeleteAgreement(ctx, id)
}

func (a agreementService) UpdateAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service UpdateAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize fields

	// Get Existing Agreement and update fields that are different
	currTime := time.Now().UTC()
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	savedAgreement, getErr := a.GetAgreement(ctx, agreement.Id)
	if getErr != nil {
		return nil, getErr
	}

	if agreement.Title == "" {
		agreement.Title = savedAgreement.Title
	}
	if agreement.Description == "" {
		agreement.Description = savedAgreement.Description
	}
	if len(agreement.Participants) == 0 {
		agreement.Participants = savedAgreement.Participants
	}
	if agreement.AgreementDeadline.DeadlineDateTime.IsZero() {
		agreement.AgreementDeadline = savedAgreement.AgreementDeadline
	}
	if agreement.Status == "" {
		agreement.Status = savedAgreement.Status
	}
	if agreement.Public == "" {
		agreement.Public = savedAgreement.Public
	}
	if len(agreement.Tags) == 0 {
		agreement.Tags = savedAgreement.Tags
	}

	agreement.CreatedBy = savedAgreement.CreatedBy

	// Archive Agreement Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreement.Id, "modified", "agreement was modified", savedAgreement)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service UpdateAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.UpdateAgreement(ctx, agreement)
}

func (a agreementService) GetAgreement(ctx context.Context, id string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service GetAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize the id string

	logger.Info("agreement service GetAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.GetAgreement(ctx, id)
}

func (a agreementService) SearchAgreements(ctx context.Context, key string, val string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service SearchAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	// Cannot search for all agreements
	if key == "" || val == "" {
		return nil, rest_errors.NewBadRequestError("key/val cannot be empty")
	}

	// Sanitize key + val

	logger.Info("agreement service SearchAgreements finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.SearchAgreements(ctx, key, val)
}

func (a agreementService) AddUserToAgreement(ctx context.Context, agreementId string, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement service AddUserToAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and friendId

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service AddUserToAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.AddUserToAgreement(ctx, agreementId, friendId)
}

func (a agreementService) RemoveUserFromAgreement(ctx context.Context, agreementId, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement service RemoveUserFromAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and friendId

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service RemoveUserFromAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.RemoveUserFromAgreement(ctx, agreementId, friendId)
}

func (a agreementService) SetDeadline(ctx context.Context, agreementId string, deadline domain.Deadline) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service SetDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and deadline instance

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	// Check Nullable fields
	if deadline.NotifyDateTime.IsZero() {
		deadline.NotifyDateTime = deadline.DeadlineDateTime.Add(time.Hour * -24).UTC()
	}

	deadline.LastUpdateDatetime = time.Now().UTC()

	// Status must be passed in request
	if deadline.Status == "" {
		return nil, rest_errors.NewBadRequestError("missing status field of deadline instance")
	}

	logger.Info("agreement service SetDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.SetDeadline(ctx, agreementId, deadline)
}

func (a agreementService) DeleteDeadline(ctx context.Context, agreementId string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service DeleteDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service DeleteDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.DeleteDeadline(ctx, agreementId)
}

func archiveAgreementHelper(ctx context.Context, agreementRepo repository.AgreementRepositoryInterface, agreementArchiveRepo repository.AgreementArchiveRepositoryInterface, id, status, info string, agreement *domain.Agreement) (*domain.AgreementArchive, rest_errors.RestError) {
	if agreement == nil {
		var err rest_errors.RestError
		agreement, err = agreementRepo.GetAgreement(ctx, id)
		if err != nil {
			logger.Error("agreement service DeleteAgreement could not get current agreement", err, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, err
		}
	}

	agreement.Status = status
	currTime := time.Now().UTC()
	agreement.LastUpdateDateTime = currTime
	agreementArchive := domain.AgreementArchive{
		Id:             uuid.NewString(),
		AgreementData:  *agreement,
		CreateDateTime: time.Now().UTC(),
		Info:           info,
	}

	return &agreementArchive, nil
}

func (a agreementService) ActionAndNotification(ctx context.Context, action string, notification domain.Notification) (*domain.Notification, rest_errors.RestError) {
	logger.Info("agreement service ActionAndNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize action string and notification instance

	// Set uuid
	notification.Id = uuid.NewString()

	// Get appropriate inputs for repository
	actionInputs := getActionAndNotificationInputs(action)
	if actionInputs != nil {
		return nil, rest_errors.NewBadRequestError("action not supported")
	}

	doneChan := make(chan int)
	go func(c chan int) {
		defer close(c)
		a.AgreementRepository.ActionAndNotification(ctx, actionInputs, notification)
		doneChan <- 1
	}(doneChan)

	notificationResult, repoErr := a.NotificationRepository.SaveNotification(ctx, notification)

	select {
	case done := <-doneChan:
		logger.Info(fmt.Sprintf("notification created: %v, chan output: %d", notification, done), context_utils.GetTraceAndClientIds(ctx)...)
	case <-time.After(5 * time.Second):
		logger.Error("agreement service ActionAndNotification - couldn't save notification", fmt.Errorf("could not update agreement/user for notification: %v", notification), context_utils.GetTraceAndClientIds(ctx)...)
		a.NotificationRepository.DeleteNotification(ctx, notification.Id) // if this fails then complete failure... - maybe put in Fatal Inconsistency DB
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("could not ActionAndNotification for %v", notification), errors.New("database error"))
	}

	logger.Info("agreement service ActionAndNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return notificationResult, repoErr
}

func getActionAndNotificationInputs(action string) []string {
	// Up to 2 Actions (for agreements), 2 inputs per action () - the data is passed via obj
	res, ok := actionCodes[action]
	if !ok {
		goto INVALIDACTION
	}

	return res

INVALIDACTION:
	return nil
}

var actionCodes map[string][]string

func init() {
	actionCodes = map[string][]string{}
}
