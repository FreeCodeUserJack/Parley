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

func NewUserController(userService services.UserServiceInterface) UsersControllerInterface {
	return &usersResource{
		UserService: userService,
	}
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

type usersResource struct {
	UserService services.UserServiceInterface
}

func (u usersResource) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/new", u.NewUser)
	router.Get("/search", u.SearchUsers)

	router.Route("/{userId}", func(r chi.Router) {
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
	logger.Info("user controller NewUser reading body", context_utils.GetTraceAndClientIds(r.Context())...)

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		restErr := rest_errors.NewBadRequestError("missing req body")
		logger.Error(restErr.Message(), err, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	var reqUser domain.User

	jsonErr := json.Unmarshal(reqBody, &reqUser)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}

	fmt.Printf("%+v\n", reqUser)

	result, serviceErr := u.UserService.NewUser(r.Context(), reqUser)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller NewUser returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusCreated, result)
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
