package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"university-pass/internal/database"
	"university-pass/internal/handler"
	"university-pass/internal/repository"
	"university-pass/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

	authHandler := handler.NewAuthHandler(authService, accessRepo)

	workerCtx := context.Background()
	go logService.StartLogWorker(workerCtx)
	log.Println("Log worker started")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "pong"}`))
	})

	r.Post("/auth/login", authHandler.Login)
	r.Post("/scan/verify-user", authHandler.VerifyUser)
	r.Post("/scan/verify-guest", authHandler.VerifyGuest)
	fmt.Println("running: 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Println(err)
	}
}
