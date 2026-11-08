package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"skillswap/internal/agreement"
	"skillswap/internal/engagementapi"
	"skillswap/internal/httpapp"
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
	server, err := httpapp.NewServer(cfg, logger, func(mux *http.ServeMux, deps httpapp.Dependencies) {
		registerEngagements(mux, deps, logger)
	})
	if err != nil {
		return err
	}

	stop, releaseSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer releaseSignals()
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
