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
	uuid, err := uuid.NewUUID()
	if err != nil {
		logger.Error("error creating uuid", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error creating uuid", errors.New("uuid could not be created"))
	}
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
