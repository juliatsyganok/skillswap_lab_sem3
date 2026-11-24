package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skillswap/internal/transfer"
)

type transferStub struct {
	called     bool
	completion transfer.CompletionCommand
}

func (s *transferStub) Complete(_ context.Context, c transfer.CompletionCommand) (transfer.CompletionResult, error) {
	s.completion = c
	return transfer.CompletionResult{Changed: true}, nil
}

func (s *transferStub) Transfer(_ context.Context, c transfer.Command) (transfer.Result, error) {
	s.called = true
	return transfer.Result{AgreementID: c.AgreementID, Revision: c.ExpectedRevision + 1, CommandID: c.CommandID}, nil
}
func (s *transferStub) GetAgreement(_ context.Context, id, _ string) (transfer.Result, error) {
	return transfer.Result{AgreementID: id}, nil
}
func (s *transferStub) GetCommand(_ context.Context, id, _ string) (transfer.Result, error) {
	return transfer.Result{CommandID: id}, nil
}
func TestTransferHTTP(t *testing.T) {
	s := &transferStub{}
	h := NewTransferHandler(s, func(context.Context) (string, bool) { return "u1", true })
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest("POST", "/v1/agreements/a1/transfers", strings.NewReader(`{"command_id":"c1","proposal_id":"p1","expected_revision":1}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 || !s.called {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompletionHTTP(t *testing.T) {
	s := &transferStub{}
	h := NewTransferHandler(s, func(context.Context) (string, bool) { return "u1", true })
	mux := http.NewServeMux()
	h.Register(mux)

	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"summary":"done","rating":5} {}`, 400},
		{`{"summary":"done","rating":5,"actor_id":"u2"}`, 400},
		{`{"summary":"done","rating":5}`, 200},
	} {
		req := httptest.NewRequest("POST", "/v1/agreements/a1/completion", strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("code=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
		}
	}
	if want := (transfer.CompletionCommand{ActorID: "u1", AgreementID: "a1", Summary: "done", Rating: 5}); s.completion != want {
		t.Fatalf("command = %+v, want %+v", s.completion, want)
	}
}
