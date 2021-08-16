package repository

import (
	"context"
	"errors"

	"github.com/FreeCodeUserJack/Parley/pkg/db"
	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AgreementRepositoryInterface interface {
	NewAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	DeleteAgreement(context.Context, string) (string, rest_errors.RestError)
}

type agreementRepository struct {
}

func NewAgreementRepository() AgreementRepositoryInterface {
	return &agreementRepository{}
}

func (a agreementRepository) NewAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository NewAgreement start", context_utils.GetTraceAndClientIds(ctx)...)
	mongoDBClient, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := mongoDBClient.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, err := collection.InsertOne(context.TODO(), agreement)
	if err != nil {
		logger.Error("error when trying to create new agreement", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to create new agreement", errors.New("database error"))
	}

	// fmt.Println(len(res.InsertedID.(primitive.Binary).Data))

	// uuid, err := uuid.FromBytes(res.InsertedID.(primitive.Binary).Data)
	// if err != nil {
	// 	logger.Error("agreement repository NewAgreement uuid.FromBytes on DB uuid error", err, context_utils.GetTraceAndClientIds(ctx)...)
	// 	return nil, rest_errors.NewInternalServerError("error when turning uuid from DB to obj", errors.New("uuid error"))
	// }

	agreement.Id = res.InsertedID.(string)

	logger.Info("agreement repository NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreement, nil
}

func (a agreementRepository) DeleteAgreement(ctx context.Context, uuid string) (string, rest_errors.RestError) {
	logger.Info("agreement repository DeleteAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: uuid}}
	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.DeleteOne(context.TODO(), filter)
	if dbErr != nil {
		logger.Error("agreement repository DeleteAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to delete doc with id: "+uuid, errors.New("database error"))
	} else if res.DeletedCount == 0 {
		logger.Error("agreement repository DeleteAgreement no doc found", errors.New("no doc with id: "+uuid+" found"), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("doc with id: " + uuid + " not found")
	}

	logger.Info("agreement repository NewAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return uuid, nil
}
