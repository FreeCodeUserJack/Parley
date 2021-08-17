package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/services"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/http_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/go-chi/chi"
)

func NewAgreementController(service services.AgreementServiceInterface) AgreementControllerInterface {
	return &agreementsResource{
		AgreementService: service,
	}
}

type AgreementControllerInterface interface {
	Routes() chi.Router
	NewAgreement(w http.ResponseWriter, r *http.Request)
	DeleteAgreement(w http.ResponseWriter, r *http.Request)
	UpdateAgreement(w http.ResponseWriter, r *http.Request)
	GetAgreement(w http.ResponseWriter, r *http.Request)
	AddUserToAgreement(w http.ResponseWriter, r *http.Request)
	RemoveUserFromAgreement(w http.ResponseWriter, r *http.Request)
	AddDeadline(w http.ResponseWriter, r *http.Request)
	DeleteDeadline(w http.ResponseWriter, r *http.Request)
	UpdateDeadline(w http.ResponseWriter, r *http.Request)
}

type agreementsResource struct {
	AgreementService services.AgreementServiceInterface
}

func (a agreementsResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", a.NewAgreement)
	router.Get("/search", a.SearchAgreements)

	router.Route("/{agreementId}", func(r chi.Router) {
		r.Delete("/", a.DeleteAgreement)
		r.Put("/", a.UpdateAgreement)
		r.Get("/", a.GetAgreement)
		r.Post("/friend/{friendId}", a.AddUserToAgreement)
		r.Delete("/friend/{friendId}", a.RemoveUserFromAgreement)
		r.Post("/deadline", a.AddDeadline)
		r.Delete("/deadline", a.DeleteDeadline)
		r.Put("/deadline", a.UpdateDeadline)
	})

	return router
}

func (a agreementsResource) NewAgreement(w http.ResponseWriter, r *http.Request) {

	logger.Info("agreement controller NewAgreement reading body", context_utils.GetTraceAndClientIds(r.Context())...)

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var reqAgreement domain.Agreement

	jsonErr := json.Unmarshal(reqBody, &reqAgreement)
	if jsonErr != nil {
		fmt.Println(jsonErr)
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	result, serviceErr := a.AgreementService.NewAgreement(r.Context(), reqAgreement)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("successfully returned request", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, result)
}

func (a agreementsResource) DeleteAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller DeleteAgreement reading url id", context_utils.GetTraceAndClientIds(r.Context())...)
	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("agreementId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	uuid, err := a.AgreementService.DeleteAgreement(r.Context(), agreementId)
	if err != nil {
		http_utils.ResponseError(w, err)
		return
	}

	logger.Info("agreement controller DeleteAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, domain.Response{Message: "document deleted", Id: uuid})
}

// pass in id via url param, then body containing fields that should be updated
func (a agreementsResource) UpdateAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller UpdateAgreement about to read agreement id", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("agreementId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller UpdateAgreement about to read body", context_utils.GetTraceAndClientIds(r.Context())...)

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var reqAgreement domain.Agreement

	jsonErr := json.Unmarshal(reqBody, &reqAgreement)
	if jsonErr != nil {
		fmt.Println(jsonErr)
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	reqAgreement.Id = agreementId

	res, serviceErr := a.AgreementService.UpdateAgreement(r.Context(), reqAgreement)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller UpdateAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (a agreementsResource) GetAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller GetAgreement about to get req url path id", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("agreementId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	retAgreement, serviceErr := a.AgreementService.GetAgreement(r.Context(), agreementId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller GetAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, retAgreement)
}

func (a agreementsResource) SearchAgreements(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller SearchAgreements about to get req url query params", context_utils.GetTraceAndClientIds(r.Context())...)

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")
	searchKey := queryParams[0]
	searchValue := queryParams[1]

	searchValue, escapeErr := url.QueryUnescape(searchValue)
	if escapeErr != nil {
		logger.Error("agreement controller SearchAgreements failed to unescape query value", escapeErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("please input valid key/value query param pair"))
		return
	}

	result, err := a.AgreementService.SearchAgreements(r.Context(), searchKey, searchValue)
	if err != nil {
		http_utils.ResponseError(w, err)
		return
	}

	logger.Info("agreement controller SearchAgreements about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, result)
}

func (a agreementsResource) AddUserToAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller AddUserToAgreement about to get agreementId and friendId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	friendId := chi.URLParam(r, "friendId")

	// fmt.Println(agreementId, friendId)

	if agreementId == "" || friendId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId or friendId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	returnedId, serviceErr := a.AgreementService.AddUserToAgreement(r.Context(), agreementId, friendId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller AddUserToAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, domain.Response{Message: "Added friendId to agremeent", Id: returnedId})
}

func (a agreementsResource) RemoveUserFromAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) AddDeadline(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) DeleteDeadline(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) UpdateDeadline(w http.ResponseWriter, r *http.Request) {

}
