package controllers

import (
	"encoding/json"
	"errors"
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
	RespondAgreementChange(w http.ResponseWriter, r *http.Request)
	GetAgreementEventResponses(w http.ResponseWriter, r *http.Request)
	InviteUsersToEvent(w http.ResponseWriter, r *http.Request)
	RespondEventInvite(w http.ResponseWriter, r *http.Request)
}

type agreementsResource struct {
	AgreementService services.AgreementServiceInterface
}

func (a agreementsResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", a.NewAgreement)
	router.Get("/search", a.SearchAgreements)
	router.Post("/actionAndNotification", a.ActionAndNotification)
	router.Put("/respondChange", a.RespondAgreementChange)

	router.Route("/{agreementId}", func(r chi.Router) {
		r.Delete("/", a.CloseAgreement)
		r.Put("/", a.UpdateAgreement)
		r.Get("/", a.GetAgreement)
		r.Post("/friend/{friendId}", a.AddUserToAgreement)
		r.Delete("/friend/{friendId}", a.RemoveUserFromAgreement)
		r.Put("/deadline", a.SetDeadline)
		r.Delete("/deadline", a.DeleteDeadline)
		r.Get("/eventResponses", a.GetAgreementEventResponses)
		r.Post("/inviteEventUsers", a.InviteUsersToEvent)
		r.Put("/respondEventInvite", a.RespondEventInvite)
	})

	return router
}

func (a agreementsResource) NewAgreement(w http.ResponseWriter, r *http.Request) {

	logger.Info("agreement controller NewAgreement reading body", context_utils.GetTraceAndClientIds(r.Context())...)

	var reqAgreement domain.Agreement
	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&reqAgreement)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	// fmt.Printf("%+v\n", reqAgreement)

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
	http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "notifications sent to close agreement id: ", Id: uuid})
}

// UpdateAgreement pass in id via url param, then body containing fields that should be updated
func (a agreementsResource) UpdateAgreement(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller UpdateAgreement about to read agreement id", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")

	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("agreementId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller UpdateAgreement about to read query params", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("agreement controller UpdateAgreement - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")

	if len(queryParams) != 2 {
		logger.Error("agreement controller UpdateAgreement - expected 1 query param: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	typeKey := queryParams[0]
	typeVal := queryParams[1]

	logger.Info("agreement controller UpdateAgreement about to read body", context_utils.GetTraceAndClientIds(r.Context())...)

	var reqAgreement domain.Agreement
	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&reqAgreement)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	reqAgreement.Id = agreementId

	res, serviceErr := a.AgreementService.UpdateAgreement(r.Context(), reqAgreement, typeKey, typeVal)
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

	logger.Info("agreement controller SetDeadline about to read query params", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("agreement controller SetDeadline - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")

	if len(queryParams) != 2 {
		logger.Error("agreement controller SetDeadline - expected 1 query param: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	typeKey := queryParams[0]
	typeVal := queryParams[1]

	logger.Info("agreement controller SetDeadline about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var reqDeadline domain.Deadline
	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&reqDeadline)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	result, serviceErr := a.AgreementService.SetDeadline(r.Context(), agreementId, reqDeadline, typeKey, typeVal)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	if result == nil {
		// collaborative return
		http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "sent notifications for deadline change confirmation"})
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

	logger.Info("agreement controller DeleteDeadline about to read query params", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("agreement controller DeleteDeadline - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")

	if len(queryParams) != 2 {
		logger.Error("agreement controller DeleteDeadline - expected 1 query param: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	typeKey := queryParams[0]
	typeVal := queryParams[1]

	returnedAgreement, serviceErr := a.AgreementService.DeleteDeadline(r.Context(), agreementId, typeKey, typeVal)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	if returnedAgreement == nil {
		// collaborative return
		http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "sent notifications for deadline change confirmation"})
		return
	}

	logger.Info("agreement controller DeleteDeadline about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, returnedAgreement)
}

func (a agreementsResource) ActionAndNotification(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller ActionAndNotification about to read query params", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("agreement controller ActionAndNotification - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")

	if len(queryParams) != 2 {
		logger.Error("agreement controller ActionAndNotification - expected 1 query param: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	typeKey := queryParams[0]
	typeVal := queryParams[1]

	logger.Info("agreement controller ActionAndNotification about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var notification domain.Notification
	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&notification)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	notificationRes, serviceErr := a.AgreementService.ActionAndNotification(r.Context(), notification, typeKey, typeVal)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller ActionAndNotification about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, notificationRes)
}

func (a agreementsResource) RespondAgreementChange(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller ActionAndNotification about to read body", context_utils.GetTraceAndClientIds(r.Context())...)

	var notification domain.Notification

	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&notification)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	// fmt.Printf("incoming notification: %+v\n", notification)

	res, serviceErr := a.AgreementService.RespondAgreementChange(r.Context(), notification)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller ActionAndNotification about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (a agreementsResource) GetAgreementEventResponses(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller GetAgreementEventResponses about to get agreementId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	res, serviceErr := a.AgreementService.GetAgreementEventResponses(r.Context(), agreementId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller GetAgreementEventResponses about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (a agreementsResource) InviteUsersToEvent(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller InviteUsersToEvent about to get agreementId", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller InviteUsersToEvent about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var uuids dto.UuidsRequest

	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&uuids)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	res, serviceErr := a.AgreementService.InviteUsersToEvent(r.Context(), agreementId, uuids.Payload)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller InviteUsersToEvent about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, dto.Response{Message: "Invited users for agreement id:", Id: res})
}

func (a agreementsResource) RespondEventInvite(w http.ResponseWriter, r *http.Request) {
	logger.Info("agreement controller RespondEventInvite about to get agreement id", context_utils.GetTraceAndClientIds(r.Context())...)

	agreementId := chi.URLParam(r, "agreementId")
	if agreementId == "" {
		reqErr := rest_errors.NewBadRequestError("missing agreementId")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("agreement controller RespondEventInvite about to read body", context_utils.GetTraceAndClientIds(r.Context())...)

	var eventResponse domain.EventResponse

	defer r.Body.Close()
	jsonErr := json.NewDecoder(r.Body).Decode(&eventResponse)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	res, serviceErr := a.AgreementService.RespondEventInvite(r.Context(), agreementId, eventResponse)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("agreement controller RespondEventInvite about to return to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

// func concatString(input []string) string {
// 	res := ""

// 	for _, str := range input {
// 		res = res + str
// 	}

// 	return res
// }
