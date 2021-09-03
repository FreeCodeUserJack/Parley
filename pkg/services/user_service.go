package services

import (
	"context"
	"errors"
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

func (u userService)	UpdateUser(ctx context.Context, userId string, user domain.User) (*domain.User, rest_errors.RestError) {
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