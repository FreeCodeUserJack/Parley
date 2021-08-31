package controllers

import (
	"fmt"
	"net/http"

	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	logger.Error("requested route path not registered to router", fmt.Errorf("requested path: %s not in router", r.URL), context_utils.GetTraceAndClientIds(r.Context())...)
	http.Error(w, "Resource Not Found", http.StatusNotFound)
}
