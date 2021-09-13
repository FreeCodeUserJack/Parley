package app

import (
	"fmt"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/security_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt"

	"github.com/casbin/casbin"
	"github.com/go-chi/chi"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/users/new" || r.URL.Path == "/api/v1/auth/verifyEmail" || !router.Match(chi.NewRouteContext(), r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		err := TokenValid(r)
		if err != nil {
			http.Error(w, "auth middleware error: Authentication Failed", http.StatusUnauthorized)
			return
		}

		e := casbin.NewEnforcer("./auth_models.conf", "./auth_policy.csv")
		err = VerifyAuthorization(e, r)
		if err != nil {
			http.Error(w, "auth middleware error: Authorization Failed", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func TokenValid(r *http.Request) error {
	token, err := VerifyToken(r)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
}

func VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := ExtractToken(r)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv(security_utils.EnvAccessSecret)), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	//normally Authorization the_token_xxx
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func VerifyAuthorization(e *casbin.Enforcer, r *http.Request) error {
	// get roles. TODO: need to read from DB
	role := ""
	if role == "" {
		role = "anonymous"
	}
	// if it's a member, check if the user still exists
	//if role == "member" {
	//	uid, err := session.GetInt(r, "userID")
	//	if err != nil {
	//		writeError(http.StatusInternalServerError, "ERROR", w, err)
	//		return
	//	}
	//	exists := users.Exists(uid)
	//	if !exists {
	//		writeError(http.StatusForbidden, "FORBIDDEN", w, errors.New("user does not exist"))
	//		return
	//	}
	//}

	// casbin enforce
	res, err := e.EnforceSafe(role, r.URL.Path, r.Method)

	if err != nil {
		return err
	}
	if res {
		logger.Info("has the authorization")
		return nil
	} else {
		logger.Info("FORBIDDEN")
		return err
	}

}
