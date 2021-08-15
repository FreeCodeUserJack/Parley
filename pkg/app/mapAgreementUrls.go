package app

import (
	"net/http"

	"github.com/go-chi/chi"
)

type agreementsResource struct{}

func (a agreementsResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", a.NewAgreement)

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