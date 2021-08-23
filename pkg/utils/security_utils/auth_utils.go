package security_utils

import (
	"context"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func GenerateToken(ctx context.Context, userId string) (*domain.TokenDetails, error) {
	td := &domain.TokenDetails{}
	td.UserId = userId

	td.AtExpires = time.Now().UTC().Add(time.Minute * 20).Unix()
	td.AccessUuid = uuid.NewString()
	td.RtExpires = time.Now().UTC().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUuid = uuid.NewString()

	// create access token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUuid
	atClaims["user_id"] = userId
	atClaims["exp"] = td.AtExpires

	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)

	var err error
	td.AccessToken, err = at.SignedString([]byte(accessSecret))
	if err != nil {
		logger.Error("crypto_utils GenerateToken - could not generate access token", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, err
	}

	// create refresh token
	rtClaims := jwt.MapClaims{}
	rtClaims["authorized"] = true
	rtClaims["access_uuid"] = td.RefreshUuid
	rtClaims["user_id"] = userId
	rtClaims["exp"] = td.RtExpires

	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)

	td.RefreshToken, err = rt.SignedString([]byte(refreshSecret))
	if err != nil {
		logger.Error("crypto_utils GenerateToken - could not generate refresh token", err, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, err
	}

	return td, nil
}
