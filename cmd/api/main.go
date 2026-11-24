package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"skillswap/internal/agreement"
	"skillswap/internal/engagementapi"
	"skillswap/internal/httpapi"
	"skillswap/internal/httpapp"
	"skillswap/internal/transfer"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := httpapp.LoadConfig()
	if err != nil {
		return err
	}
	stop, releaseSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer releaseSignals()

	var db *sql.DB
	if dsn := os.Getenv("SKILLSWAP_DATABASE_URL"); dsn != "" {
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			return fmt.Errorf("invalid SKILLSWAP_DATABASE_URL")
		}
		defer db.Close()
		ctx, cancel := context.WithTimeout(stop, cfg.CommandTimeout)
		err = db.PingContext(ctx)
		cancel()
		if err != nil {
			return fmt.Errorf("PostgreSQL unavailable: check SKILLSWAP_DATABASE_URL and database status")
		}
	}

	server, err := httpapp.NewServer(cfg, logger, func(mux *http.ServeMux, deps httpapp.Dependencies) {
		registerEngagements(mux, deps, logger)
		if db != nil {
			registerAgreements(mux, db, cfg, logger)
		}
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}
	defer listener.Close()

	logger.Info("server listening", "addr", listener.Addr().String())
	if err := httpapp.Serve(stop, listener, server, cfg.ShutdownTimeout); err != nil {
		return err
	}
	logger.Info("server stopped")
	return nil
}

func registerEngagements(mux *http.ServeMux, deps httpapp.Dependencies, logger *slog.Logger) {
	directory := agreement.LocalOfferDirectory{Offers: deps.Responses}
	handler := &engagementapi.Handler{
		UseCase:   agreement.NewEngagements(deps.Responses, directory, deps.Agreement),
		Actor:     httpapp.ActorID,
		RequestID: httpapp.RequestID,
		Logger:    logger,
	}
	handler.Register(mux)
}

func registerAgreements(mux *http.ServeMux, db *sql.DB, cfg httpapp.Config, logger *slog.Logger) {
	handler := httpapi.NewTransferHandler(transfer.NewService(transfer.NewPostgresRepository(db)), func(ctx context.Context) (string, bool) {
		id := httpapp.ActorID(ctx)
		return id, id != ""
	})
	handler.Timeout = cfg.CommandTimeout
	handler.LogError = func(ctx context.Context, err error) {
		logger.ErrorContext(ctx, "agreement operation failed", "request_id", httpapp.RequestID(ctx), "error", err)
	}
	handler.Register(mux)
}
