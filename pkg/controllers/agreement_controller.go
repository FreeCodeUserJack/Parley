package controllers

import (
	"net/http"

	"github.com/FreeCodeUserJack/Parley/pkg/services"
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

type agreementsResource struct{
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