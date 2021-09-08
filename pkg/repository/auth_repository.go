package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/db"
	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/dto"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type AuthRepositoryInterface interface {
	Login(context.Context, dto.LoginRequest) (*domain.User, rest_errors.RestError)
	Logout(context.Context, string) (string, rest_errors.RestError)
	VerifyEmail(context.Context, string, string) (string, rest_errors.RestError)
	GetUser(context.Context, string) (*domain.User, rest_errors.RestError)
	VerifyPhone(context.Context, string, string) (string, rest_errors.RestError)
	GetAccVerification(context.Context, string) (*domain.AccountVerification, rest_errors.RestError)
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

func (a authRepository) VerifyEmail(ctx context.Context, userId, accountVerificationId string) (string, rest_errors.RestError) {
	logger.Info("auth repository VerifyEmail - start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	emailVerificationColl := client.Database(db.DatabaseName).Collection(db.AccountVerificationCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update User
		filter := bson.D{primitive.E{Key: "_id", Value: userId}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "account_verified", Value: "true"},
			primitive.E{Key: "account_verification", Value: accountVerificationId},
		}}}

		res1, err1 := userColl.UpdateOne(sessCtx, filter, updater)
		if err1 != nil {
			logger.Error("auth repository VerifyEmail db error", err1, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to update user doc with id: "+userId, errors.New("database error"))
		}

		if res1.MatchedCount == 0 {
			logger.Error("auth repository VerifyEmail no user doc found", errors.New("no user doc with id: "+userId+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("user doc with id: " + userId + " not found")
		}

		// Update Account Verification
		filter2 := bson.D{primitive.E{Key: "_id", Value: accountVerificationId}}

		updater2 := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "read_datetime", Value: time.Now().UTC()},
			primitive.E{Key: "status", Value: "email"},
		}}}

		res2, updateErr := emailVerificationColl.UpdateOne(ctx, filter2, updater2)
		if updateErr != nil {
			logger.Error("auth repository VerifyEmail transaction to update account verification read_datetime failed", updateErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not update account verification", errors.New("database error"))
		}

		if res2.MatchedCount == 0 {
			logger.Error("auth repository VerifyEmail no acc verification doc found", errors.New("no email verification doc with id: "+userId+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("acc verification doc with id: " + userId + " not found")
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("auth repository VerifyEmail - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("auth repository VerifyEmail - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("auth repository VerifyEmail - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return userId, nil
}

func (a authRepository) GetUser(ctx context.Context, userId string) (*domain.User, rest_errors.RestError) {
	logger.Info("auth repository GetUser - start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: userId}}

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	var user domain.User

	dbErr := collection.FindOne(ctx, filter).Decode(&user)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("auth repository GetUser - No user found for id: %s: ", userId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
		}
		logger.Error("auth repository GetUser - db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get doc with id: "+userId, errors.New("database error"))
	}

	logger.Info("auth repository GetUser - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &user, nil
}

func (a authRepository) VerifyPhone(ctx context.Context, userId, accountVerificationId string) (string, rest_errors.RestError) {
	logger.Info("auth repository VerifyPhone - start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	accountVerificationColl := client.Database(db.DatabaseName).Collection(db.AccountVerificationCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update User
		filter := bson.D{primitive.E{Key: "_id", Value: userId}}

		updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "account_verified", Value: "true"},
			primitive.E{Key: "account_verification", Value: accountVerificationId},
		}}}

		res1, err1 := userColl.UpdateOne(sessCtx, filter, updater)
		if err1 != nil {
			logger.Error("auth repository VerifyPhone db error", err1, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to update user doc with id: "+userId, errors.New("database error"))
		}

		if res1.MatchedCount == 0 {
			logger.Error("auth repository VerifyPhone no user doc found", errors.New("no user doc with id: "+userId+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("user doc with id: " + userId + " not found")
		}

		// Update Account Verification
		filter2 := bson.D{primitive.E{Key: "_id", Value: accountVerificationId}}

		updater2 := bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "read_datetime", Value: time.Now().UTC()},
			primitive.E{Key: "status", Value: "phone"},
		}}}

		res2, updateErr := accountVerificationColl.UpdateOne(ctx, filter2, updater2)
		if updateErr != nil {
			logger.Error("auth repository VerifyPhone transaction to update account verification read_datetime failed", updateErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not update account verification", errors.New("database error"))
		}

		if res2.MatchedCount == 0 {
			logger.Error("auth repository VerifyPhone no acc verification doc found", errors.New("no email verification doc with id: "+userId+" found"), context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewNotFoundError("acc verification doc with id: " + userId + " not found")
		}

		return nil, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("auth repository VerifyPhone - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	_, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("auth repository VerifyPhone - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	logger.Info("auth repository VerifyPhone - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return userId, nil
}

func (a authRepository) GetAccVerification(ctx context.Context, userId string) (*domain.AccountVerification, rest_errors.RestError) {
	logger.Info("auth repository GetAccVerification - start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AccountVerificationCollectionName)

	filter := bson.D{
		primitive.E{Key: "user_id", Value: userId},
		primitive.E{Key: "status", Value: "new"},
	}

	var accver domain.AccountVerification
	dbErr := collection.FindOne(ctx, filter).Decode(&accver)
	if dbErr != nil {
		if dbErr.Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("auth repository GetAccVerification - No acc verification found for id: %s: ", userId), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No acc verification found for user id: %s", userId))
		}
		logger.Error("auth repository GetAccVerification - db error", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get doc with id: "+userId, errors.New("database error"))
	}

	logger.Info("auth repository GetAccVerification - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &accver, nil
}
