package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/db"
	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type AgreementRepositoryInterface interface {
	NewAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	CloseAgreement(context.Context, string, string) (string, rest_errors.RestError)
	CloseAgreementDirected(context.Context, string, string, []domain.Notification) (string, rest_errors.RestError)
	UpdateAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	UpdateAgreementDirected(context.Context, domain.Agreement, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	GetAgreement(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	SearchAgreements(context.Context, string, string) ([]domain.Agreement, rest_errors.RestError)
	AddUserToAgreement(context.Context, string, string) (string, rest_errors.RestError)
	RemoveUserFromAgreement(context.Context, string, string) (string, rest_errors.RestError)
	SetDeadline(context.Context, string, domain.Deadline) (*domain.Agreement, rest_errors.RestError)
	SetDeadlineDirected(context.Context, string, domain.Deadline, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadline(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadlineDirected(context.Context, string, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	ActionAndNotification(context.Context, []string, domain.Notification) (*domain.Notification, rest_errors.RestError)
}

type agreementRepository struct {
}

func NewAgreementRepository() AgreementRepositoryInterface {
	return &agreementRepository{}
}

func (a agreementRepository) NewAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository NewAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	_, dbErr := collection.InsertOne(ctx, agreement)
	if dbErr != nil {
		logger.Error("agreement repository NewAgreement - error when trying to create new agreement", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to create new agreement", errors.New("database error"))
	}

	logger.Info("agreement repository NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreement, nil
}

func (a agreementRepository) CloseAgreement(ctx context.Context, uuid string, completionReason string) (string, rest_errors.RestError) {
	logger.Info("agreement repository CloseAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: uuid}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "status", Value: completionReason},
		primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.UpdateOne(ctx, filter, updater)
	if dbErr != nil {
		logger.Error("agreement repository CloseAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to close doc with id: "+uuid, errors.New("database error"))
	} else if res.MatchedCount == 0 {
		logger.Error("agreement repository CloseAgreement no doc found", errors.New("no doc with id: "+uuid+" found"), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError("doc with id: " + uuid + " not found")
	}

	logger.Info("agreement repository CloseAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return uuid, nil
}

func (a agreementRepository) CloseAgreementDirected(ctx context.Context, uuid string, completionReason string, notifications []domain.Notification) (string, rest_errors.RestError) {
	logger.Info("agreement repository CloseAgreementDirected start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
		filter := bson.D{primitive.E{Key: "_id", Value: uuid}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "status", Value: completionReason},
			primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
		}}}

		res1, err1 := agreementColl.UpdateOne(sessCtx, filter, updater)
		if err1 != nil {
			logger.Error("agreement repository CloseAgreementDirected db error", err1, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to close doc with id: "+uuid, errors.New("database error"))
		}

		if res1.MatchedCount == 0 {
			logger.Error("agreement repository CloseAgreementDirected no doc found", errors.New("no doc with id: "+uuid+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("doc with id: " + uuid + " not found")
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository CloseAgreementDirected transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository CloseAgreementDirected - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository CloseAgreementDirected - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository CloseAgreementDirected finish", context_utils.GetTraceAndClientIds(ctx)...)
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

	_, dbErr := collection.UpdateOne(ctx, filter, updater)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error("agreement repository DeleteAgreement no doc found", errors.New("no doc with id: "+agreement.Id+" found"), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewBadRequestError("doc with id: " + agreement.Id + " not found")
		}
		logger.Error("agreement repository DeleteAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to update doc with id: "+agreement.Id, errors.New("database error"))
	}

	logger.Info("agreement repository UpdateAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreement, nil
}

func (a agreementRepository) UpdateAgreementDirected(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository UpdateAgreementDirected start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
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

		res1, err1 := agreementColl.UpdateOne(sessCtx, filter, updater)
		if err1 != nil {
			logger.Error("agreement repository UpdateAgreementDirected db error", err1, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to close doc with id: "+agreement.Id, errors.New("database error"))
		}

		if res1.MatchedCount == 0 {
			logger.Error("agreement repository UpdateAgreementDirected no doc found", errors.New("no doc with id: "+agreement.Id+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("doc with id: " + agreement.Id + " not found")
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository UpdateAgreementDirected transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository UpdateAgreementDirected - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository UpdateAgreementDirected - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository UpdateAgreementDirected finish", context_utils.GetTraceAndClientIds(ctx)...)
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
			logger.Error(fmt.Sprintf("agreement repository GetAgreement - No agreement found for id: %s: ", id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", id))
		}
		logger.Error("agreement repository GetAgreement db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get doc with id: "+id, errors.New("database error"))
	}

	logger.Info("agreement repository GetAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &returnedAgreement, nil
}

func (a agreementRepository) SearchAgreements(ctx context.Context, key string, val string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository SearchAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	var resultAgreements []domain.Agreement

	// nameFilter := bson.D{primitive.E{Key: "title", Value: val}}
	nameFilter := bson.M{"title": bson.M{
		"$regex": primitive.Regex{Pattern: ".*" + val + ".*", Options: "i"},
	}}
	tagsFilter := bson.M{"tags": bson.M{
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

	updater := bson.D{
		primitive.E{
			Key: "$push", Value: bson.D{
				primitive.E{Key: "participants", Value: friendId},
			},
		},
		primitive.E{
			Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
			},
		},
	}

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
		logger.Error(fmt.Sprintf("agreement repository AddUserToAgreement - No agreement found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
	}

	logger.Info("agreement repository AddUserToAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return friendId, nil
}

func (a agreementRepository) RemoveUserFromAgreement(ctx context.Context, agreementId, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement repository RemoveUserFromAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreementId}}

	updater := bson.D{
		primitive.E{
			Key: "$pull", Value: bson.D{
				primitive.E{Key: "participants", Value: friendId},
			},
		},
		primitive.E{
			Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
			},
		},
	}

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
		logger.Error(fmt.Sprintf("agreement repository RemoveUserFromAgreement - No agreement found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
	}

	logger.Info("agreement repository RemoveUserFromAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return friendId, nil
}

func (a agreementRepository) SetDeadline(ctx context.Context, agreementId string, deadline domain.Deadline) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository SetDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreementId}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "agreement_deadline", Value: deadline},
		primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res := collection.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

	if res.Err() != nil {
		if res.Err().Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreementId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
		}
		logger.Error(fmt.Sprintf("agreement repository SetDeadline could not FindOneAndUpdate id: %s", agreementId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", agreementId), errors.New("database error"))
	}

	var resAgreement domain.Agreement
	decodeErr := res.Decode(&resAgreement)
	if decodeErr != nil {
		logger.Error("agreement repository SetDeadline could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
	}

	logger.Info("agreement repository SetDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) SetDeadlineDirected(ctx context.Context, id string, deadline domain.Deadline, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository SetDeadlineDirected start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
		filter := bson.D{primitive.E{Key: "_id", Value: id}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "agreement_deadline", Value: deadline},
			primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
		}}}

		res := agreementColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No agreement found for id: %s: ", id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", id))
			}
			logger.Error(fmt.Sprintf("agreement repository SetDeadlineDirected could not FindOneAndUpdate id: %s", id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", id), errors.New("database error"))
		}

		var resAgreement domain.Agreement
		decodeErr := res.Decode(&resAgreement)
		if decodeErr != nil {
			logger.Error("agreement repository SetDeadlineDirected could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository SetDeadlineDirected transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return resAgreement, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository SetDeadlineDirected - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository SetDeadlineDirected - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resAgreement, ok := res.(domain.Agreement)
	if !ok {
		logger.Error("agreement repository SetDeadlineDirected - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("agreement repository SetDeadlineDirected start", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) DeleteDeadline(ctx context.Context, agreementId string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository DeleteDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: agreementId}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "agreement_deadline.status", Value: "deleted"},
		primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
	}}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res := collection.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

	if res.Err() != nil {
		if res.Err().Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreementId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
		}
		logger.Error(fmt.Sprintf("agreement repository DeleteDeadline could not FindOneAndUpdate id: %s", agreementId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", agreementId), errors.New("database error"))
	}

	var resAgreement domain.Agreement
	decodeErr := res.Decode(&resAgreement)
	if decodeErr != nil {
		logger.Error("agreement repository DeleteDeadline could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
	}

	logger.Info("agreement repository DeleteDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) DeleteDeadlineDirected(ctx context.Context, id string, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository DeleteDeadlineDirected start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
		filter := bson.D{primitive.E{Key: "_id", Value: id}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "agreement_deadline.status", Value: "deleted"},
			primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
		}}}

		res := agreementColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No agreement found for id: %s: ", id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", id))
			}
			logger.Error(fmt.Sprintf("agreement repository DeleteDeadlineDirected could not FindOneAndUpdate id: %s", id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", id), errors.New("database error"))
		}

		var resAgreement domain.Agreement
		decodeErr := res.Decode(&resAgreement)
		if decodeErr != nil {
			logger.Error("agreement repository DeleteDeadlineDirected could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository DeleteDeadlineDirected transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return resAgreement, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository DeleteDeadlineDirected - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository DeleteDeadlineDirected - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resAgreement, ok := res.(domain.Agreement)
	if !ok {
		logger.Error("agreement repository DeleteDeadlineDirected - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("agreement repository DeleteDeadlineDirected finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) ActionAndNotification(ctx context.Context, actionInputs []string, notification domain.Notification) (*domain.Notification, rest_errors.RestError) {
	logger.Info("agreement repository ActionAndNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// insert notification
		if _, err := notificationColl.InsertOne(sessCtx, notification); err != nil {
			return nil, err
		}

		filter := bson.D{primitive.E{Key: "_id", Value: notification.AgreementId}}
		updater1 := bson.D{primitive.E{Key: actionInputs[0], Value: bson.D{
			primitive.E{Key: actionInputs[1], Value: notification.UserId},
		}}}

		_, err2 := agreementColl.UpdateOne(sessCtx, filter, updater1)
		if err2 != nil {
			return nil, err2
		}

		// update agreement slices
		if len(actionInputs) > 2 {
			updater2 := bson.D{primitive.E{Key: actionInputs[2], Value: bson.D{
				primitive.E{Key: actionInputs[3], Value: notification.UserId},
			}}}

			_, err := agreementColl.UpdateOne(sessCtx, filter, updater2)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository ActionAndNoritication - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		logger.Error("agreement repository ActionAndNoritication - transaction failed", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	// fmt.Printf("%+v\n", result)

	logger.Info("agreement repository ActionAndNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &notification, nil
}
