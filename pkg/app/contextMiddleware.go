package app

import (
	"context"
	"net/http"

	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/google/uuid"
)

func contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), context_utils.ClientId("clientId"), r.Header.Get("clientId"))
		uuid, err := uuid.NewUUID()

		if err != nil {
			logger.Error("could not generate uuid for request traceId, setting uuid to zero", err)
		}

		ctx = context.WithValue(ctx, context_utils.TraceId("traceId"), uuid)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
