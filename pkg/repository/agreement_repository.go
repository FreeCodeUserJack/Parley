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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgreementRepositoryInterface interface {
	NewAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	DeleteAgreement(context.Context, string) (string, rest_errors.RestError)
	UpdateAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	GetAgreement(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	SearchAgreements(context.Context, string, string) ([]domain.Agreement, rest_errors.RestError)
	AddUserToAgreement(context.Context, string, string) (string, rest_errors.RestError)
	RemoveUserFromAgreement(context.Context, string, string) (string, rest_errors.RestError)
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

	res, err := collection.InsertOne(ctx, agreement)
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

	res, dbErr := collection.DeleteOne(ctx, filter)
	if dbErr != nil {
		logger.Error("agreement repository DeleteAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to delete doc with id: " + uuid, errors.New("database error"))
	} else if res.DeletedCount == 0 {
		logger.Error("agreement repository DeleteAgreement no doc found", errors.New("no doc with id: " + uuid + " found"), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError("doc with id: " + uuid + " not found")
	}

	logger.Info("agreement repository DeleteAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return uuid, nil
}

func (a agreementRepository) UpdateAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository UpdateAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreement.Id}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "title", Value: agreement.Title},
		primitive.E{Key: "description", Value: agreement.Description},
		primitive.E{Key: "participants", Value: agreement.Participants},
		primitive.E{Key: "last_update_datetime", Value: agreement.LastUpdateDateTime},
		primitive.E{Key: "agreement_deadline", Value: agreement.AgreementDeadline},
		primitive.E{Key: "status", Value: agreement.Status},
		primitive.E{Key: "public", Value: agreement.Public},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.UpdateOne(ctx, filter, updater)
	if dbErr != nil {
		logger.Error("agreement repository DeleteAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to update doc with id: " + agreement.Id, errors.New("database error"))
	} else if res.MatchedCount == 0 {
		logger.Error("agreement repository DeleteAgreement no doc found", errors.New("no doc with id: " + agreement.Id + " found"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("doc with id: " + agreement.Id + " not found")
	}

	logger.Info("agreement repository UpdateAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreement, nil
}

func (a agreementRepository) GetAgreement(ctx context.Context, id string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository GetAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	var returnedAgreement domain.Agreement

	filter := bson.D{primitive.E{Key: "_id", Value: id}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	dbErr := collection.FindOne(ctx, filter).Decode(&returnedAgreement)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("No agreement found for id: %s: ", id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", id))
		}
		logger.Error("agreement repository GetAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get doc with id: " + id, errors.New("database error"))
	}

	logger.Info("agreement repository GetAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &returnedAgreement, nil
}

func (a agreementRepository) SearchAgreements(ctx context.Context, key string, val string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository SearchAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	var resultAgreements []domain.Agreement

	// nameFilter := bson.D{primitive.E{Key: "title", Value: val}}
	nameFilter := bson.M{"title": bson.M {
		"$regex": primitive.Regex{Pattern: ".*" + val + ".*", Options: "i"},
	}}
	tagsFilter := bson.M{"tags": bson.M {
		"$regex": primitive.Regex{Pattern: ".*" + val + ".*", Options: "i"},
	}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	var cur *mongo.Cursor
	var findErr error
	
	if key == "name" {
		cur, findErr = collection.Find(ctx, nameFilter)
	} else { // tags
		cur, findErr = collection.Find(ctx, tagsFilter)
	}

	keyValErrString := fmt.Sprintf("agreement repository SearchAgreements search failed for key:value - %s:%s", key, val)

	if findErr != nil {
		logger.Error(keyValErrString, findErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(keyValErrString, errors.New("database error"))
	}

	for cur.Next(ctx) {
		buf := domain.Agreement{}
		err := cur.Decode(&buf)
		if err != nil {
			logger.Error(keyValErrString, findErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(keyValErrString, errors.New("database error"))	
		}
		resultAgreements = append(resultAgreements, buf)
	}

	cur.Close(ctx)

	if len(resultAgreements) == 0 {
		logger.Error(keyValErrString, errors.New("no documents found for search"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewNotFoundError(keyValErrString)
	}

	logger.Info("agreement repository SearchAgreements finish", context_utils.GetTraceAndClientIds(ctx)...)
	return resultAgreements, nil
}

func (a agreementRepository) AddUserToAgreement(ctx context.Context, agreementId string, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement repository AddUserToAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreementId}}

	updater := bson.D{primitive.E{Key: "$push", Value: bson.D{
		primitive.E{Key:"participants", Value: friendId},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.UpdateOne(ctx, filter, updater)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("agreement repository AddUserToAgreement failed to update (agreementId:friendId): %s:%s", agreementId, friendId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("", errors.New("database error"))
	}

	if res.MatchedCount == 0 {
		logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
	}

	logger.Info("agreement repository AddUserToAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return friendId, nil
}

func (a agreementRepository) RemoveUserFromAgreement(ctx context.Context, agreementId, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement repository RemoveUserFromAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreementId}}

	updater := bson.D{primitive.E{Key: "$pull", Value: bson.D{
		primitive.E{Key:"participants", Value: friendId},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.UpdateOne(ctx, filter, updater)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("agreement repository RemoveUserFromAgreement failed to update (agreementId:friendId): %s:%s", agreementId, friendId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("", errors.New("database error"))
	}

	if res.MatchedCount == 0 {
		logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
	}

	logger.Info("agreement repository RemoveUserFromAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return friendId, nil
}