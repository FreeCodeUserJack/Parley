package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/FreeCodeUserJack/Parley/pkg/db"
	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/dto"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthRepositoryInterface interface {
	Login(context.Context, dto.LoginRequest) (*domain.User, rest_errors.RestError)
	Logout(context.Context, string) (string, rest_errors.RestError)
}

type authRepository struct {
}

func NewAuthRepository() AuthRepositoryInterface {
	return &authRepository{}
}

func (a authRepository) Login(ctx context.Context, loginReq dto.LoginRequest) (*domain.User, rest_errors.RestError) {
	logger.Info("auth repository Login - start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "email", Value: loginReq.Email}}

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
			logger.Error(fmt.Sprintf("auth repository Login - No user found for email: %s: ", loginReq.Email), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for email: %s", loginReq.Email))
		}
		logger.Error("auth repository Login db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error when trying to get doc for email: %s", loginReq.Email), errors.New("database error"))
	}

	user.Email = loginReq.Email

	logger.Info("auth repository Login - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &user, nil
}

func (a authRepository) Logout(ctx context.Context, id string) (string, rest_errors.RestError) {
	return "", nil
}
