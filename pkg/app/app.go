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
	// educational purposes
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("StartApplication panic caught: %v\n", r)
		}
	}()

	logger.Info("app initialization started")

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

	// Enforce JSON Middleware
	router.Use(enforceJSONHandler)

	// Auth Middleware
	router.Use(authMiddleware)

	// Health Check
	router.Get("/api/v1/health", controllers.HealthCheck)

	router.NotFound(controllers.NotFoundHandler)
	router.HandleFunc("/favicon.ico", controllers.FaviconHandler)

	// Auth
	router.Mount("/api/v1/auth", controllers.NewAuthController(services.NewAuthService(repository.NewAuthRepository(), repository.NewTokenRepository())).Routes())

	// setup Users repo/service and mount Users routes
	router.Mount("/api/v1/users", controllers.NewUserController(services.NewUserService(repository.NewUserRepository())).Routes())

	// setup Agreements repo/service and mount Agreements routes
	agreementRepo := repository.NewAgreementRepository()
	agreementArchiveRepo := repository.NewAgreementArchiveRepository()
	notificationRepo := repository.NewNotificationRepository()
	agreementService := services.NewAgreementService(agreementRepo, agreementArchiveRepo, notificationRepo)
	router.Mount("/api/v1/agreements", controllers.NewAgreementController(agreementService).Routes())

	// setup Notification repo/service and mount Notifications routes
	router.Mount("/api/v1/notifications", controllers.NewNotificationController(services.NewNotificationService(repository.NewNotificationRepository())).Routes())

	logger.Info("app initialization finished")

	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatal("app failed to ListenAndServe", err)
	}

	// cert, _ := tls.LoadX509KeyPair("../../conf/server/pizzazzapp.crt", "../../conf/server/pizzazzapp.key")

	// s := &http.Server{
	// 	Addr:    ":8080",
	// 	Handler: router,
	// 	TLSConfig: &tls.Config{
	// 		Certificates: []tls.Certificate{cert},
	// 	},
	// }

	// if err := s.ListenAndServeTLS("", ""); err != nil {
	// 	logger.Fatal("app failed to ListenAndServeTLS", err)
	// }
}
