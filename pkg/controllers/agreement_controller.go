package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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
		logger.Error(serviceErr.Message(), serviceErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("successfully returned request", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, result)
}

func (a agreementsResource) DeleteAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) UpdateAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) GetAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) SearchAgreements(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) AddUserToAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) RemoveUserFromAgreement(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) AddDeadline(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) DeleteDeadline(w http.ResponseWriter, r *http.Request) {

}

func (a agreementsResource) UpdateDeadline(w http.ResponseWriter, r *http.Request) {

}
