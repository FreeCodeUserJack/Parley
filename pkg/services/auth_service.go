package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/dto"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/security_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

type AuthServiceInterface interface {
	Login(context.Context, dto.LoginRequest) (*domain.TokenDetails, rest_errors.RestError)
	Logout(context.Context, string) (string, rest_errors.RestError)
}

type authService struct {
	AuthRepository  repository.AuthRepositoryInterface
	TokenRepository repository.TokenRepositoryInterface
}

func NewAuthService(authRepo repository.AuthRepositoryInterface, tokenRepo repository.TokenRepositoryInterface) AuthServiceInterface {
	return &authService{
		AuthRepository:  authRepo,
		TokenRepository: tokenRepo,
	}
}

func (a authService) Login(ctx context.Context, loginReq dto.LoginRequest) (*domain.TokenDetails, rest_errors.RestError) {
	logger.Info("auth service Login - start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize Data
	loginReq.Sanitize()

	user, repoErr := a.AuthRepository.Login(ctx, loginReq)
	if repoErr != nil {
		return nil, repoErr
	}

	if user.Status == "deleted" || user.Status == "suspended" {
		logger.Error("user status is not active", fmt.Errorf("user status not active: %+v", user), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("user_account is not active, status is: " + user.Status)
	}

	checkPasswordErr := security_utils.CheckPasswordHash(loginReq.Password, user.Password)
	if !checkPasswordErr {
		return nil, rest_errors.NewBadRequestError("auth failed - credentials did not match")
	}

	token, tokenErr := security_utils.GenerateToken(ctx, user.Id)
	if tokenErr != nil {
		return nil, rest_errors.NewInternalServerError("error trying to generate token", errors.New("token generation error"))
	}

	resToken, saveErr := a.TokenRepository.SaveToken(ctx, *token)
	if saveErr != nil {
		return nil, rest_errors.NewInternalServerError("error trying to save token", errors.New("save token error"))
	}

	logger.Info("auth service Login - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return resToken, nil
}

func (a authService) Logout(ctx context.Context, id string) (string, rest_errors.RestError) {
	logger.Info("auth service Logout - start", context_utils.GetTraceAndClientIds(ctx)...)
	logger.Info("auth service Logout - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return "", nil
}
