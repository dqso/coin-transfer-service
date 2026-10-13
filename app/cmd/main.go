package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dqso/coin-transfer-service/app/internal/repository/postgres"
	transporthttp "github.com/dqso/coin-transfer-service/app/internal/transport/http"
	"github.com/dqso/coin-transfer-service/app/internal/usecase"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("service stopped with error", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// pgx reads the connection settings from the PG* environment variables.
	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	storage := postgres.NewStorage(pool)

	uc := usecase.NewUsecase(storage)

	httpHandler, err := transporthttp.NewHandler(log, uc,
		pool.Ping,
	)
	if err != nil {
		return fmt.Errorf("create http handler: %w", err)
	}

	httpServer := &http.Server{
		Addr:              ":3000",
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	httpServerErr := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", httpServer.Addr))
		httpServerErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-httpServerErr:
		// The server stopped before we asked it to, for example the port is busy.
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-httpServerErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("http server stopped")

	return nil
}
