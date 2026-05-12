package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/auth"
	"github.com/mxtxy/bx-take-home/backend/internal/db"
	"github.com/mxtxy/bx-take-home/backend/internal/httpapi"
	"github.com/mxtxy/bx-take-home/backend/internal/jobs"
	"github.com/mxtxy/bx-take-home/backend/internal/notifications"
	"github.com/mxtxy/bx-take-home/backend/internal/quotes"
	"github.com/mxtxy/bx-take-home/backend/internal/scheduling"
	"github.com/mxtxy/bx-take-home/backend/internal/technicians"
	"github.com/mxtxy/bx-take-home/backend/internal/ws"
)

const (
	defaultDevelopmentJWTSecret = "dev_only_change_me"
	minProductionJWTSecretLen   = 32
)

func main() {
	database, err := db.Open(db.DefaultDSN())
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	secret, err := sessionSecretFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	cookieName := os.Getenv("COOKIE_NAME")
	if cookieName == "" {
		cookieName = "brix_session"
	}
	cookieSecure, _ := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	hub := ws.NewHub()
	router := httpapi.NewRouter(httpapi.Dependencies{
		Auth: auth.NewService(database, auth.Options{
			Secret:     secret,
			SessionTTL: 8 * time.Hour,
		}),
		Quotes:        quotes.NewService(database),
		Technicians:   technicians.NewService(database),
		Jobs:          jobs.NewQueryService(database),
		Scheduling:    scheduling.NewService(database, scheduling.Options{}),
		Notifications: notifications.NewService(database, notifications.Options{}),
		Hub:           hub,
		CookieName:    cookieName,
		CookieSecure:  cookieSecure,
		AllowedOrigin: os.Getenv("CORS_ALLOWED_ORIGIN"),
	})

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("backend listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func sessionSecretFromEnv() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	developmentMode := appEnv == "development" || appEnv == "test" || appEnv == "local"
	if secret == "" {
		if developmentMode {
			return defaultDevelopmentJWTSecret, nil
		}
		return "", errors.New("JWT_SECRET is required outside development and test")
	}
	if !developmentMode && secret == defaultDevelopmentJWTSecret {
		return "", errors.New("JWT_SECRET must not use the development default outside development and test")
	}
	if !developmentMode && len(secret) < minProductionJWTSecretLen {
		return "", errors.New("JWT_SECRET must be at least 32 characters outside development and test")
	}
	return secret, nil
}
