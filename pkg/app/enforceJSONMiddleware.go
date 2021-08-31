package app

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
)

func enforceJSONHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			contentType := r.Header.Get("Content-Type")

			if contentType != "" {
				mt, _, err := mime.ParseMediaType(contentType)
				if err != nil {
					logger.Error("content type can't be parsed", fmt.Errorf("request content type not parseable: %+v", r), context_utils.GetTraceAndClientIds(r.Context())...)
					http.Error(w, "Malformed Content-Type header", http.StatusBadRequest)
					return
				}

				if mt != "application/json" {
					logger.Error("content type is not json", fmt.Errorf("request content type not json: %+v", r), context_utils.GetTraceAndClientIds(r.Context())...)
					http.Error(w, "Content-Type header must be 'application/json'", http.StatusBadRequest)
					return
				}
			} else {
				logger.Error("content type is '' empty string", fmt.Errorf("request content type is empty: %+v", r), context_utils.GetTraceAndClientIds(r.Context())...)
				http.Error(w, "Content-Type Header cannot be empty string", http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
