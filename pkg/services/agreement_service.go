package services

import (
	"context"
	"errors"
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
}

type agreementService struct {
	AgreementRepository repository.AgreementRepositoryInterface
}

func NewAgreementService(agreementRepo repository.AgreementRepositoryInterface) AgreementServiceInterface {
	return &agreementService{
		AgreementRepository: agreementRepo,
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
	currTime := time.Now().UTC().Unix()
	agreement.CreateDateTime = currTime
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	if agreement.AgreementDeadline.NotifyDateTime == 0 {
		agreement.AgreementDeadline.NotifyDateTime = time.Unix(currTime, 0).Add(time.Hour * -24).UTC().Unix()
	}

	logger.Info("agreement service NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.NewAgreement(ctx, agreement)
}

func (a agreementService) DeleteAgreement(ctx context.Context, id string) (string, rest_errors.RestError) {
	logger.Info("agreement service DeleteAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	//Sanitize the id string

	logger.Info("agreement service DeleteAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.DeleteAgreement(ctx, id)
}

func (a agreementService) UpdateAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service UpdateAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize fields

	// Get Existing Agreement and update fields that are different
	currTime := time.Now().UTC().Unix()
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	savedAgreement, getErr := a.GetAgreement(ctx, agreement.Id)
	if getErr != nil {
		return nil, rest_errors.NewInternalServerError("could not get doc for update with id: " + agreement.Id, errors.New("database error"))
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
	if agreement.AgreementDeadline.DeadlineDateTime == 0 {
		agreement.AgreementDeadline = savedAgreement.AgreementDeadline
	}
	if agreement.Status == "" {
		agreement.Status = savedAgreement.Status
	}
	if agreement.Public == "" {
		agreement.Public =savedAgreement.Public
	}
	if len(agreement.Tags) == 0 {
		agreement.Tags = savedAgreement.Tags
	}

	agreement.CreatedBy = savedAgreement.CreatedBy
	agreement.ArchiveId = savedAgreement.ArchiveId

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

	logger.Info("agreement service AddUserToAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.AddUserToAgreement(ctx, agreementId, friendId)
}

func (a agreementService) RemoveUserFromAgreement(ctx context.Context, agreementId, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement service RemoveUserFromAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and friendId

	logger.Info("agreement service RemoveUserFromAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.RemoveUserFromAgreement(ctx, agreementId, friendId)
}

func (a agreementService) SetDeadline(ctx context.Context, agreementId string, deadline domain.Deadline) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service AddDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and deadline instance

	// Check Nullable fields
	if deadline.NotifyDateTime == 0 {
		deadline.NotifyDateTime = time.Unix(deadline.DeadlineDateTime, 0).Add(time.Hour * -24).UTC().Unix()
	}

	deadline.LastUpdateDatetime = time.Now().UTC().Unix()

	// Status must be passed in request
	if deadline.Status == "" {
		return nil, rest_errors.NewBadRequestError("missing status field of deadline instance")
	}

	logger.Info("agreement service AddDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.SetDeadline(ctx, agreementId, deadline)
}