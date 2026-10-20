package matching

import (
	"reflect"
	"testing"

	"skillswap/domain"
)

func TestConfirmedPartners(t *testing.T) {
	offers := []domain.Offer{
		{ID: "o1", AuthorID: "u1"},
		{ID: "o2", AuthorID: "u2"},
	}

	tests := []struct {
		name      string
		userID    string
		responses []domain.Response
		want      map[string]bool
	}{
		{
			name:      "user is responder",
			userID:    "u3",
			responses: []domain.Response{{ID: "r1", OfferID: "o1", UserID: "u3", Status: domain.ResponseAccepted}},
			want:      map[string]bool{"u1": true},
		},
		{
			name:      "user is offer author",
			userID:    "u1",
			responses: []domain.Response{{ID: "r1", OfferID: "o1", UserID: "u3", Status: domain.ResponseAccepted}},
			want:      map[string]bool{"u3": true},
		},
		{
			name:      "pending is not confirmed",
			userID:    "u1",
			responses: []domain.Response{{ID: "r1", OfferID: "o1", UserID: "u3", Status: domain.ResponsePending}},
			want:      map[string]bool{},
		},
		{
			name:      "rejected is not confirmed",
			userID:    "u1",
			responses: []domain.Response{{ID: "r1", OfferID: "o1", UserID: "u3", Status: domain.ResponseRejected}},
			want:      map[string]bool{},
		},
		{
			name:      "agreement of other users",
			userID:    "u4",
			responses: []domain.Response{{ID: "r1", OfferID: "o2", UserID: "u3", Status: domain.ResponseAccepted}},
			want:      map[string]bool{},
		},
		{
			name:      "unknown offer is skipped",
			userID:    "u3",
			responses: []domain.Response{{ID: "r1", OfferID: "missing", UserID: "u3", Status: domain.ResponseAccepted}},
			want:      map[string]bool{},
		},
		{
			name:      "no responses is initial state",
			userID:    "u1",
			responses: nil,
			want:      map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConfirmedPartners(tt.userID, offers, tt.responses)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmedPartners_DoesNotMutateInput(t *testing.T) {
	offers := []domain.Offer{{ID: "o1", AuthorID: "u1", WantsInReturn: []domain.Skill{{Name: "go"}}}}
	responses := []domain.Response{{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponseAccepted}}
	offersBefore := []domain.Offer{{ID: "o1", AuthorID: "u1", WantsInReturn: []domain.Skill{{Name: "go"}}}}
	responsesBefore := append([]domain.Response(nil), responses...)

	ConfirmedPartners("u1", offers, responses)

	if !reflect.DeepEqual(offers, offersBefore) {
		t.Fatalf("offers changed: %v", offers)
	}
	if !reflect.DeepEqual(responses, responsesBefore) {
		t.Fatalf("responses changed: %v", responses)
	}
}

func TestExcludePartners(t *testing.T) {
	tests := []struct {
		name     string
		ids      []string
		partners map[string]bool
		want     []string
	}{
		{"keeps order", []string{"u2", "u3", "u4"}, map[string]bool{"u3": true}, []string{"u2", "u4"}},
		{"no partners", []string{"u2", "u3"}, map[string]bool{}, []string{"u2", "u3"}},
		{"all excluded is empty non-nil", []string{"u2"}, map[string]bool{"u2": true}, []string{}},
		{"nil ids is empty non-nil", nil, map[string]bool{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExcludePartners(tt.ids, tt.partners)
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
