package httpapp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Serve drains active requests when stop is canceled, then closes connections
// if the independent shutdown deadline expires.
func Serve(stop context.Context, listener net.Listener, server *http.Server, grace time.Duration) error {
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	select {
	case err := <-served:
		return serveError(err)
	case <-stop.Done():
	}

	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		return errors.Join(fmt.Errorf("graceful shutdown: %w", shutdownErr), server.Close(), serveError(<-served))
	}
	return serveError(<-served)
}

func serveError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
