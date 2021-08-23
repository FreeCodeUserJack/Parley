package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/FreeCodeUserJack/Parley/pkg/dto"
	"github.com/FreeCodeUserJack/Parley/pkg/services"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/http_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/go-chi/chi"
)

type OauthControllerInterface interface {
	Routes() chi.Router
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
}

type oauthController struct {
	AuthService services.AuthServiceInterface
}

func NewAuthController(authService services.AuthServiceInterface) OauthControllerInterface {
	return &oauthController{
		AuthService: authService,
	}
}

func (o oauthController) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/login", o.Login)
	router.Get("/logout", o.Logout)

	return router
}

func (o oauthController) Login(w http.ResponseWriter, r *http.Request) {
	logger.Info("auth controller Login getting body", context_utils.GetTraceAndClientIds(r.Context())...)

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), err, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var loginReq dto.LoginRequest

	jsonErr := json.Unmarshal(reqBody, &loginReq)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	token, serviceErr := o.AuthService.Login(r.Context(), loginReq)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	res := dto.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}

	logger.Info("auth controller Login returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (o oauthController) Logout(w http.ResponseWriter, r *http.Request) {
	logger.Info("auth controller Logout getting url param", context_utils.GetTraceAndClientIds(r.Context())...)

	logger.Info("auth controller Logout returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
}
