package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"university-pass/internal/database"
	"university-pass/internal/handler"
	mw "university-pass/internal/middleware"
	"university-pass/internal/repository"
	"university-pass/internal/service"

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
// @description Ключ конкретной турникета, выдаётся админом

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.InitDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Pg.Close()

	if db.Rdb != nil {
		defer db.Rdb.Close()
	}

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

	workerCtx := context.Background()
	go logService.StartLogWorker(workerCtx)
	log.Println("Log worker started")

	r := chi.NewRouter()

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
			r.Use(mw.RequireRole("admin"))

			r.Route("/users", func(r chi.Router) {
				r.Post("/", userAdminHandler.Create)
				r.Get("/", userAdminHandler.List)
				r.Patch("/{id}", userAdminHandler.Update)
				r.Delete("/{id}", userAdminHandler.Deactivate) // soft
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
	fmt.Println("running: 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Println(err)
	}
}
