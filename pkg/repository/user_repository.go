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
	NewUserVerifyEmail(context.Context, domain.User, domain.EmailVerification) (*domain.User, rest_errors.RestError)
	GetUser(context.Context, string) (*domain.User, rest_errors.RestError)
	UpdateUser(context.Context, string, domain.User) (*domain.User, rest_errors.RestError)
	DeleteUser(context.Context, string) (*domain.User, rest_errors.RestError)
	GetFriends(context.Context, string, []string) ([]domain.User, rest_errors.RestError)
	RemoveFriend(context.Context, string, string, domain.Notification) (*domain.User, rest_errors.RestError)
	SearchUsers(context.Context, [][]string) ([]domain.User, rest_errors.RestError)
	GetAgreements(context.Context, string) ([]domain.Agreement, rest_errors.RestError)
	AddFriend(context.Context, string, string, domain.Notification) (*domain.User, rest_errors.RestError)
	RespondFriendRequest(context.Context, string, string, domain.Notification) (*domain.User, rest_errors.RestError)
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

func (u userRepository) NewUserVerifyEmail(ctx context.Context, user domain.User, emailVerification domain.EmailVerification) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository NewUserVerifyEmail start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	emailVerificationColl := client.Database(db.DatabaseName).Collection(db.EmailVerificationCollectionName, wcMajorityCollectionOpts)
	userColl := client.Database(db.DatabaseName).Collection(db.UsersCollectionName, wcMajorityCollectionOpts)

	// callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
	// 	// Insert User
	// 	_, dbErr := userColl.InsertOne(ctx, user)

	// 	if dbErr != nil {
	// 		logger.Error(fmt.Sprintf("user repository NewUserVerifyEmail could not insert user: %+v", user), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
	// 		return nil, rest_errors.NewInternalServerError("error trying to insert user", errors.New("database error"))
	// 	}

	// 	// Insert Email Verification
	// 	_, insertErr := emailVerificationColl.InsertOne(sessCtx, emailVerification)
	// 	if insertErr != nil {
	// 		logger.Error("user repository NewUserVerifyEmail transaction to insert email verification failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
	// 		return nil, rest_errors.NewInternalServerError("could not insert email verification", errors.New("database error"))
	// 	}

	// 	return nil, nil
	// }

	session, err := client.StartSession()
	if err != nil {
		logger.Error("user repository NewUserVerifyEmail - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	// _, transactionErr := session.WithTransaction(ctx, callback)
	// if transactionErr != nil {
	// 	session.AbortTransaction(ctx)
	// 	logger.Error("user repository NewUserVerifyEmail - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
	// 	return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	// }

	session.StartTransaction()

	_, dbErr := userColl.InsertOne(ctx, user)

	if dbErr != nil {
		session.AbortTransaction(ctx)
		logger.Error(fmt.Sprintf("user repository NewUserVerifyEmail could not insert user: %+v", user), dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error trying to insert user", errors.New("database error"))
	}

	// Insert Email Verification
	_, insertErr := emailVerificationColl.InsertOne(ctx, emailVerification)
	if insertErr != nil {
		session.AbortTransaction(ctx)
		logger.Error("user repository NewUserVerifyEmail transaction to insert email verification failed", insertErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("could not insert email verification", errors.New("database error"))
	}

	session.CommitTransaction(ctx)

	logger.Info("user repository NewUserVerifyEmail finish", context_utils.GetTraceAndClientIds(ctx)...)
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
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user and get doc back id: %s", userId), errors.New("database error"))
		}

		var retUser domain.User
		decodeErr := res.Decode(&retUser)
		if decodeErr != nil {
			logger.Error("user repository RemoveFriend could not decode update doc to User type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to decode db obj into user instance", errors.New("database error"))
		}

		// Update Friend
		filter2 := bson.D{primitive.E{Key: "_id", Value: friendId}}

		updater2 := bson.D{primitive.E{Key: "$pull", Value: bson.D{
			primitive.E{Key: "friends", Value: userId},
		}}}

		_, dbErr := userColl.UpdateOne(ctx, filter2, updater2)
		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No friend found for id: %s: ", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No friend found for id: %s", friendId))
			}
			logger.Error(fmt.Sprintf("user repository RemoveFriend could not Update friend id: %s", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update friend id: %s", friendId), errors.New("database error"))
		}

		// Insert Notification
		_, insertErr := notificationColl.InsertOne(sessCtx, notification)
		if insertErr != nil {
			logger.Error("user repository RemoveFriend transaction to insert notification failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notification", errors.New("database error"))
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

func (u userRepository) SearchUsers(ctx context.Context, queries [][]string) ([]domain.User, rest_errors.RestError) {
	logger.Info("user repository SearchUsers start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.UsersCollectionName)

	filter := bson.D{}

	for _, query := range queries {
		filter = append(filter, primitive.E{Key: query[0], Value: bson.D{
			primitive.E{Key: "$regex", Value: primitive.Regex{
				Pattern: ".*" + query[1] + ".*", Options: "i",
			}},
		}})
	}

	curr, dbErr := collection.Find(ctx, filter)

	if dbErr != nil {
		logger.Error("user repository SearchUsers - error trying to find users", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error trying to search users", errors.New("database error"))
	}

	var res []domain.User

	for curr.Next(ctx) {
		buf := domain.User{}
		err := curr.Decode(&buf)
		if err != nil {
			logger.Error("user repository SearchUsers - bson decode error from mongo to user instance", err, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error trying to decode searched user into user instance", errors.New("database error"))
		}
		res = append(res, buf)
	}

	if len(res) == 0 {
		logger.Error("user repository SearchUsers - no users found for search", fmt.Errorf("no users found for this queries: %v", queries), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewNotFoundError("no users found for search users")
	}

	logger.Info("user repository SearchUsers finish", context_utils.GetTraceAndClientIds(ctx)...)
	return res, nil
}

func (u userRepository) GetAgreements(ctx context.Context, userId string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("user repository GetAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	client, mongoErr := db.GetMongoClient()
	if mongoErr != nil {
		logger.Error("error when trying to get db client", mongoErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error when trying to get db client", errors.New("database error"))
	}

	collection := client.Database(db.DatabaseName).Collection(db.AgreementCollectionName)

	filter := bson.D{primitive.E{Key: "created_by", Value: userId}}

	curr, dbErr := collection.Find(ctx, filter)
	if dbErr != nil {
		logger.Error("user repository GetAgreements - error trying to find agreements", dbErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("error trying to search agreements", errors.New("database error"))
	}

	var res []domain.Agreement

	for curr.Next(ctx) {
		buf := domain.Agreement{}
		err := curr.Decode(&buf)
		if err != nil {
			logger.Error("user repository GetAgreements - bson decode error from mongo to agreement instance", err, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error trying to decode searched agreement into agreement instance", errors.New("database error"))
		}
		res = append(res, buf)
	}

	if len(res) == 0 {
		logger.Error("user repository GetAgreements - no agreements found for search", fmt.Errorf("no agreements found for userId: %s", userId), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewNotFoundError("no agreements found for search users")
	}

	logger.Info("user repository GetAgreements finish", context_utils.GetTraceAndClientIds(ctx)...)
	return res, nil
}

func (u userRepository) AddFriend(ctx context.Context, userId, friendId string, notification domain.Notification) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository AddFriend start", context_utils.GetTraceAndClientIds(ctx)...)

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

		updater := bson.D{primitive.E{Key: "$push", Value: bson.D{
			primitive.E{Key: "sent_friend_requests", Value: friendId},
		}}}

		res := userColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
			}
			logger.Error(fmt.Sprintf("user repository AddFriend could not FindOneAndUpdate id: %s", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user and get doc back id: %s", userId), errors.New("database error"))
		}

		var retUser domain.User
		decodeErr := res.Decode(&retUser)
		if decodeErr != nil {
			logger.Error("user repository AddFriend could not decode update doc to User type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to decode db obj to user instance", errors.New("database error"))
		}

		// Update Friend
		filter2 := bson.D{primitive.E{Key: "_id", Value: friendId}}

		updater2 := bson.D{primitive.E{Key: "$push", Value: bson.D{
			primitive.E{Key: "pending_friend_requests", Value: userId},
		}}}

		_, dbErr := userColl.UpdateOne(ctx, filter2, updater2)
		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No friend found for id: %s: ", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No friend found for id: %s", friendId))
			}
			logger.Error(fmt.Sprintf("user repository AddFriend could not Update friend id: %s", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update friend: %s", friendId), errors.New("database error"))
		}

		// Insert Notification
		_, insertErr := notificationColl.InsertOne(sessCtx, notification)
		if insertErr != nil {
			logger.Error("user repository AddFriend transaction to insert notification failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notification", errors.New("database error"))
		}

		return retUser, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("user repository AddFriend - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("user repository AddFriend - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resUser, ok := res.(domain.User)
	if !ok {
		logger.Error("user repository AddFriend - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("user repository AddFriend finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resUser, nil
}

func (u userRepository) RespondFriendRequest(ctx context.Context, userId, friendId string, notification domain.Notification) (*domain.User, rest_errors.RestError) {
	logger.Info("user repository RespondFriendRequest start", context_utils.GetTraceAndClientIds(ctx)...)

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

		updater := bson.D{
			primitive.E{
				Key: "$pull", Value: bson.D{
					primitive.E{Key: "pending_friend_requests", Value: friendId},
				},
			},
		}

		if notification.Type == "notifyAcceptFriendInvite" {
			updater = append(updater, primitive.E{Key: "$push", Value: bson.D{
				primitive.E{Key: "friends", Value: friendId},
			}})
		}

		res := userColl.FindOneAndUpdate(ctx, filter, updater, options.FindOneAndUpdate().SetReturnDocument(options.After))

		if res.Err() != nil {
			if res.Err().Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No user found for id: %s: ", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No user found for id: %s", userId))
			}
			logger.Error(fmt.Sprintf("user repository RespondFriendRequest could not FindOneAndUpdate id: %s", userId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update user and get doc back id: %s", userId), errors.New("database error"))
		}

		var retUser domain.User
		decodeErr := res.Decode(&retUser)
		if decodeErr != nil {
			logger.Error("user repository RespondFriendRequest could not decode update doc to User type instance", decodeErr, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError("error when trying to decode db obj to user instance", errors.New("database error"))
		}

		// Update Friend
		filter2 := bson.D{primitive.E{Key: "_id", Value: friendId}}

		updater2 := bson.D{primitive.E{Key: "$pull", Value: bson.D{
			primitive.E{Key: "sent_friend_requests", Value: userId},
		}}}

		if notification.Type == "notifyAcceptFriendInvite" {
			updater2 = append(updater2, primitive.E{Key: "$push", Value: bson.D{
				primitive.E{Key: "friends", Value: userId},
			}})
		}

		_, dbErr := userColl.UpdateOne(ctx, filter2, updater2)
		if dbErr != nil {
			if dbErr.Error() == "mongo: no documents in result" {
				logger.Error(fmt.Sprintf("No friend found for id: %s: ", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
				return nil, rest_errors.NewNotFoundError(fmt.Sprintf("No friend found for id: %s", friendId))
			}
			logger.Error(fmt.Sprintf("user repository RespondFriendRequest could not Update friend id: %s", friendId), res.Err(), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewInternalServerError(fmt.Sprintf("error trying to update friend: %s", friendId), errors.New("database error"))
		}

		// Insert Notification
		_, insertErr := notificationColl.InsertOne(sessCtx, notification)
		if insertErr != nil {
			logger.Error("user repository RespondFriendRequest transaction to insert notification failed", insertErr, context_utils.GetTraceAndClientIds(sessCtx)...)
			return nil, rest_errors.NewInternalServerError("could not insert notification", errors.New("database error"))
		}

		return retUser, nil
	}

	session, err := client.StartSession()
	if err != nil {
		logger.Error("user repository RespondFriendRequest - could not start session", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db session failed", errors.New("database error"))
	}
	defer session.EndSession(ctx)

	res, transactionErr := session.WithTransaction(ctx, callback)
	if transactionErr != nil {
		logger.Error("user repository RespondFriendRequest - transaction failed", transactionErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db transaction failed", errors.New("database error"))
	}

	resUser, ok := res.(domain.User)
	if !ok {
		logger.Error("user repository RespondFriendRequest - assertion failed", fmt.Errorf("could not assert into domain.Agreement: %v", res), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewInternalServerError("db type assertion failed", errors.New("assertion error"))
	}

	logger.Info("user repository RespondFriendRequest finish", context_utils.GetTraceAndClientIds(ctx)...)
	return &resUser, nil
}
