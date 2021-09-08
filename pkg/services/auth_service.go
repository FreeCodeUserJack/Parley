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
	VerifyEmail(context.Context, []string) (string, rest_errors.RestError)
	VerifyPhone(context.Context, []string) (string, rest_errors.RestError)
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

func (a authService) VerifyEmail(ctx context.Context, queryParams []string) (string, rest_errors.RestError) {
	logger.Info("auth service VerifyEmail - start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize query params
	queryParams = domain.SanitizeStringSlice(queryParams)

	if queryParams[0] != "userId" || queryParams[2] != "authId" {
		logger.Error("query params keys are invalid", fmt.Errorf("query params: %v", queryParams), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("invalid query params key")
	}

	// Get Current User
	user, getErr := a.AuthRepository.GetUser(ctx, queryParams[1])
	if getErr != nil {
		logger.Error("auth service VerifyEmail - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", getErr
	}

	// Validate
	if user.AccountVerified == "email" || user.AccountVerified == "phone" {
		logger.Error("already verified account", fmt.Errorf("user instance: %+v", user), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("already verified acount " + user.AccountVerified)
	}

	logger.Info("auth service VerifyEmail - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AuthRepository.VerifyEmail(ctx, queryParams[1], queryParams[3])
}

func (a authService) VerifyPhone(ctx context.Context, queryParams []string) (string, rest_errors.RestError) {
	logger.Info("auth service VerifyPhone - start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize query params
	queryParams = domain.SanitizeStringSlice(queryParams)

	if queryParams[0] != "userId" || queryParams[2] != "otp" {
		logger.Error("query params keys are invalid", fmt.Errorf("query params: %v", queryParams), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("invalid query params key")
	}

	// Get Current User
	user, getErr := a.AuthRepository.GetUser(ctx, queryParams[1])
	if getErr != nil {
		logger.Error("auth service VerifyPhone - could not get saved user", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", getErr
	}

	// Validate
	if user.AccountVerified == "email" || user.AccountVerified == "phone" {
		logger.Error("already verified account", fmt.Errorf("user instance: %+v", user), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("already verified acount " + user.AccountVerified)
	}

	// Get Current Account Verification
	accVerification, getErr := a.AuthRepository.GetAccVerification(ctx, queryParams[1])
	if getErr != nil {
		logger.Error("auth service VerifyPhone - could not get saved account verification", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", getErr
	}

	// Validate
	if accVerification.OTP != queryParams[3] {
		logger.Error("mismatched otp", fmt.Errorf("OTP %s does not match account verification otp %+v", queryParams[3], accVerification), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("otp does not match")
	}

	logger.Info("auth service VerifyPhone - finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AuthRepository.VerifyPhone(ctx, queryParams[1], accVerification.Id)
}
