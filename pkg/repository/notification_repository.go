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

type NotificationRepositoryInterface interface {
	SaveNotification(context.Context, domain.Notification) (*domain.Notification, rest_errors.RestError)
	DeleteNotification(context.Context, string) rest_errors.RestError
	GetUserNotifications(context.Context, string, string) ([]domain.Notification, rest_errors.RestError)
}

type notificationRepository struct{}

func NewNotificationRepository() NotificationRepositoryInterface {
	return &notificationRepository{}
}

func (n notificationRepository) SaveNotification(ctx context.Context, notification domain.Notification) (*domain.Notification, rest_errors.RestError) {
	logger.Info("notification repository SaveNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName)

	_, dbErr := collection.InsertOne(ctx, notification)
	if dbErr != nil {
		logger.Error("notification repository SaveNotification - error when trying to create new agreement", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to create new notification", errors.New("database error"))
	}

	logger.Info("notification repository SaveNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &notification, nil
}

func (n notificationRepository) DeleteNotification(ctx context.Context, id string) rest_errors.RestError {
	logger.Info("notification repository DeleteNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	res, dbErr := collection.DeleteOne(ctx, filter)
	if dbErr != nil {
		logger.Error("notification repository DeleteNotification db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return rest_errors.NewInternalServerError("error when trying to delete doc with id: "+id, errors.New("database error"))
	} else if res.DeletedCount == 0 {
		logger.Error("notification repository DeleteNotification no doc found", errors.New("no doc with id: "+id+" found"), context_utils.GetTraceAndClientIds(ctx)...)
		return rest_errors.NewNotFoundError("doc with id: " + id + " not found")
	}

	logger.Info("notification repository DeleteNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return nil
}

func (n notificationRepository) GetUserNotifications(ctx context.Context, userId, statusVal string) ([]domain.Notification, rest_errors.RestError) {
	logger.Info("notification repository GetUserNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	var filter bson.D

	if statusVal == "old" || statusVal == "new" {
		filter = bson.D{primitive.E{Key: "user_id", Value: userId}, primitive.E{Key: "status", Value: statusVal}}
	} else {
		filter = bson.D{primitive.E{Key: "user_id", Value: userId}}
	}

	var notifications []domain.Notification

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName)

	cur, findError := collection.Find(ctx, filter)
	if findError != nil {
		logger.Error("notification repository GetUserNotification - could not find for user id: "+userId, findError, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to find user with id: "+userId, errors.New("database error"))
	}

	for cur.Next(ctx) {
		note := domain.Notification{}
		err := cur.Decode(&note)
		if err != nil {
			logger.Error("notification repository GetUserNotification - decode error for user id: "+userId, findError, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to decode notifications for user id: "+userId, err)
		}

		notifications = append(notifications, note)
	}

	cur.Close(ctx)

	// it's ok if no notifications, don't need to return error - front end checks length of returned notifications

	logger.Info("notification repository GetUserNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	return notifications, nil
}
