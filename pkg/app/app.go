package app

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/FreeCodeUserJack/Parley/pkg/controllers"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var (
	port string = "8080"
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

	// use our custome logger
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger.GetLogger()})

	router.Use(middleware.DefaultLogger)
	router.Use(middleware.Recoverer)

	router.Get("/api/v1/health", controllers.HealthCheck)

	// mount Users and Agreements routes
	router.Mount("/api/v1/users", usersResource{}.Routes())
	router.Mount("/api/v1/agreements", agreementsResource{}.Routes())

	log.Fatal(http.ListenAndServe(":" + port, router))

	logger.Info("app initilization finished")
}