package controllers

import (
	"net/http"

	"github.com/go-chi/chi"
)

var (
	UsersController UsersControllerInterface
)

func init() {
	UsersController = &usersResource{}
}

type UsersControllerInterface interface {
	Routes() chi.Router
	NewUser(w http.ResponseWriter, r *http.Request)
	SearchUsers(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	UpdateUser(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
	GetFriends(w http.ResponseWriter, r *http.Request)
	GetAgreements(w http.ResponseWriter, r *http.Request)
}

type usersResource struct{}

func (u usersResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", u.NewUser)
	router.Get("/search", u.SearchUsers)

	router.Route("/{userId}", func (r chi.Router) {
		r.Get("/", u.GetUser)
		r.Put("/", u.UpdateUser)
		r.Delete("/", u.DeleteUser)
		r.Post("/friend/{friendId}", u.AddFriend)
		r.Delete("/friend/{friendId}", u.DeleteFriend)
		r.Get("/friends", u.GetFriends)
		r.Get("/agreements", u.GetAgreements)
	})

	return router
}

func (u usersResource) NewUser(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) SearchUsers(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) GetUser(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) UpdateUser(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) DeleteUser(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) AddFriend(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) DeleteFriend(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) GetFriends(w http.ResponseWriter, r *http.Request) {

}

func (u usersResource) GetAgreements(w http.ResponseWriter, r *http.Request) {

}