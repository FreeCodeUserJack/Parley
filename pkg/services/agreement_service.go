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
	agreement.Agreement_Deadline.LastUpdateDatetime = currTime

	if agreement.Agreement_Deadline.NotifyDateTime == 0 {
		agreement.Agreement_Deadline.NotifyDateTime = time.Unix(currTime, 0).UTC().Add(time.Hour * -24).Unix()
	}

	logger.Info("agreement service NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.NewAgreement(ctx, agreement)
}

func (a agreementService) DeleteAgreement(ctx context.Context, id string) (string, rest_errors.RestError) {
	logger.Info("agreement service DeleteAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	// Check for UUID size
	// if len([]byte(id)) != 16 {
	// 	logger.Error("invalid uuid size", errors.New("bytes length of uuid is not 16: " + strconv.Itoa(len([]byte(id)))), context_utils.GetTraceAndClientIds(ctx)...)
	// 	return nil, rest_errors.NewBadRequestError("invalid uuid")
	// }

	logger.Info("agreement service DeleteAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.DeleteAgreement(ctx, id)
}
