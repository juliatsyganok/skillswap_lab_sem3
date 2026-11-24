package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"skillswap/internal/httpapp"
)

func TestEngagementRoutesOnAppServer(t *testing.T) {
	cfg, err := httpapp.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server, err := httpapp.NewServer(cfg, logger, func(mux *http.ServeMux, deps httpapp.Dependencies) {
		registerEngagements(mux, deps, logger)
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	steps := []struct {
		user, method, path string
		status             int
		field, want        string
	}{
		{"learner", http.MethodGet, "/v1/engagements/response-1", 200, "status", "pending"},
		{"", http.MethodPut, "/v1/engagements/response-1/acceptance", 400, "code", "invalid_demo_user"},
		{"learner", http.MethodPut, "/v1/engagements/response-1/acceptance", 403, "code", "not_offer_author"},
		{"author", http.MethodPut, "/v1/engagements/response-1/acceptance", 200, "changed", "true"},
		{"author", http.MethodPut, "/v1/engagements/response-1/acceptance", 200, "changed", "false"},
		{"learner", http.MethodGet, "/v1/engagements/response-1", 200, "status", "accepted"},
		{"mentor", http.MethodGet, "/v1/engagements/response-1", 403, "code", "not_participant"},
	}
	for _, s := range steps {
		req, _ := http.NewRequest(s.method, ts.URL+s.path, nil)
		if s.user != "" {
			req.Header.Set("X-Demo-User", s.user)
		}
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != s.status || resp.Header.Get("X-Request-ID") == "" {
			t.Fatalf("%s %s as %q: %d %v %v", s.method, s.path, s.user, resp.StatusCode, body, err)
		}
		if got := fmtValue(body[s.field]); got != s.want {
			t.Fatalf("%s %s as %q: %s=%s, want %s", s.method, s.path, s.user, s.field, got, s.want)
		}
	}
}

func fmtValue(v any) string {
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) >= 2 && s[0] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
