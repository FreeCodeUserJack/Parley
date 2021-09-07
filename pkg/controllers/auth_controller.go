package controllers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"

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
	VerifyEmail(http.ResponseWriter, *http.Request)
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
	router.Get("/verifyEmail", o.VerifyEmail)

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

func (o oauthController) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	logger.Info("auth controller VerifyEmail getting url params", context_utils.GetTraceAndClientIds(r.Context())...)

	tmpl := template.Must(template.ParseFiles("../../web/templates/email_verified.html"))

	htmlInputs := struct {
		Error    bool
		UserId   string
		ErrorMsg string
	}{
		Error:    false,
		UserId:   "",
		ErrorMsg: "",
	}

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") || !strings.Contains(r.URL.String(), "&") {
		logger.Error("oauth controller VerifyEmail - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		htmlInputs.Error = true
		htmlInputs.ErrorMsg = "missing query params"
		tmpl.Execute(w, htmlInputs)
		return
	}

	querySplit := strings.Split(strings.Split(r.URL.String(), "?")[1], "&")

	queryParams := []string{
		strings.Split(querySplit[0], "=")[0],
		strings.Split(querySplit[0], "=")[1],
		strings.Split(querySplit[1], "=")[0],
		strings.Split(querySplit[1], "=")[1],
	}

	if len(queryParams) != 4 {
		logger.Error("oauth controller VerifyEmail - expected 2 query params: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		htmlInputs.Error = true
		htmlInputs.ErrorMsg = "incorrect # of query params"
		tmpl.Execute(w, htmlInputs)
		return
	}

	res, serviceErr := o.AuthService.VerifyEmail(r.Context(), queryParams)
	if serviceErr != nil {
		htmlInputs.Error = true
		htmlInputs.ErrorMsg = serviceErr.Message()
		tmpl.Execute(w, htmlInputs)
		return
	}

	htmlInputs.UserId = res

	logger.Info("auth controller VerifyEmail returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	tmpl.Execute(w, htmlInputs)
}
