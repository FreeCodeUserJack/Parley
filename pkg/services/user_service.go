package services

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/email_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/security_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/sms_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/google/uuid"
)

type UserServiceInterface interface {
	NewUser(context.Context, domain.User) (*domain.User, rest_errors.RestError)
	GetUser(context.Context, string) (*domain.User, rest_errors.RestError)
	UpdateUser(context.Context, string, domain.User) (*domain.User, rest_errors.RestError)
	DeleteUser(context.Context, string) (*domain.User, rest_errors.RestError)
	GetFriends(context.Context, string, []string) ([]domain.User, rest_errors.RestError)
	RemoveFriend(context.Context, string, string) (*domain.User, rest_errors.RestError)
	SearchUsers(context.Context, []string) ([]domain.User, rest_errors.RestError)
	GetAgreements(context.Context, string) ([]domain.Agreement, rest_errors.RestError)
	AddFriend(context.Context, string, string, string) (*domain.User, rest_errors.RestError)
	RespondFriendRequest(context.Context, string, string, string) (*domain.User, rest_errors.RestError)
}

type userService struct {
	UserRepository repository.UserRepositoryInterface
}

func NewUserService(userRepo repository.UserRepositoryInterface) UserServiceInterface {
	return userService{
		UserRepository: userRepo,
	}
}

func (u userService) NewUser(ctx context.Context, user domain.User) (*domain.User, rest_errors.RestError) {
	logger.Info("user service NewUser start", context_utils.GetTraceAndClientIds(ctx)...)

	if !user.Validate() {
		logger.Error("user service NewUser - user validation failed", errors.New("users request is not valid"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("req failed validation")
	}

	// Sanitize Data
	user.Sanitize()

	// Hash the password
	hash := security_utils.GetHash(ctx, user.Password)
	if hash == "" {
		return nil, rest_errors.NewInternalServerError("error when trying to get hash password", errors.New("hash error"))
	}

	user.Password = hash

	user.Id = uuid.NewString()

	currTime := time.Now().UTC()
	user.CreateDateTime = currTime
	user.LastUpdateDateTime = currTime

	// Initialize slices
	user.Agreements = []string{}
	user.InvitedAgreements = []string{}
	user.RequestedAgreements = []string{}
	user.PendingAgreementRemovals = []string{}
	user.PendingLeaveAgreements = []string{}
	user.PendingFriendRequests = []string{}
	user.Friends = []string{}
	user.SentFriendRequests = []string{}

	logger.Info("user service NewUser calling UserRepository NewUser", context_utils.GetTraceAndClientIds(ctx)...)

	// Check if need to verify email
	if user.Email != "" {
		user.AccountVerified = "false"
		accountVerification, err := email_utils.SendEmail(ctx, user.Email, user)
		if err != nil {
			return nil, err
		}
		return u.UserRepository.NewUserVerifyAccount(ctx, user, *accountVerification)
	}

	// Check if need to verify phone number
	if user.Phone != "" {
		otp, err := sms_utils.GenerateOTP(6)
		if err != nil {
			logger.Error("could not generate OTP - random 6 digit code", err, context_utils.GetTraceAndClientIds(ctx)...)
		}

		accountVerification := domain.AccountVerification{
			Id:             uuid.NewString(),
			CreateDateTime: time.Now().UTC(),
			UserId:         user.Id,
			Phone:          user.Phone,
			Type:           "phone",
			OTP:            otp,
			Status:         "new",
		}

		retUser, repoErr := u.UserRepository.NewUserVerifyAccount(ctx, user, accountVerification)
		if repoErr != nil {
			return nil, repoErr
		}

		sms_utils.SendSMS(ctx, user.Phone, otp)
		return retUser, nil
	}

	logger.Info("user service NewUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.NewUser(ctx, user)
}

func (u userService) GetUser(ctx context.Context, userId string) (*domain.User, rest_errors.RestError) {
	logger.Info("user service GetUser start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize userId
	userId = strings.TrimSpace(html.EscapeString(userId))

	logger.Info("user service GetUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.GetUser(ctx, userId)
}

func (u userService) UpdateUser(ctx context.Context, userId string, user domain.User) (*domain.User, rest_errors.RestError) {
	logger.Info("user service UpdateUser start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize Data
	userId = strings.TrimSpace(html.EscapeString(userId))
	user.Sanitize()

	// Get Saved User
	savedUser, getErr := u.UserRepository.GetUser(ctx, userId)
	if getErr != nil {
		logger.Error("user service UpdateUser - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	if user.FirstName != "" {
		savedUser.FirstName = user.FirstName
	}
	if user.LastName != "" {
		savedUser.LastName = user.LastName
	}
	if !user.DOB.IsZero() {
		savedUser.DOB = user.DOB
	}
	// if user.Role != "" {
	// 	savedUser.Role = user.Role
	// }
	if user.Status != "" {
		savedUser.Status = user.Status
	}
	if user.Public != "" {
		savedUser.Public = user.Public
	}
	if user.Phone != "" {
		savedUser.Phone = user.Phone
	}

	logger.Info("user service UpdateUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.UpdateUser(ctx, userId, *savedUser)
}

func (u userService) DeleteUser(ctx context.Context, userId string) (*domain.User, rest_errors.RestError) {
	logger.Info("user service DeleteUser start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize User Id
	userId = strings.TrimSpace(html.EscapeString(userId))

	logger.Info("user service DeleteUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.DeleteUser(ctx, userId)
}

func (u userService) GetFriends(ctx context.Context, userId string, uuids []string) ([]domain.User, rest_errors.RestError) {
	logger.Info("user service GetFriends start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize User Id
	userId = strings.TrimSpace(html.EscapeString(userId))

	logger.Info("user service GetFriends finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.GetFriends(ctx, userId, uuids)
}

func (u userService) RemoveFriend(ctx context.Context, userId string, friendId string) (*domain.User, rest_errors.RestError) {
	logger.Info("user service RemoveFriend start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize ids
	userId = strings.TrimSpace(html.EscapeString(userId))
	friendId = strings.TrimSpace(html.EscapeString(friendId))

	// Get Saved User
	user, getErr := u.UserRepository.GetUser(ctx, userId)
	if getErr != nil {
		logger.Error("user service RemoveFriend - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	// Validate
	if !isInSlice(friendId, user.Friends) {
		logger.Error("friendId is not a friend of userId", fmt.Errorf("friendId: %s, user instance: %+v", friendId, user), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError(fmt.Sprintf("friendId: %s, is not a friend of userId: %s", friendId, userId))
	}

	notification := domain.Notification{
		Id:               uuid.NewString(),
		Title:            fmt.Sprintf("%s %s removed you from their friend list", user.FirstName, user.LastName),
		Message:          "",
		CreateDateTime:   time.Now().UTC(),
		Status:           "new",
		UserId:           friendId,
		ContactId:        userId,
		ContactFirstName: user.FirstName,
		Type:             "notifyUpdate",
		Action:           "update",
	}

	// Remove friendId from User and userId from Friend - then send notification to Friend

	logger.Info("user service RemoveFriend finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.RemoveFriend(ctx, userId, friendId, notification)
}

func (u userService) SearchUsers(ctx context.Context, queries []string) ([]domain.User, rest_errors.RestError) {
	logger.Info("user service SearchUsers start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize Data
	queries = domain.SanitizeStringSlice(queries)

	var input [][]string

	for _, query := range queries {
		buf := strings.Split(query, "=")
		if len(buf) != 2 {
			logger.Error("query is malformed", fmt.Errorf("contains malformed query: %v", queries), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewBadRequestError("query is malformed")
		}
		input = append(input, buf)
	}

	// Validate Queries
	for _, q := range input {
		if q[0] != "id" && q[0] != "first_name" && q[0] != "last_name" && q[0] != "email" && q[0] != "phone" {
			logger.Error("query key is not accepted", fmt.Errorf("queries contain invalid key(s): %v", queries), context_utils.GetTraceAndClientIds(ctx)...)
			return nil, rest_errors.NewBadRequestError("query contains invalid keys for search")
		}
	}

	logger.Info("user service SearchUsers finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.SearchUsers(ctx, input)
}

func (u userService) GetAgreements(ctx context.Context, userId string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("user service GetAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize ids
	userId = strings.TrimSpace(html.EscapeString(userId))

	logger.Info("user service GetAgreements finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.GetAgreements(ctx, userId)
}

func (u userService) AddFriend(ctx context.Context, userId, friendId, message string) (*domain.User, rest_errors.RestError) {
	logger.Info("user service AddFriend start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize inputs
	userId = strings.TrimSpace(html.EscapeString(userId))
	friendId = strings.TrimSpace(html.EscapeString(friendId))
	message = strings.TrimSpace(html.EscapeString(message))

	// Get Saved User
	user, getErr := u.UserRepository.GetUser(ctx, userId)
	if getErr != nil {
		logger.Error("user service AddFriend - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	// Validate
	if isInSlice(friendId, user.Friends) {
		logger.Error("user is already a friend", fmt.Errorf("friendId: %s already in user.Friends: %+v", friendId, user), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("friend is already on friend list")
	}
	if isInSlice(friendId, user.SentFriendRequests) {
		logger.Error("friend request already sent", fmt.Errorf("friendId: %s already in user.SentFriendRequests: %+v", friendId, user), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("already sent friend request")
	}

	notification := domain.Notification{
		Id:               uuid.NewString(),
		Title:            fmt.Sprintf("Friend request from %s %s!", user.FirstName, user.LastName),
		Message:          message,
		CreateDateTime:   time.Now().UTC(),
		Status:           "new",
		UserId:           friendId,
		ContactId:        userId,
		ContactFirstName: user.FirstName,
		Type:             "notifyInvite",
		Action:           "requires_response",
	}

	logger.Info("user service AddFriend finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.AddFriend(ctx, userId, friendId, notification)
}

func (u userService) RespondFriendRequest(ctx context.Context, userId, friendId, message string) (*domain.User, rest_errors.RestError) {
	logger.Info("user service RespondFriendRequest start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize inputs
	userId = strings.TrimSpace(html.EscapeString(userId))
	friendId = strings.TrimSpace(html.EscapeString(friendId))
	message = strings.TrimSpace(html.EscapeString(message))

	if message != "accepted" && message != "declined" {
		logger.Error("invalid message value", fmt.Errorf("message value invalid: %s", message), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("message value is invalid")
	}

	// Get Saved User
	user, getErr := u.UserRepository.GetUser(ctx, userId)
	if getErr != nil {
		logger.Error("user service AddFriend - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	// Validate
	if !isInSlice(friendId, user.PendingFriendRequests) {
		logger.Error("user service AddFriend - friend never sent a request to this user", fmt.Errorf("friendId: %s not in user.PendingFriendRequest: %+v", friendId, user), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("your friend never sent you an invite!")
	}

	var typeStr string
	if message == "accepted" {
		typeStr = "notifyAcceptFriendInvite"
	} else { // "declined"
		typeStr = "notifyDeclineFriendInvite"
	}

	notification := domain.Notification{
		Id:               uuid.NewString(),
		Title:            fmt.Sprintf("%s %s %s your friend request", user.FirstName, user.LastName, message),
		Message:          "",
		CreateDateTime:   time.Now().UTC(),
		Status:           "new",
		UserId:           friendId,
		ContactId:        userId,
		ContactFirstName: user.FirstName,
		Type:             typeStr,
		Action:           "notify",
	}

	logger.Info("user service RespondFriendRequest finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.RespondFriendRequest(ctx, userId, friendId, notification)
}
