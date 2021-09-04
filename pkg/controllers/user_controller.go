package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
	AddFriend(w http.ResponseWriter, r *http.Request)
	RespondFriendRequest(w http.ResponseWriter, r *http.Request)
	RemoveFriend(w http.ResponseWriter, r *http.Request)
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
		r.Delete("/friend/{friendId}", u.RemoveFriend)
		r.Get("/friends", u.GetFriends)
		r.Get("/agreements", u.GetAgreements)
		r.Put("/respond/{friendId}", u.RespondFriendRequest)
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
	logger.Info("user controller SearchUsers about to get query params", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("user controller SearchUsers - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queries := strings.Split(strings.Split(r.URL.String(), "?")[1], "&")

	res, serviceErr := u.UserService.SearchUsers(r.Context(), queries)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller SearchUsers returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) GetUser(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller GetUser about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	user, serviceErr := u.UserService.GetUser(r.Context(), userId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller GetUser returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, user)
}

func (u usersResource) UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller UpdateUser about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller UpdateUser about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var user domain.User
	jsonErr := json.NewDecoder(r.Body).Decode(&user)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	res, serviceErr := u.UserService.UpdateUser(r.Context(), userId, user)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller UpdateUser returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) DeleteUser(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller DeleteUser about to get userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	res, serviceErr := u.UserService.DeleteUser(r.Context(), userId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller DeleteUser returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) AddFriend(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller AddFriend about to get userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller AddFriend about to get friendId", context_utils.GetTraceAndClientIds(r.Context())...)

	friendId := chi.URLParam(r, "friendId")
	if friendId == "" {
		reqErr := rest_errors.NewBadRequestError("friendId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller AddFriend about to get body", context_utils.GetTraceAndClientIds(r.Context())...)
	var message dto.Message
	jsonErr := json.NewDecoder(r.Body).Decode(&message)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	res, serviceErr := u.UserService.AddFriend(r.Context(), userId, friendId, message.Text)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller AddFriend returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller DeleteFriend about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller DeleteFriend about to get path param friendId", context_utils.GetTraceAndClientIds(r.Context())...)

	friendId := chi.URLParam(r, "friendId")
	if friendId == "" {
		reqErr := rest_errors.NewBadRequestError("friendId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	res, serviceErr := u.UserService.RemoveFriend(r.Context(), userId, friendId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller DeleteFriend returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) GetFriends(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller GetFriends about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller GetFriends about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var uuids dto.UuidsRequest
	jsonErr := json.NewDecoder(r.Body).Decode(&uuids)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	res, serviceErr := u.UserService.GetFriends(r.Context(), userId, uuids.Payload)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller GetFriends returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) GetAgreements(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller GetAgreements about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	res, serviceErr := u.UserService.GetAgreements(r.Context(), userId)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller GetAgreements returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}

func (u usersResource) RespondFriendRequest(w http.ResponseWriter, r *http.Request) {
	logger.Info("user controller RespondFriendRequest about to get path param userId", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller RespondFriendRequest about to get path param friendId", context_utils.GetTraceAndClientIds(r.Context())...)

	friendId := chi.URLParam(r, "friendId")
	if friendId == "" {
		reqErr := rest_errors.NewBadRequestError("friendId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("user controller RespondFriendRequest about to get body", context_utils.GetTraceAndClientIds(r.Context())...)

	var message dto.Message
	jsonErr := json.NewDecoder(r.Body).Decode(&message)
	if jsonErr != nil {
		restErr := rest_errors.NewBadRequestError("invalid json body")
		logger.Error(restErr.Message(), restErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, restErr)
		return
	}
	defer r.Body.Close()

	res, serviceErr := u.UserService.RespondFriendRequest(r.Context(), userId, friendId, message.Text)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("user controller RespondFriendRequest returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, res)
}
