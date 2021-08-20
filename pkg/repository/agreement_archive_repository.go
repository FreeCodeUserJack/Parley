package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/FreeCodeUserJack/Parley/pkg/db"
	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

type AgreementArchiveRepositoryInterface interface {
	ArchiveAgreement(context.Context, domain.AgreementArchive) (*domain.AgreementArchive, rest_errors.RestError)
}

type agreementArchiveRepository struct{}

func NewAgreementArchiveRepository() AgreementArchiveRepositoryInterface {
	return &agreementArchiveRepository{}
}

func (a agreementArchiveRepository) ArchiveAgreement(ctx context.Context, agreementArchive domain.AgreementArchive) (*domain.AgreementArchive, rest_errors.RestError) {
	logger.Info("agreement archive repository ArchiveAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementArchiveCollectionName)

	res, dbErr := collection.InsertOne(ctx, agreementArchive)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("agreement archive repository ArchiveAgreement could not insert agreement into archive: %v", agreementArchive), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error trying to archive agreement", errors.New("database error"))
	}

	agreementArchive.Id = res.InsertedID.(string)

	logger.Info("agreement archive repository ArchiveAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreementArchive, nil
}
