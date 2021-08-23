package services

import (
	"context"
	"errors"
	"fmt"
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
	user.Notifications = []string{}
	user.PendingFriendRequests = []string{}

	fmt.Println(user.DOB)

	logger.Info("user service NewUser finish", context_utils.GetTraceAndClientIds(ctx)...)
	return u.UserRepository.NewUser(ctx, user)
}
