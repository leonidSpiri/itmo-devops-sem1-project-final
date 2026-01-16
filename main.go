package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	cfg := loadConfigFromEnv()

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	h := &pricesHandler{db: db}
	r.HandleFunc("/api/v0/prices", h.handlePostPrices).Methods(http.MethodPost)
	r.HandleFunc("/api/v0/prices", h.handleGetPrices).Methods(http.MethodGet)

	srv := &http.Server{
		Addr:              cfg.httpAddr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

errCh := make(chan error, 1)

go func() {
    log.Printf("listening on %s", cfg.httpAddr())
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        errCh <- err
    }
}()

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

select {
case sig := <-stop:
    log.Printf("signal: %v", sig)
case err := <-errCh:
    log.Printf("http server failed: %v", err)
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(ctx)

}

type pricesHandler struct {
	db *sql.DB
}
