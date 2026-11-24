package httpapp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeWaitsForActiveRequestOnShutdown(t *testing.T) {
	checkShutdown(t, false)
}

func TestServeClosesActiveConnectionWhenGraceExpires(t *testing.T) {
	checkShutdown(t, true)
}

func checkShutdown(t *testing.T, force bool) {
	t.Helper()
	watchdog, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	runCtx, signal := context.WithCancel(watchdog)
	defer signal()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, draining, release, canceled := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
			close(canceled)
		}
	})}
	defer server.Close()
	server.RegisterOnShutdown(func() { close(draining) })
	grace := 2 * time.Second
	if force {
		grace = 50 * time.Millisecond
	}
	done := make(chan error, 1)
	go func() { done <- Serve(runCtx, listener, server, grace) }()
	req, err := http.NewRequestWithContext(watchdog, "GET", "http://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		status int
		err    error
	}
	client := &http.Client{}
	defer client.CloseIdleConnections()
	requestDone := make(chan result, 1)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			requestDone <- result{err: err}
			return
		}
		resp.Body.Close()
		requestDone <- result{status: resp.StatusCode}
	}()

	waitForSignal(t, watchdog, entered, "request entry")
	signal()
	waitForSignal(t, watchdog, draining, "shutdown start")
	if !force {
		select {
		case err := <-done:
			t.Fatalf("server exited with active handler: %v", err)
		default:
		}
		close(release)
	}
	select {
	case err := <-done:
		if (force && !errors.Is(err, context.DeadlineExceeded)) || (!force && err != nil) {
			t.Fatalf("shutdown result: %v", err)
		}
	case <-watchdog.Done():
		t.Fatal("server did not stop")
	}
	if force {
		waitForSignal(t, watchdog, canceled, "request context cancellation")
	}
	select {
	case got := <-requestDone:
		if (force && got.err == nil) || (!force && (got.err != nil || got.status != http.StatusNoContent)) {
			t.Fatalf("active request: %+v", got)
		}
	case <-watchdog.Done():
		t.Fatal("client did not finish")
	}
}

func waitForSignal(t *testing.T, watchdog context.Context, signal <-chan struct{}, step string) {
	t.Helper()
	select {
	case <-signal:
	case <-watchdog.Done():
		t.Fatalf("timed out waiting for %s", step)
	}
}
