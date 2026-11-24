package httpapp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

type requestKey int

const (
	requestIDKey requestKey = iota
	actorIDKey
)

var demoID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func ActorID(ctx context.Context) string {
	value, _ := ctx.Value(actorIDKey).(string)
	return value
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{code, message})
}

func observeRequests(mux *http.ServeMux, logger *slog.Logger, budget time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := rand.Text()
		w.Header().Set("X-Request-ID", id)
		writer := &statusWriter{ResponseWriter: w}
		defer func() {
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request", "request_id", id, "method", r.Method,
				"route", route, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}()

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		if !(r.URL.Path == "/healthz" && (r.Method == http.MethodGet || r.Method == http.MethodHead)) {
			values := r.Header.Values("X-Demo-User")
			if len(values) != 1 || !demoID.MatchString(values[0]) {
				_, r.Pattern = mux.Handler(r)
				WriteError(writer, http.StatusBadRequest, "invalid_demo_user", "Укажите корректный X-Demo-User")
				return
			}
			ctx = context.WithValue(ctx, actorIDKey, values[0])
		}
		ctx, cancel := context.WithTimeout(ctx, budget)
		defer cancel()
		r = r.WithContext(ctx)
		mux.ServeHTTP(writer, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	if status >= 200 || status == http.StatusSwitchingProtocols {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
