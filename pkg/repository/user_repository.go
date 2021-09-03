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

type UserRepositoryInterface interface {
	NewUser(context.Context, domain.User) (*domain.User, rest_errors.RestError)
	GetUser(context.Context, string) (*domain.User, rest_errors.RestError)
	UpdateUser(context.Context, string, domain.User) (*domain.User, rest_errors.RestError)
	DeleteUser(context.Context, string) (*domain.User, rest_errors.RestError)
	GetFriends(context.Context, string, []string) ([]domain.User, rest_errors.RestError)
	RemoveFriend(context.Context, string, string, domain.Notification) (*domain.User, rest_errors.RestError)
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

func (u userRepository) UpdateUser(ctx context.Context, userId string, user domain.User) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository UpdateUser start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: userId}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "first_name", Value: user.FirstName},
		primitive.E{Key: "last_name", Value: user.LastName},
		primitive.E{Key: "dob", Value: user.DOB},
		primitive.E{Key: "status", Value: user.Status},
		primitive.E{Key: "public", Value: user.Public},
		primitive.E{Key: "phone", Value: user.Phone},
	}}}

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	res := collection.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

	if res.Err() != nil {
		if res.Err().Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("No user found for id: %s: ", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
		}
		logger.Error(fmt.Sprintf("user repository UpdateUser could not FindOneAndUpdate id: %s", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", userId), errors.New("database error"))
	}

	var retUser domain.User
	decodeErr := res.Decode(&retUser)
	if decodeErr != nil {
		logger.Error("user repository UpdateUser could not decode update doc to User type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
	}

	logger.Info("user repository UpdateUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &retUser, nil
}

func (u userRepository) DeleteUser(ctx context.Context, userId string) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository DeleteUser start", context_utils.GetTraceAndClientIds(ctx)...)

	filter := bson.D{primitive.E{Key: "_id", Value: userId}}

	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "status", Value: "deleted"},
	}}}

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	res := collection.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

	if res.Err() != nil {
		if res.Err().Error() == "mongo: no documents in result" {
			logger.Error(fmt.Sprintf("No user found for id: %s: ", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
		}
		logger.Error(fmt.Sprintf("user repository DeleteUser could not FindOneAndUpdate id: %s", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", userId), errors.New("database error"))
	}

	var retUser domain.User
	decodeErr := res.Decode(&retUser)
	if decodeErr != nil {
		logger.Error("user repository UpdateUser could not decode update doc to User type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
	}

	logger.Info("user repository DeleteUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &retUser, nil
}

func (u userRepository) GetFriends(ctx context.Context, userId string, uuids []string) ([]domain.User, rest_errors.RestError) {
	logger.Info("user repository GetFriends Start", context_utils.GetTraceAndClientIds(ctx)...)

	client, err := db.GetMongoClient()
	if err != nil {
		logger.Error("error when trying to get db client", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	filter := bson.D{primitive.E{Key: "_id", Value: bson.D{
		primitive.E{Key: "$in", Value: uuids},
	}}}

	curr, findErr := collection.Find(ctx, filter)
	if findErr != nil {
		logger.Error("user repository GetFriends - find error for userId: "+userId, findErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error trying to get friends for userId: "+userId, errors.New("database error"))
	}

	var friends []domain.User
	for curr.Next(ctx) {
		buf := domain.User{}
		err := curr.Decode(&buf)
		if err != nil {
			logger.Error("user repository GetFriends - could not decode db user to user instance", err, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error trying to decode db obj to user instance", errors.New("datbase error"))
		}
		friends = append(friends, buf)
	}

	curr.Close(ctx)

	if len(friends) == 0 {
		logger.Error("no friends found for list of userIds", errors.New("no documents found for search"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewNotFoundError("no friends found for userId: " + userId)
	}

	logger.Info("user repository GetFriends finish", context_utils.GetTraceAndClientIds(ctx)...)
	return friends, nil
}

func (u userRepository) RemoveFriend(ctx context.Context, userId, friendId string, notification domain.Notification) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository RemoveFriend start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	notificationColl := client.Database(db.DatabaseName).Collection(db.NotificationCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update User
		filter := bson.D{primitive.E{Key: "_id", Value: userId}}

		updater := bson.D{primitive.E{Key: "$pull", Value: bson.D{
			primitive.E{Key: "friends", Value: friendId},
		}}}

		res := userColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
			}
			logger.Error(fmt.Sprintf("user repository RemoveFriend could not FindOneAndUpdate id: %s", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", userId), errors.New("database error"))
		}

		var retUser domain.User
		decodeErr := res.Decode(&retUser)
		if decodeErr != nil {
			logger.Error("user repository RemoveFriend could not decode update doc to Agreement type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to retrieve updated document", errors.New("database error"))
		}

		// Update Friend
		filter2 := bson.D{primitive.E{Key: "_id", Value: friendId}}

		updater2 := bson.D{primitive.E{Key: "$pull", Value: bson.D{
			primitive.E{Key: "friends", Value: userId},
		}}}

		_, dbErr := userColl.UpdateOne(ctx, filter2, updater2)
		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", friendId))
			}
			logger.Error(fmt.Sprintf("user repository RemoveFriend could not FindOneAndUpdate id: %s", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to delete deadline and get doc back id: %s", friendId), errors.New("database error"))
		}

		// Insert Notification
		_, insertErr := notificationColl.InsertOne(sessCtx, notification)
		if insertErr != nil {
			logger.Error("user repository RemoveFriend transaction to insert notifications failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notifications", errors.New("database error"))
		}

		return retUser, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("user repository RemoveFriend - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("user repository RemoveFriend - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resUser, ok := res.(domain.User)
	if !ok {
		logger.Error("user repository RemoveFriend - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("user repository RemoveFriend finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resUser, nil
}
