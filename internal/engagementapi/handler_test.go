package engagementapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"skillswap/domain"
	"skillswap/internal/agreement"
	"skillswap/internal/storage"
)

type actorKey struct{}

func actorFromContext(ctx context.Context) string {
	actor, _ := ctx.Value(actorKey{}).(string)
	return actor
}

type directoryStub struct {
	err    error
	info   *agreement.OfferInfo
	calls  int
	before func(ctx context.Context) error
}

func (d *directoryStub) OfferInfo(ctx context.Context, offerID string) (agreement.OfferInfo, error) {
	d.calls++
	if d.before != nil {
		if err := d.before(ctx); err != nil {
			return agreement.OfferInfo{}, err
		}
	}
	if d.err != nil {
		return agreement.OfferInfo{}, d.err
	}
	if d.info != nil {
		return *d.info, nil
	}
	return agreement.OfferInfo{ID: offerID, Active: true}, nil
}

type fixture struct {
	store     *storage.InMemoryAgreementStore
	directory *directoryStub
	mux       *http.ServeMux
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := storage.NewInMemoryAgreementStore()
	for _, o := range []domain.Offer{
		{ID: "offer-1", AuthorID: "author", Skill: domain.Skill{Name: "go"}, Status: domain.OfferActive},
		{ID: "offer-2", AuthorID: "author", Skill: domain.Skill{Name: "go"}, Status: domain.OfferClosed},
	} {
		if err := store.SaveOffer(o); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []domain.Response{
		{ID: "response-1", OfferID: "offer-1", UserID: "learner", Status: domain.ResponsePending},
		{ID: "response-2", OfferID: "offer-1", UserID: "other", Status: domain.ResponseRejected},
		{ID: "response-3", OfferID: "offer-2", UserID: "learner", Status: domain.ResponsePending},
	} {
		if err := store.SaveResponse(r); err != nil {
			t.Fatal(err)
		}
	}
	directory := &directoryStub{}
	return &fixture{store: store, directory: directory, mux: newMux(agreement.NewEngagements(store, directory, agreement.NewService(store)))}
}

func newMux(uc UseCase) *http.ServeMux {
	mux := http.NewServeMux()
	(&Handler{UseCase: uc, Actor: actorFromContext}).Register(mux)
	return mux
}

func call(t *testing.T, mux *http.ServeMux, actor, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	}
	req = req.WithContext(context.WithValue(req.Context(), actorKey{}, actor))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: %q", ct)
	}
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func (f *fixture) status(t *testing.T, id string) domain.ResponseStatus {
	t.Helper()
	r, err := f.store.GetResponse(id)
	if err != nil {
		t.Fatal(err)
	}
	return r.Status
}

func TestAcceptRepeatAndRead(t *testing.T) {
	f := newFixture(t)

	before := call(t, f.mux, "learner", http.MethodGet, "/v1/engagements/response-1", "")
	if before.Code != http.StatusOK || decode[EngagementDTO](t, before).Status != "pending" {
		t.Fatalf("initial read: %d %s", before.Code, before.Body)
	}

	first := call(t, f.mux, "author", http.MethodPut, "/v1/engagements/response-1/acceptance", "")
	want := AcceptanceDTO{EngagementDTO: EngagementDTO{ID: "response-1", OfferID: "offer-1", AuthorID: "author", LearnerID: "learner", Status: "accepted"}, Changed: true}
	if first.Code != http.StatusOK || decode[AcceptanceDTO](t, first) != want {
		t.Fatalf("accept: %d %s", first.Code, first.Body)
	}

	repeat := call(t, f.mux, "author", http.MethodPut, "/v1/engagements/response-1/acceptance", "")
	want.Changed = false
	if repeat.Code != http.StatusOK || decode[AcceptanceDTO](t, repeat) != want {
		t.Fatalf("repeat: %d %s", repeat.Code, repeat.Body)
	}

	for _, actor := range []string{"author", "learner"} {
		read := call(t, f.mux, actor, http.MethodGet, "/v1/engagements/response-1", "")
		if read.Code != http.StatusOK || decode[EngagementDTO](t, read) != want.EngagementDTO {
			t.Fatalf("read by %s: %d %s", actor, read.Code, read.Body)
		}
	}
	all, err := f.store.ListResponses()
	if err != nil || len(all) != 3 {
		t.Fatalf("responses: %d, %v", len(all), err)
	}
	if f.directory.calls != 2 {
		t.Fatalf("directory calls: got %d, want 2 (только команды)", f.directory.calls)
	}
}

func TestDomainRefusalsKeepState(t *testing.T) {
	tests := []struct {
		name, actor, method, target string
		status                      int
		code                        string
	}{
		{"learner cannot accept", "learner", http.MethodPut, "/v1/engagements/response-1/acceptance", 403, "not_offer_author"},
		{"stranger cannot read", "stranger", http.MethodGet, "/v1/engagements/response-1", 403, "not_participant"},
		{"unknown engagement", "author", http.MethodPut, "/v1/engagements/response-9/acceptance", 404, "engagement_not_found"},
		{"unknown engagement read", "author", http.MethodGet, "/v1/engagements/response-9", 404, "engagement_not_found"},
		{"rejected cannot be accepted", "author", http.MethodPut, "/v1/engagements/response-2/acceptance", 409, "invalid_transition"},
		{"closed offer", "author", http.MethodPut, "/v1/engagements/response-3/acceptance", 409, "invalid_transition"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			rec := call(t, f.mux, tt.actor, tt.method, tt.target, "")
			if rec.Code != tt.status || decode[errorDTO](t, rec).Code != tt.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tt.status, tt.code)
			}
			if f.status(t, "response-1") != domain.ResponsePending || f.status(t, "response-2") != domain.ResponseRejected || f.status(t, "response-3") != domain.ResponsePending {
				t.Fatal("refused request changed state")
			}
		})
	}
}

func TestDependencyFailuresLeaveNoRecord(t *testing.T) {
	tests := []struct {
		name      string
		directory directoryStub
		status    int
		code      string
	}{
		{"missing", directoryStub{err: fmt.Errorf("upstream 404: %w", domain.ErrNotFound)}, 404, "offer_not_found"},
		{"error", directoryStub{err: fmt.Errorf("upstream 500: %w", agreement.ErrDirectoryFailed)}, 502, "dependency_failed"},
		{"large", directoryStub{err: fmt.Errorf("body over limit: %w", agreement.ErrDirectoryFailed)}, 502, "dependency_failed"},
		{"wrong id", directoryStub{info: &agreement.OfferInfo{ID: "offer-X", Active: true}}, 502, "dependency_failed"},
		{"inactive", directoryStub{info: &agreement.OfferInfo{ID: "offer-1"}}, 409, "offer_unavailable"},
		{"slow", directoryStub{before: func(ctx context.Context) error {
			<-ctx.Done()
			return fmt.Errorf("do request: %w", ctx.Err())
		}}, 504, "dependency_timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.NewInMemoryAgreementStore()
			_ = store.SaveOffer(domain.Offer{ID: "offer-1", AuthorID: "author", Status: domain.OfferActive})
			_ = store.SaveResponse(domain.Response{ID: "response-1", OfferID: "offer-1", UserID: "learner", Status: domain.ResponsePending})
			directory := tt.directory
			mux := newMux(agreement.NewEngagements(store, &directory, agreement.NewService(store)))

			req := httptest.NewRequest(http.MethodPut, "/v1/engagements/response-1/acceptance", nil)
			ctx, cancel := context.WithTimeout(context.WithValue(req.Context(), actorKey{}, "author"), 20*time.Millisecond)
			defer cancel()
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req.WithContext(ctx))

			if rec.Code != tt.status || decode[errorDTO](t, rec).Code != tt.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tt.status, tt.code)
			}
			if got, _ := store.GetResponse("response-1"); got.Status != domain.ResponsePending {
				t.Fatalf("status after failure: %s", got.Status)
			}
		})
	}
}

type useCaseSpy struct {
	calls int
	err   error
}

func (s *useCaseSpy) Accept(context.Context, string, string) (agreement.Engagement, bool, error) {
	s.calls++
	return agreement.Engagement{}, false, s.err
}

func (s *useCaseSpy) Get(context.Context, string, string) (agreement.Engagement, error) {
	s.calls++
	return agreement.Engagement{}, s.err
}

func TestInvalidInputDoesNotCallUseCase(t *testing.T) {
	long := strings.Repeat("a", 65)
	tests := []struct {
		name, actor, method, target, body, code string
	}{
		{"no actor", "", http.MethodPut, "/v1/engagements/response-1/acceptance", "", "invalid_demo_user"},
		{"bad actor", "bad user", http.MethodGet, "/v1/engagements/response-1", "", "invalid_demo_user"},
		{"bad id chars", "author", http.MethodPut, "/v1/engagements/resp%20onse/acceptance", "", "invalid_id"},
		{"id starts with dash", "author", http.MethodGet, "/v1/engagements/-x", "", "invalid_id"},
		{"id too long", "author", http.MethodGet, "/v1/engagements/" + long, "", "invalid_id"},
		{"query on put", "author", http.MethodPut, "/v1/engagements/response-1/acceptance?status=accepted", "", "query_not_allowed"},
		{"query on get", "author", http.MethodGet, "/v1/engagements/response-1?x=1", "", "query_not_allowed"},
		{"empty query", "author", http.MethodGet, "/v1/engagements/response-1?", "", "query_not_allowed"},
		{"body on put", "author", http.MethodPut, "/v1/engagements/response-1/acceptance", `{"actor":"author"}`, "body_not_allowed"},
		{"body on get", "author", http.MethodGet, "/v1/engagements/response-1", "x", "body_not_allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &useCaseSpy{}
			rec := call(t, newMux(spy), tt.actor, tt.method, tt.target, tt.body)
			if rec.Code != http.StatusBadRequest || decode[errorDTO](t, rec).Code != tt.code {
				t.Fatalf("got %d %s, want 400 %s", rec.Code, rec.Body, tt.code)
			}
			if spy.calls != 0 {
				t.Fatalf("use case called %d times", spy.calls)
			}
		})
	}
}

func TestChunkedBodyIsRejectedWithoutLength(t *testing.T) {
	spy := &useCaseSpy{}
	req := httptest.NewRequest(http.MethodPut, "/v1/engagements/response-1/acceptance", strings.NewReader("{}"))
	req.ContentLength = -1
	req = req.WithContext(context.WithValue(req.Context(), actorKey{}, "author"))
	rec := httptest.NewRecorder()
	newMux(spy).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || spy.calls != 0 {
		t.Fatalf("got %d, calls %d", rec.Code, spy.calls)
	}
}

func TestWrappedErrorsKeepStatusAndCode(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrNotFound, 404, "engagement_not_found"},
		{fmt.Errorf("%w: %w", agreement.ErrOfferMissing, domain.ErrNotFound), 404, "offer_not_found"},
		{agreement.ErrNotOfferAuthor, 403, "not_offer_author"},
		{agreement.ErrNotParticipant, 403, "not_participant"},
		{domain.ErrInvalidTransition, 409, "invalid_transition"},
		{agreement.ErrOfferUnavailable, 409, "offer_unavailable"},
		{agreement.ErrDirectoryFailed, 502, "dependency_failed"},
		{context.DeadlineExceeded, 504, "dependency_timeout"},
		{fmt.Errorf("%w: %w", agreement.ErrDirectoryFailed, context.DeadlineExceeded), 504, "dependency_timeout"},
		{context.Canceled, 503, "request_canceled"},
		{domain.ErrStorageUnavailable, 500, "internal_error"},
		{errors.New("boom"), 500, "internal_error"},
	}
	for _, tt := range tests {
		wrapped := fmt.Errorf("layer two: %w", fmt.Errorf("layer one: %w", tt.err))
		t.Run(tt.code+"/"+tt.err.Error(), func(t *testing.T) {
			for _, req := range []struct{ method, target string }{
				{http.MethodPut, "/v1/engagements/response-1/acceptance"},
				{http.MethodGet, "/v1/engagements/response-1"},
			} {
				rec := call(t, newMux(&useCaseSpy{err: wrapped}), "author", req.method, req.target, "")
				body := decode[errorDTO](t, rec)
				if rec.Code != tt.status || body.Code != tt.code {
					t.Fatalf("%s: got %d %+v, want %d %s", req.method, rec.Code, body, tt.status, tt.code)
				}
				if strings.Contains(body.Message, "layer") {
					t.Fatalf("internal error leaked: %q", body.Message)
				}
			}
		})
	}
}
