package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skillswap/internal/transfer"
)

type transferStub struct{ called bool }

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
