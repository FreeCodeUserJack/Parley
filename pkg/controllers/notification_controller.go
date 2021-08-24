package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/FreeCodeUserJack/Parley/pkg/services"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/http_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/go-chi/chi"
)

func NewNotificationController(notificationServ services.NotificationServiceInterface) NotificationControllerInterface {
	return &notificationController{
		NotificationService: notificationServ,
	}
}

type NotificationControllerInterface interface {
	Routes() chi.Router
	GetUserNotifications(w http.ResponseWriter, r *http.Request)
}

type notificationController struct {
	NotificationService services.NotificationServiceInterface
}

func (n notificationController) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/{userId}", n.GetUserNotifications)

	return router
}

func (n notificationController) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	logger.Info("notification controller GetUserNotifications reading url param", context_utils.GetTraceAndClientIds(r.Context())...)

	userId := chi.URLParam(r, "userId")

	if userId == "" {
		reqErr := rest_errors.NewBadRequestError("userId is missing")
		logger.Error(reqErr.Message(), reqErr, context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, reqErr)
		return
	}

	logger.Info("notification controller GetUserNotifications reading url query param", context_utils.GetTraceAndClientIds(r.Context())...)

	if !strings.Contains(r.URL.String(), "?") || !strings.Contains(r.URL.String(), "=") {
		logger.Error("notification controller CloseAgreement - no query params: "+r.URL.String(), errors.New("missing query"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("missing query params"))
		return
	}

	queryParams := strings.Split(strings.Split(r.URL.String(), "?")[1], "=")

	if len(queryParams) != 2 {
		logger.Error("notification controller CloseAgreement - expected 1 query param: "+r.URL.String(), errors.New("# query param mismatched"), context_utils.GetTraceAndClientIds(r.Context())...)
		http_utils.ResponseError(w, rest_errors.NewBadRequestError("incorrect # of query params"))
		return
	}

	queryKey := queryParams[0]
	queryVal := queryParams[1]

	notifications, serviceErr := n.NotificationService.GetUserNotifications(r.Context(), userId, queryKey, queryVal)
	if serviceErr != nil {
		http_utils.ResponseError(w, serviceErr)
		return
	}

	logger.Info("notification controller GetUserNotifications returning to client", context_utils.GetTraceAndClientIds(r.Context())...)
	http_utils.ResponseJSON(w, http.StatusOK, notifications)
}
