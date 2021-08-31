package app

import (
	"net/http"

	"github.com/go-chi/chi"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/api/v1/login" || r.URL.Path == "/api/v1/health" || !router.Match(chi.NewRouteContext(), r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "auth middleware error", http.StatusBadRequest)
		// next.ServeHTTP(w, r)
	})
}
