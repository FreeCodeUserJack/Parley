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
)

type UserRepositoryInterface interface {
	NewUser(context.Context, domain.User) (*domain.User, rest_errors.RestError)
	GetUser(context.Context, string) (*domain.User, rest_errors.RestError)
}

type userRepository struct{}

func NewUserRepository() UserRepositoryInterface {
	return &userRepository{}
}

func (u userRepository) NewUser(ctx context.Context, user domain.User) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository NewUser start", context_utils.GetTraceAndClientIds(ctx)...)

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	_, dbErr := collection.InsertOne(ctx, user)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("user repository NewUser - error when trying to create new user: %v", user), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to create new user", errors.New("database error"))
	}

	logger.Info("user repository NewUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &user, nil
}

func (u userRepository) GetUser(ctx context.Context, userId string) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository GetUser start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: userId}}

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	var user domain.User
	dbErr := collection.FindOne(ctx, filter).Decode(&user)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("user repository GetUser No user found for id: %s: ", userId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
		}
		logger.Error(fmt.Sprintf("user repository GetUser could not GetUser id: %s", userId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to get user id: %s", userId), errors.New("database error"))
	}

	logger.Info("user repository GetUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &user, nil
}
