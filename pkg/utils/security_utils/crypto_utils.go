package security_utils

import (
	"context"
	"fmt"
	"os"

	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"golang.org/x/crypto/bcrypt"
)

const (
	envPasswordSalt  = "PASSWORD_SALT"
	envAccessSecret  = "ACCESS_SECRET"
	envRefreshSecret = "REFRESH_SECRET"
)

var (
	// passwordSalt = "jacklagall"
	accessSecret  = "ynufiemgjzepla"
	refreshSecret = "zuqnamdkeioglp"
)

func init() {
	// if salt := os.Getenv(envPasswordSalt); salt != "" {
	// 	passwordSalt = salt
	// }

	if secret := os.Getenv(envAccessSecret); secret != "" {
		accessSecret = secret
	}

	if refresh := os.Getenv(envRefreshSecret); refresh != "" {
		refreshSecret = refresh
	}
}

func GetHash(ctx context.Context, pwd string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("", err, context_utils.GetTraceAndClientIds(ctx)...)
		return ""
	}

	return string(hash)
}

func CheckPasswordHash(password, passwordHash string) bool {
	// // test
	// hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	// err2 := bcrypt.CompareHashAndPassword([]byte("$2a$10$a9bRBbXhbGVCLDzBDLQh0eHtyTO/NZJfqWroNgkfUajTTvR7hhT0u"), []byte("pass123"))
	// fmt.Println(string(hash), err2)

	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

	fmt.Println(passwordHash, password, err)
	fmt.Printf("%q\n", passwordHash)

	if err != nil {
		logger.Error("CheckPasswordHash error", err)
		return false
	}

	return true
}
