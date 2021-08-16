package app

import (
	"fmt"
	"net/http"
	"os"

	"github.com/FreeCodeUserJack/Parley/pkg/controllers"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/services"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var (
	port   string = "8080"
	router *chi.Mux
)

func StartApplication() {

	logger.Info("app initilization started")

	if envPort := os.Getenv("APP_PORT"); envPort != "" {
		port = envPort
	}

	logger.Info(fmt.Sprintf("Starting up on port %s", port))

	router = chi.NewRouter()

	router.Use(contextMiddleware)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)

	// use our custom logger
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger.GetLogger(), NoColor: true})

	router.Use(middleware.DefaultLogger)
	router.Use(middleware.Recoverer)

	router.Get("/api/v1/health", controllers.HealthCheck)

	// setup Users repo/service and mount Users routes
	router.Mount("/api/v1/users", controllers.UsersController.Routes())

	// setup Agreements repo/service and mount Agreements routes
	agreementRepo := repository.NewAgreementRepository()
	agreementService := services.NewAgreementService(agreementRepo)
	router.Mount("/api/v1/agreements", controllers.NewAgreementController(agreementService).Routes())

	logger.Info("app initilization finished")

	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatal("app failed to ListenAndServe", err)
	}
}
