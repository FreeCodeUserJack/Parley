package crypto_utils

import (
	"context"
	"os"

	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"golang.org/x/crypto/bcrypt"
)

const (
	envPasswordSalt = "PASSWORD_SALT"
)

var (
	passwordSalt = "jacklagall"
)

func init() {
	if salt := os.Getenv(envPasswordSalt); salt != "" {
		passwordSalt = salt
	}
}

func GetHash(ctx context.Context, pwd string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		logger.Error("", err, context_utils.GetTraceAndClientIds(ctx)...)
		return ""
	}

	return string(hash)
}
