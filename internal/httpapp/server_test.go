package httpapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"skillswap/domain"
)

func TestServerWiresSharedStateAndRequestMetadata(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	srv, err := NewServer(cfg, logger, func(mux *http.ServeMux, deps Dependencies) {
		mux.HandleFunc("PUT /v1/test/{id}", func(w http.ResponseWriter, r *http.Request) {
			actor := ActorID(r.Context())
			if actor != "author" || RequestID(r.Context()) != w.Header().Get("X-Request-ID") {
				t.Errorf("missing request metadata: actor=%q id=%q", actor, RequestID(r.Context()))
			}
			if _, ok := r.Context().Deadline(); !ok {
				t.Error("request context has no deadline")
			}
			resp, changed, err := deps.Agreement.AcceptContext(r.Context(), actor, r.PathValue("id"))
			if err != nil || !changed || resp.Status != domain.ResponseAccepted {
				t.Errorf("accept: %+v, %t, %v", resp, changed, err)
			}
			stored, err := deps.Responses.GetResponse(resp.ID)
			if err != nil || stored != resp {
				t.Errorf("shared state: %+v, %v", stored, err)
			}
			user, err := deps.Profiles.Get("learner")
			if err != nil {
				t.Fatal(err)
			}
			offers, err := deps.Discovery.FindRelevant(user)
			if err != nil || len(offers) != 4 {
				t.Errorf("discovery after accept: %+v, %v", offers, err)
			}
			for _, offer := range offers {
				if offer.ID == "offer-1" {
					t.Error("accepted offer still in discovery")
				}
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("PUT", "/v1/test/response-1?secret=hidden", nil)
	req.Header.Set("X-Demo-User", "author")
	req.Header.Set("X-Request-ID", "untrusted")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	id := rec.Header().Get("X-Request-ID")
	if rec.Code != http.StatusNoContent || id == "" || id == "untrusted" {
		t.Fatalf("response: %d, id=%q", rec.Code, id)
	}
	var event struct {
		ID     string `json:"request_id"`
		Method string `json:"method"`
		Route  string `json:"route"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event.ID != id || event.Method != "PUT" || event.Route != "PUT /v1/test/{id}" || event.Status != 204 {
		t.Fatalf("unexpected log: %s", logs.String())
	}
	if bytes.Contains(logs.Bytes(), []byte("hidden")) || bytes.Contains(logs.Bytes(), []byte("response-1")) {
		t.Fatalf("log contains raw request data: %s", logs.String())
	}
}

func TestDemoIdentityRejectsInvalidHeaderBeforeHandler(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(cfg, slog.New(slog.DiscardHandler), func(mux *http.ServeMux, _ Dependencies) {
		mux.HandleFunc("GET /v1/test", func(http.ResponseWriter, *http.Request) {
			t.Error("invalid identity reached handler")
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, values := range [][]string{nil, {""}, {"bad user"}, {"author,learner"}, {"author", "learner"}} {
		req := httptest.NewRequest("GET", "/v1/test", nil)
		for _, v := range values {
			req.Header.Add("X-Demo-User", v)
		}
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || rec.Header().Get("X-Request-ID") == "" {
			t.Errorf("identity %q: status=%d headers=%v", values, rec.Code, rec.Header())
		}
		var problem struct{ Code string }
		if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil || problem.Code != "invalid_demo_user" {
			t.Errorf("identity %q: body=%q error=%v", values, rec.Body.String(), err)
		}
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("health: status=%d headers=%v", rec.Code, rec.Header())
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct{ key, value string }{
		{"SKILLSWAP_ADDR", "localhost"},
		{"SKILLSWAP_UPSTREAM_URL", "file:///tmp/local"},
		{"SKILLSWAP_UPSTREAM_URL", "http://127.0.0.1:"},
		{"SKILLSWAP_UPSTREAM_URL", "http://127.0.0.1:0"},
		{"SKILLSWAP_UPSTREAM_URL", "http://127.0.0.1:65536"},
		{"SKILLSWAP_COMMAND_TIMEOUT", "0s"},
		{"SKILLSWAP_SHUTDOWN_TIMEOUT", "five"},
		{"SKILLSWAP_UPSTREAM_MAX_BODY", "-1"},
		{"SKILLSWAP_MAX_HEADER_BYTES", ""},
		{"SKILLSWAP_MAX_HEADER_BYTES", "9223372036854775807"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("accepted %s=%q", tt.key, tt.value)
			}
		})
	}
}

func TestLoadConfigAppliesOverridesToServerAndClient(t *testing.T) {
	t.Setenv("SKILLSWAP_ADDR", "127.0.0.1:9090")
	t.Setenv("SKILLSWAP_UPSTREAM_URL", "http://127.0.0.1:9091/profiles")
	t.Setenv("SKILLSWAP_UPSTREAM_TIMEOUT", "7s")
	t.Setenv("SKILLSWAP_READ_HEADER_TIMEOUT", "1s")
	t.Setenv("SKILLSWAP_IDLE_TIMEOUT", "9s")
	t.Setenv("SKILLSWAP_MAX_HEADER_BYTES", "8192")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(cfg, slog.New(slog.DiscardHandler), func(_ *http.ServeMux, deps Dependencies) {
		if deps.Config.UpstreamURL != "http://127.0.0.1:9091/profiles" || deps.HTTPClient.Timeout.String() != "7s" {
			t.Errorf("unexpected client configuration: %+v, %v", deps.Config, deps.HTTPClient.Timeout)
		}
		if _, err := deps.Responses.GetResponse("absent"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("missing response: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if srv.Addr != "127.0.0.1:9090" || srv.ReadHeaderTimeout.String() != "1s" || srv.IdleTimeout.String() != "9s" || srv.MaxHeaderBytes != 8192 {
		t.Fatalf("unexpected server configuration: %+v", srv)
	}
}
