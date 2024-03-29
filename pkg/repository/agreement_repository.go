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
	CollaborativeUpdateAgreementNotifications(context.Context, domain.Agreement, []domain.Notification) (string, rest_errors.RestError)
	UpdateAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	UpdateAgreementNotifications(context.Context, domain.Agreement, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	GetAgreement(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	SearchAgreements(context.Context, string, string) ([]domain.Agreement, rest_errors.RestError)
	AddUserToAgreement(context.Context, string, string) (string, rest_errors.RestError)
	RemoveUserFromAgreement(context.Context, string, string) (string, rest_errors.RestError)
	SetDeadline(context.Context, string, domain.Deadline) (*domain.Agreement, rest_errors.RestError)
	SetDeadlineDirected(context.Context, string, domain.Deadline, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadline(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadlineDirected(context.Context, string, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	ActionAndNotification(context.Context, []string, domain.Notification) (*domain.Notification, rest_errors.RestError)
	UpdateAgreementRead(context.Context, domain.Agreement, string) (*domain.Agreement, rest_errors.RestError)
	RespondAgreementChange(context.Context, domain.Agreement, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	NewEventAgreement(context.Context, domain.Agreement, []domain.Notification) (*domain.Agreement, rest_errors.RestError)
	GetAgreementEventResponses(context.Context, string) ([]domain.EventResponse, rest_errors.RestError)
	InviteUsersToEvent(context.Context, domain.Agreement, []domain.Notification) (string, rest_errors.RestError)
	RespondEventInvite(context.Context, domain.Agreement, domain.EventResponse) (*domain.EventResponse, rest_errors.RestError)
}

type agreementRepository struct {
}

func NewAgreementRepository() AgreementRepositoryInterface {
	return &agreementRepository{}
}

func (a agreementRepository) NewAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository NewAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository NewAgreement - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	session.StartTransaction()

	// Insert New Agreement
	_, dbErr1 := agreementColl.InsertOne(ctx, agreement)
	if dbErr1 != nil {
		session.AbortTransaction(ctx)
		logger.Error("agreement repository NewAgreement - error when trying to create new agreement", dbErr1, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to create new agreement", errors.New("database error"))
	}

	fmt.Printf("user id: %s\n\n", agreement.CreatedBy)

	// Update User
	filter := bson.D{primitive.E{Key: "_id", Value: agreement.CreatedBy}}

	updater := bson.D{
		primitive.E{Key: "$push", Value: bson.D{
			primitive.E{Key: "agreements", Value: agreement.Id}},
		},
		primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
		},
	}

	_, dbErr := userColl.UpdateOne(ctx, filter, updater)

	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			session.AbortTransaction(ctx)
			logger.Error(fmt.Sprintf("No user found for id: %s: ", agreement.CreatedBy), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", agreement.Id))
		}
		session.AbortTransaction(ctx)
		logger.Error(fmt.Sprintf("agreement repository NewAgreement could not FindOneAndUpdate user id: %s", agreement.CreatedBy), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user id: %s", agreement.CreatedBy), errors.New("database error"))
	}

	session.CommitTransaction(ctx)

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

// To Update Agreement via update, close, set/delete deadline and send notifications for collaborative type agreements
func (a agreementRepository) CollaborativeUpdateAgreementNotifications(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (string, rest_errors.RestError) {
	logger.Info("agreement repository CloseAgreementNotifications start", context_utils.GetTraceAndClientIds(ctx)...)

	fmt.Printf("%+v\n\n", agreement)

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
		filter := bson.D{primitive.E{Key: "_id", Value: agreement.Id}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "updated_agreement", Value: agreement.UpdatedAgreement},
			primitive.E{Key: "status", Value: agreement.Status},
		}}}

		res1, err1 := agreementColl.UpdateOne(sessCtx, filter, updater)
		if err1 != nil {
			logger.Error("agreement repository CloseAgreementNotifications db error", err1, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to close doc with id: "+agreement.Id, errors.New("database error"))
		}

		if res1.MatchedCount == 0 {
			logger.Error("agreement repository CloseAgreementNotifications no doc found", errors.New("no doc with id: "+agreement.Id+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("doc with id: " + agreement.Id + " not found")
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository CloseAgreementNotifications transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository CloseAgreementNotifications - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository CloseAgreementNotifications - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository CloseAgreementNotifications finish", context_utils.GetTraceAndClientIds(ctx)...)
	return agreement.Id, nil
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
		primitive.E{Key: "tags", Value: agreement.Tags},
		primitive.E{Key: "location", Value: agreement.Location},
		primitive.E{Key: "updated_agreement", Value: agreement.UpdatedAgreement},
		primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
		primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline},
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

func (a agreementRepository) UpdateAgreementNotifications(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
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
			primitive.E{Key: "tags", Value: agreement.Tags},
			primitive.E{Key: "location", Value: agreement.Location},
			primitive.E{Key: "updated_agreement", Value: agreement.UpdatedAgreement},
			primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
			primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline},
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

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
		res, dbErr := agreementColl.UpdateOne(ctx, filter, updater)
		if dbErr != nil {
			logger.Error(fmt.Sprintf("agreement repository RemoveUserFromAgreement failed to update (agreementId:friendId): %s:%s", agreementId, friendId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return "", rest_errors.NewInternalServerError("", errors.New("database error"))
		}

		if res.MatchedCount == 0 {
			logger.Error(fmt.Sprintf("agreement repository RemoveUserFromAgreement - No agreement found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return "", rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreementId))
		}

		// Update User
		filter2 := bson.D{primitive.E{Key: "_id", Value: friendId}}

		updater2 := bson.D{
			primitive.E{Key: "$pull", Value: bson.D{
				primitive.E{Key: "agreements", Value: agreementId}},
			},
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
			},
		}

		_, dbErr2 := userColl.UpdateOne(ctx, filter2, updater2)

		if dbErr2 != nil {
			if dbErr2.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", friendId), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", friendId))
			}
			logger.Error(fmt.Sprintf("agreement repository NewEventAgreement could not update user id: %s", friendId), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user id: %s", friendId), errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository RemoveUserFromAgreement - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository RemoveUserFromAgreement - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
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
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to set deadline and get doc back id: %s", agreementId), errors.New("database error"))
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
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to set deadline and get doc back id: %s", id), errors.New("database error"))
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
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// insert notification
		if _, err := notificationColl.InsertOne(sessCtx, notification); err != nil {
			return nil, err
		}

		filter := bson.D{primitive.E{Key: "_id", Value: notification.AgreementId}}
		updater1 := bson.D{primitive.E{Key: actionInputs[0], Value: bson.D{
			primitive.E{Key: actionInputs[1], Value: notification.UserId}}},
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
			},
		}

		_, err2 := agreementColl.UpdateOne(sessCtx, filter, updater1)
		if err2 != nil {
			return nil, err2
		}

		// update agreement slices
		if len(actionInputs) > 2 {
			updater2 := bson.D{primitive.E{Key: actionInputs[2], Value: bson.D{
				primitive.E{Key: actionInputs[3], Value: notification.UserId}}},
				primitive.E{Key: "$set", Value: bson.D{
					primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
				},
			}

			_, err := agreementColl.UpdateOne(sessCtx, filter, updater2)
			if err != nil {
				return nil, err
			}
		}

		// Update User if acceptInvite or acceptRequest
		filter2 := bson.D{}

		if notification.Action == "acceptInvite" || notification.Action == "acceptRemove" {
			filter2 = append(filter2, primitive.E{Key: "_id", Value: notification.ContactId})
		}

		if notification.Action == "acceptRequest" || notification.Action == "acceptLeave" {
			filter2 = append(filter2, primitive.E{Key: "_id", Value: notification.UserId})
		}

		var updater bson.D

		if notification.Action == "acceptInvite" || notification.Action == "acceptRequest" {
			updater = bson.D{primitive.E{Key: "$push", Value: bson.D{
				primitive.E{Key: "agreements", Value: notification.AgreementId},
			}}}
		}

		if notification.Action == "acceptRemove" || notification.Action == "acceptLeave" {
			updater = bson.D{primitive.E{Key: "$pull", Value: bson.D{
				primitive.E{Key: "agreements", Value: notification.AgreementId},
			}}}
		}

		updater = append(updater, primitive.E{
			Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
			},
		})

		_, dbErr2 := userColl.UpdateOne(ctx, filter2, updater)

		if dbErr2 != nil {
			if dbErr2.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for notification: %+v: ", notification), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError("No user found for actionAndNotification")
			}
			logger.Error(fmt.Sprintf("agreement repository ActionAndNotification could not update user for notification: %+v", notification), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error trying to update user for actionAndNotification", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository ActionAndNotification - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		logger.Error("agreement repository ActionAndNotification - transaction failed", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	// fmt.Printf("%+v\n", result)

	logger.Info("agreement repository ActionAndNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &notification, nil
}

// Response to Agreement Update and set user notification to old
func (a agreementRepository) UpdateAgreementRead(ctx context.Context, agreement domain.Agreement, notificationId string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository UpdateAgreementRead start", context_utils.GetTraceAndClientIds(ctx)...)

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
		filter := bson.D{
			primitive.E{Key: "_id", Value: agreement.Id},
			primitive.E{Key: "updated_agreement", Value: bson.D{
				primitive.E{Key: "$ne", Value: nil},
			}},
		}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
			primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline}}},
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
			},
		}

		res := agreementColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreement.Id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreement.Id))
			}
			logger.Error(fmt.Sprintf("agreement repository UpdateAgreementRead could not FindOneAndUpdate id: %s", agreement.Id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update agreement read and get doc back id: %s", agreement.Id), errors.New("database error"))
		}

		var resAgreement domain.Agreement
		decodeErr := res.Decode(&resAgreement)
		if decodeErr != nil {
			logger.Error("agreement repository UpdateAgreementRead could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
		}

		// Set Notification Status to Old
		filter2 := bson.D{
			primitive.E{Key: "_id", Value: notificationId},
		}

		updater2 := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "status", Value: "old"},
		}}}

		_, dbErr := notificationColl.UpdateOne(ctx, filter2, updater2)
		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error("agreement repository UpdateAgreementRead no doc found", errors.New("no doc with id: "+notificationId+" found"), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewBadRequestError("notification doc with id: " + notificationId + " not found")
			}
			logger.Error("agreement repository UpdateAgreementRead db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to update notification doc with id: "+notificationId, errors.New("database error"))
		}

		return resAgreement, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository UpdateAgreementRead - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		if transactionErr.Error() == "not_found" {
			logger.Error("agreement repository UpdateAgreementRead - either no doc found or no notification found", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewBadRequestError("conditions not met for accept/decline")
		}
		logger.Error("agreement repository UpdateAgreementRead - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resAgreement, ok := res.(domain.Agreement)
	if !ok {
		logger.Error("agreement repository UpdateAgreementRead - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("agreement repository UpdateAgreementRead finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) RespondAgreementChange(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository RespondAgreementChange start", context_utils.GetTraceAndClientIds(ctx)...)

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

		var updater bson.D

		if notifications[0].Type == "notifyChange" {
			// fmt.Printf("%+v\n", agreement)
			updater = bson.D{primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "title", Value: agreement.Title},
				primitive.E{Key: "description", Value: agreement.Description},
				primitive.E{Key: "participants", Value: agreement.Participants},
				primitive.E{Key: "last_update_datetime", Value: agreement.LastUpdateDateTime},
				primitive.E{Key: "agreement_deadline", Value: agreement.AgreementDeadline},
				primitive.E{Key: "status", Value: agreement.Status},
				primitive.E{Key: "tags", Value: agreement.Tags},
				primitive.E{Key: "location", Value: agreement.Location},
				primitive.E{Key: "updated_agreement", Value: nil},
				primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
				primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline},
				primitive.E{Key: "public", Value: agreement.Public},
			}}}
		} else {
			updater = bson.D{primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "status", Value: agreement.Status},
				primitive.E{Key: "updated_agreement", Value: agreement.UpdatedAgreement},
				primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
				primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline},
			}}}
		}

		res := agreementColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No agreement found for id: %s: ", agreement.Id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreement.Id))
			}
			logger.Error(fmt.Sprintf("agreement repository RespondAgreementChange could not FindOneAndUpdate id: %s", agreement.Id), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update agreement and get doc back id: %s", agreement.Id), errors.New("database error"))
		}

		var resAgreement domain.Agreement
		decodeErr := res.Decode(&resAgreement)
		if decodeErr != nil {
			logger.Error("agreement repository RespondAgreementChange could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository RespondAgreementChange transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return resAgreement, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository RespondAgreementChange - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository RespondAgreementChange - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resAgreement, ok := res.(domain.Agreement)
	if !ok {
		logger.Error("agreement repository RespondAgreementChange - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("agreement repository RespondAgreementChange finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resAgreement, nil
}

func (a agreementRepository) NewEventAgreement(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement repository NewEventAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Insert Agreement
		_, dbErr := agreementColl.InsertOne(ctx, agreement)
		if dbErr != nil {
			logger.Error("agreement repository NewEventAgreement - error when trying to create new agreement", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to create new agreement", errors.New("database error"))
		}

		// Update User
		filter := bson.D{primitive.E{Key: "_id", Value: agreement.CreatedBy}}

		updater := bson.D{primitive.E{Key: "$push", Value: bson.D{
			primitive.E{Key: "agreements", Value: agreement.Id}}},
			primitive.E{
				Key: "$set", Value: bson.D{
					primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
				},
			},
		}

		_, dbErr2 := userColl.UpdateOne(ctx, filter, updater)

		if dbErr2 != nil {
			if dbErr2.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", agreement.CreatedBy), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", agreement.Id))
			}
			logger.Error(fmt.Sprintf("agreement repository NewEventAgreement could not update user id: %s", agreement.CreatedBy), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user id: %s", agreement.CreatedBy), errors.New("database error"))
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository NewEventAgreement transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository NewEventAgreement - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository NewEventAgreement - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository NewEventAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &agreement, nil
}

func (a agreementRepository) GetAgreementEventResponses(ctx context.Context, agreementId string) ([]domain.EventResponse, rest_errors.RestError) {
	logger.Info("agreement repository GetAgreementEventResponses start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "agreement_id", Value: agreementId}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.EventResponseCollectionName)

	curr, dbErr := collection.Find(ctx, filter)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("agreement repository GetAgreementEventResponses could not FindOneAndUpdate id: %s", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to get event responses for agreement id: %s", agreementId), errors.New("database error"))
	}

	var responses []domain.EventResponse

	for curr.Next(ctx) {
		var response domain.EventResponse
		decodeErr := curr.Decode(&response)
		if decodeErr != nil {
			logger.Error("agreement repository GetAgreementEventResponses error decoding data to event response type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("", errors.New("database error trying to decode data to event response"))
		}
		responses = append(responses, response)
	}

	curr.Close(ctx)

	if len(responses) == 0 {
		logger.Error(fmt.Sprintf("No event responses found for id: %s: ", agreementId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No event responses found for id: %s", agreementId))
	}

	logger.Info("agreement repository GetAgreementEventResponses finish", context_utils.GetTraceAndClientIds(ctx)...)
	return responses, nil
}

func (a agreementRepository) InviteUsersToEvent(ctx context.Context, agreement domain.Agreement, notifications []domain.Notification) (string, rest_errors.RestError) {
	logger.Info("agreement repository InviteUsersToEvent start", context_utils.GetTraceAndClientIds(ctx)...)

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
		filter := bson.D{primitive.E{Key: "_id", Value: agreement.Id}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "invited_participants", Value: agreement.InvitedParticipants}}},
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
			},
		}

		_, dbErr := agreementColl.UpdateOne(ctx, filter, updater)

		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("agreement repository InviteUsersToEvent No agreement found for id: %s: ", agreement.Id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreement.Id))
			}
			logger.Error(fmt.Sprintf("agreement repository InviteUsersToEvent could not FindOneAndUpdate id: %s", agreement.Id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to invite users to event id: %s", agreement.Id), errors.New("database error"))
		}

		// Insert Notifications
		inserts := make([]interface{}, len(notifications))
		for i := range notifications {
			inserts[i] = notifications[i]
		}
		_, insertErr := notificationColl.InsertMany(sessCtx, inserts)
		if insertErr != nil {
			logger.Error("agreement repository InviteUsersToEvent transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository InviteUsersToEvent - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository InviteUsersToEvent - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository InviteUsersToEvent finish", context_utils.GetTraceAndClientIds(ctx)...)
	return agreement.Id, nil
}

func (a agreementRepository) RespondEventInvite(ctx context.Context, agreement domain.Agreement, eventResponse domain.EventResponse) (*domain.EventResponse, rest_errors.RestError) {
	logger.Info("agreement repository RespondEventInvite start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	agreementColl := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName, wcMajorityCollectionOpts)
	eventResponseColl := client.Database(db.DatabaseName).Collection(db.EventResponseCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update Agreement
		filter := bson.D{primitive.E{Key: "_id", Value: agreement.Id}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "event_responses", Value: agreement.EventResponses},
			primitive.E{Key: "agreement_accept", Value: agreement.AgreementAccept},
			primitive.E{Key: "agreement_decline", Value: agreement.AgreementDecline}}},
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()}},
			},
		}

		_, dbErr := agreementColl.UpdateOne(ctx, filter, updater)

		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("agreement repository RespondEventInvite No agreement found for id: %s: ", agreement.Id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No agreement found for id: %s", agreement.Id))
			}
			logger.Error(fmt.Sprintf("agreement repository RespondEventInvite could not FindOneAndUpdate id: %s", agreement.Id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to respond invite for event id: %s", agreement.Id), errors.New("database error"))
		}

		// Update User if Accept
		if eventResponse.Response == "accept" {
			filter := bson.D{primitive.E{Key: "_id", Value: eventResponse.UserId}}

			updater := bson.D{primitive.E{Key: "$push", Value: bson.D{
				primitive.E{Key: "agreements", Value: agreement.Id}}},
				primitive.E{
					Key: "$set", Value: bson.D{
						primitive.E{Key: "last_update_datetime", Value: time.Now().UTC()},
					},
				},
			}

			_, dbErr2 := userColl.UpdateOne(ctx, filter, updater)

			if dbErr2 != nil {
				if dbErr2.Error() == "mongo: no documents in result" {
					logger.Error(fmt.Sprintf("No user found for id: %s: ", agreement.CreatedBy), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
					return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", agreement.Id))
				}
				logger.Error(fmt.Sprintf("agreement repository RespondEventInvite could not update user id: %s", agreement.CreatedBy), dbErr2, context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user id: %s", agreement.CreatedBy), errors.New("database error"))
			}
		}

		// Insert EventResponse
		_, eventResponseErr := eventResponseColl.InsertOne(ctx, eventResponse)
		if eventResponseErr != nil {
			logger.Error("agreement repository RespondEventInvite - error when trying to create new evetnResponse", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to create new eventResponse", errors.New("database error"))
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("agreement repository RespondEventInvite - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("agreement repository RespondEventInvite - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("agreement repository RespondEventInvite finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &eventResponse, nil
}
