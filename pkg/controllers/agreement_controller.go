package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/dto"
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
	CloseAgreement(w http.ResponseWriter, r *http.Request)
	UpdateAgreement(w http.ResponseWriter, r *http.Request)
	GetAgreement(w http.ResponseWriter, r *http.Request)
	AddUserToAgreement(w http.ResponseWriter, r *http.Request)
	RemoveUserFromAgreement(w http.ResponseWriter, r *http.Request)
	SetDeadline(w http.ResponseWriter, r *http.Request)
	DeleteDeadline(w http.ResponseWriter, r *http.Request)
	SearchAgreements(w http.ResponseWriter, r *http.Request)
	ActionAndNotification(w http.ResponseWriter, r *http.Request)
}

type agreementsResource struct {
	AgreementService services.AgreementServiceInterface
}

func (a agreementsResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", a.NewAgreement)
	router.Get("/search", a.SearchAgreements)
	router.Post("/actionAndNotification", a.ActionAndNotification)

	router.Route("/{agreementId}", func(r chi.Router) {
		r.Delete("/", a.CloseAgreement)
		r.Put("/", a.UpdateAgreement)
		r.Get("/", a.GetAgreement)
		r.Post("/friend/{friendId}", a.AddUserToAgreement)
		r.Delete("/friend/{friendId}", a.RemoveUserFromAgreement)
		r.Put("/deadline", a.SetDeadline)
		r.Delete("/deadline", a.DeleteDeadline)
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

func (a agreementsResource) CloseAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller CloseAgreement reading url id", context_utils.GetTraceAndClientIds(r.Context())...)
	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("agreementId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller CloseAgreement getting query param", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") || !strings.Contains(r.URL.String(), "&") {
		logger.Error("agreement controller CloseAgreement - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	querySplit := strings.Split(strings.Split(r.URL.String(), "?")[1], "&")

	queryParams := []string{
		strings.Split(querySplit[0], "=")[0],
		strings.Split(querySplit[0], "=")[1],
		strings.Split(querySplit[1], "=")[0],
		strings.Split(querySplit[1], "=")[1],
	}

	fmt.Println(queryParams, len(queryParams))

	if len(queryParams) != 4 {
		logger.Error("agreement controller CloseAgreement - expected 2 query params: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	completionKey := queryParams[0]
	completionVal := queryParams[1]
	typeKey := queryParams[2]
	typeVal := queryParams[3]

	uuid, err := a.AgreementService.CloseAgreement(r.Context(), agreementId, completionKey, completionVal, typeKey, typeVal)
	if err != nil {
		http_utils.ResponseError(w, err)
		return
	}

	logger.Info("agreement controller CloseAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "document deleted", Id: uuid})
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
	http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "Added friendId to agremeent", Id: returnedId})
}

func (a agreementsResource) RemoveUserFromAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller RemoveUserFromAgreement about to get agreementId and friendId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	friendId := chi.URLParam(r, "friendId")

	if agreementId == "" || friendId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId or friendId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	returnedId, serviceErr := a.AgreementService.RemoveUserFromAgreement(r.Context(), agreementId, friendId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller RemoveUserFromAgreement about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "Removed friendId from agremeent", Id: returnedId})
}

func (a agreementsResource) SetDeadline(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller SetDeadline about to get agreementId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller SetDeadline about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var reqDeadline domain.Deadline

	jsonErr := json.Unmarshal(reqBytes, &reqDeadline)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	result, serviceErr := a.AgreementService.SetDeadline(r.Context(), agreementId, reqDeadline)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller SetDeadline about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, result)
}

func (a agreementsResource) DeleteDeadline(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller DeleteDeadline about to get agreementId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	returnedAgreement, serviceErr := a.AgreementService.DeleteDeadline(r.Context(), agreementId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller DeleteDeadline about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, returnedAgreement)
}

func (a agreementsResource) ActionAndNotification(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller ActionAndNotification about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var notification domain.Notification
	jsonErr := json.Unmarshal(bodyBytes, &notification)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	notificationRes, serviceErr := a.AgreementService.ActionAndNotification(r.Context(), notification)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller ActionAndNotification about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, notificationRes)
}

// func concatString(input []string) string {
// 	res := ""

// 	for _, str := range input {
// 		res = res + str
// 	}

// 	return res
// }
