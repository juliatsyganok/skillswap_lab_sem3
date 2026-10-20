package discovery

import (
	"errors"
	"reflect"
	"testing"

	"skillswap/domain"
)

type fakeResponses struct {
	responses []domain.Response
	err       error
	called    bool
}

func (f *fakeResponses) GetResponsesByUser(string) ([]domain.Response, error) {
	f.called = true
	return f.responses, f.err
}

func offerIDs(offers []domain.Offer) []string {
	ids := make([]string, 0, len(offers))
	for _, o := range offers {
		ids = append(ids, o.ID)
	}
	return ids
}

func TestExcludeConfirmed(t *testing.T) {
	offers := []domain.Offer{{ID: "o1"}, {ID: "o2"}, {ID: "o3"}}

	tests := []struct {
		name      string
		responses []domain.Response
		want      []string
	}{
		{"accepted offer excluded", []domain.Response{{OfferID: "o2", UserID: "u1", Status: domain.ResponseAccepted}}, []string{"o1", "o3"}},
		{"pending keeps offer", []domain.Response{{OfferID: "o2", UserID: "u1", Status: domain.ResponsePending}}, []string{"o1", "o2", "o3"}},
		{"rejected keeps offer", []domain.Response{{OfferID: "o2", UserID: "u1", Status: domain.ResponseRejected}}, []string{"o1", "o2", "o3"}},
		{"other user's agreement keeps offer", []domain.Response{{OfferID: "o2", UserID: "u9", Status: domain.ResponseAccepted}}, []string{"o1", "o2", "o3"}},
		{"no responses is initial state", nil, []string{"o1", "o2", "o3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := offerIDs(ExcludeConfirmed(offers, "u1", tt.responses))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeConfirmed_AllExcludedIsEmptyNonNil(t *testing.T) {
	got := ExcludeConfirmed(
		[]domain.Offer{{ID: "o1"}},
		"u1",
		[]domain.Response{{OfferID: "o1", UserID: "u1", Status: domain.ResponseAccepted}},
	)

	if got == nil || len(got) != 0 {
		t.Fatalf("want empty non-nil slice, got %#v", got)
	}
}

func TestExcludeConfirmed_DoesNotMutateInput(t *testing.T) {
	offers := []domain.Offer{{ID: "o1", WantsInReturn: []domain.Skill{{Name: "go"}}}, {ID: "o2"}}
	responses := []domain.Response{{OfferID: "o1", UserID: "u1", Status: domain.ResponseAccepted}}
	offersBefore := []domain.Offer{{ID: "o1", WantsInReturn: []domain.Skill{{Name: "go"}}}, {ID: "o2"}}
	responsesBefore := append([]domain.Response(nil), responses...)

	ExcludeConfirmed(offers, "u1", responses)

	if !reflect.DeepEqual(offers, offersBefore) {
		t.Fatalf("offers changed: %v", offers)
	}
	if !reflect.DeepEqual(responses, responsesBefore) {
		t.Fatalf("responses changed: %v", responses)
	}
}

func TestService_FindRelevant_ExcludesConfirmed(t *testing.T) {
	user := domain.User{ID: "u1", Needs: []domain.Skill{{Name: "Go"}}, Format: domain.FormatOnline}
	offers := &fakeReader{offers: []domain.Offer{
		{ID: "o1", AuthorID: "u2", Skill: domain.Skill{Name: "Go"}, Format: domain.FormatOnline, Status: domain.OfferActive},
		{ID: "o2", AuthorID: "u3", Skill: domain.Skill{Name: "Go"}, Format: domain.FormatOnline, Status: domain.OfferActive},
	}}
	responses := &fakeResponses{responses: []domain.Response{{OfferID: "o1", UserID: "u1", Status: domain.ResponseAccepted}}}

	got, err := NewService(offers, responses).FindRelevant(user)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids := offerIDs(got); !reflect.DeepEqual(ids, []string{"o2"}) {
		t.Fatalf("got %v, want [o2]", ids)
	}
}

func TestService_FindRelevant_OffersErrorStopsReads(t *testing.T) {
	responses := &fakeResponses{}

	got, err := NewService(&fakeReader{err: domain.ErrStorageUnavailable}, responses).FindRelevant(domain.User{ID: "u1"})

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("want ErrStorageUnavailable, got %v", err)
	}
	if got != nil {
		t.Fatalf("on error result must be nil, got %v", got)
	}
	if responses.called {
		t.Fatal("responses must not be read after offers failed")
	}
}

func TestService_FindRelevant_ResponsesError(t *testing.T) {
	responses := &fakeResponses{err: domain.ErrStorageUnavailable}

	got, err := NewService(&fakeReader{}, responses).FindRelevant(domain.User{ID: "u1"})

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("want ErrStorageUnavailable, got %v", err)
	}
	if got != nil {
		t.Fatalf("on error result must be nil, got %v", got)
	}
}
