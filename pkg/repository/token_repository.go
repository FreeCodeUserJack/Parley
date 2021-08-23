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
)

type TokenRepositoryInterface interface {
	SaveToken(context.Context, domain.TokenDetails) (*domain.TokenDetails, rest_errors.RestError)
	GetToken(context.Context, string) (*domain.Token, rest_errors.RestError)
}

type tokenRepository struct {
}

func NewTokenRepository() TokenRepositoryInterface {
	return &tokenRepository{}
}

func (t tokenRepository) SaveToken(ctx context.Context, token domain.TokenDetails) (*domain.TokenDetails, rest_errors.RestError) {
	logger.Info("token repository SaveToken start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.TokenCollectionName)

	token1 := domain.Token{
		UserId:      token.UserId,
		Id:          token.AccessUuid,
		ExpiresAt:   time.Unix(token.AtExpires, 0),
		TokenString: token.AccessToken,
	}

	token2 := domain.Token{
		UserId:      token.UserId,
		Id:          token.RefreshUuid,
		ExpiresAt:   time.Unix(token.RtExpires, 0),
		TokenString: token.RefreshToken,
	}

	input := make([]interface{}, 0)
	input = append(input, token1)
	input = append(input, token2)

	// fmt.Printf("%+v", input)

	_, dbErr := collection.InsertMany(ctx, input)
	if dbErr != nil {
		logger.Error(fmt.Sprintf("token repository SaveToken - error when trying to save token: %v", token), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to save token", errors.New("database error"))
	}

	logger.Info("token repository SaveToken finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &token, nil
}

func (t tokenRepository) GetToken(ctx context.Context, id string) (*domain.Token, rest_errors.RestError) {
	logger.Info("token repository GetToken start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.TokenCollectionName)

	filter := bson.D{primitive.E{Key: "_id", Value: id}}

	var token domain.Token

	dbErr := collection.FindOne(ctx, filter).Decode(&token)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("token repository GetToken - No token found for id: %s: ", id), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No token found for id: %s", id))
		}
		logger.Error("token repository GetToken db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get doc with id: "+id, errors.New("database error"))
	}

	logger.Info("token repository GetToken finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &token, nil
}
