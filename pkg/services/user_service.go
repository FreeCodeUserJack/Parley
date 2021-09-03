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
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/security_utils"
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
