package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"university-pass/internal/config"
	"university-pass/internal/database"
	"university-pass/internal/handler"
	mw "university-pass/internal/middleware"
	"university-pass/internal/repository"
	"university-pass/internal/service"

	_ "university-pass/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title University Pass API
// @version 1.0
// @description API системы
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey AdminBearer
// @in header
// @name Authorization
// @description JWT из /admin/auth/login
// @securityDefinitions.apikey ScannerKey
// @in header
// @name X-Scanner-Key
// @description Ключ турникета, выдается админом

func main() {
	cfg := config.Load()

	initCtx, cancelInit := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelInit()

	db, err := database.InitDB(initCtx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Pg.Close()
	defer db.Rdb.Close()

	userRepo := repository.NewUserRepository(db)
	guestRepo := repository.NewGuestRepository(db)
	logRepo := repository.NewLogRepository(db.Pg)
	authService := service.NewAuthService(userRepo, guestRepo)
	logService := service.NewLogService(logRepo, db.Rdb)

	accessRepo := repository.NewAccessPointRepository(db)

	authHandler := handler.NewAuthHandler(authService)
	userAdminHandler := handler.NewAdminUserHandler(userRepo)
	guestAdminHandler := handler.NewAdminGuestHandler(guestRepo)
	adminAuthHandler := handler.NewAdminAuthHandler(authService)

	recoverCtx, cancelRecover := context.WithTimeout(context.Background(), 10*time.Second)
	if err := logService.RecoverInFlight(recoverCtx); err != nil {
		log.Printf("failed to recover in-flight logs: %v", err)
	}
	cancelRecover()

	workerCtx, stopWorker := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopWorker()

	var workerWg sync.WaitGroup
	workerWg.Add(1)
	go func() {
		defer workerWg.Done()
		logService.StartLogWorker(workerCtx)
	}()
	log.Println("Log worker started")

	r := chi.NewRouter()

	r.Use(mw.Cors)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "pong"}`))
	})

	r.Route("/admin", func(r chi.Router) {
		r.Post("/auth/login", adminAuthHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(userRepo, "admin"))

			r.Route("/users", func(r chi.Router) {
				r.Post("/", userAdminHandler.Create)
				r.Get("/", userAdminHandler.List)
				r.Patch("/{id}", userAdminHandler.Update)
				r.Delete("/{id}", userAdminHandler.Deactivate)
			})

			r.Route("/guests", func(r chi.Router) {
				r.Post("/", guestAdminHandler.Create)
				r.Get("/", guestAdminHandler.List)
				r.Post("/{id}/revoke", guestAdminHandler.Revoke)
			})
		})
	})
	r.Post("/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(mw.RequireScannerKey(accessRepo))
		r.Post("/scan/verify", authHandler.Verify)
	})

	srv := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		fmt.Println("running: 8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}()

	<-workerCtx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	workerWg.Wait()
	log.Println("all workers stopped, exiting")
}
