package agreement

import (
	"errors"
	"reflect"
	"testing"

	"skillswap/domain"
)

func TestTransitionOffer(t *testing.T) {
	statuses := []domain.OfferStatus{domain.OfferActive, domain.OfferPaused, domain.OfferClosed}
	allowed := map[[2]domain.OfferStatus]bool{
		{domain.OfferActive, domain.OfferPaused}: true,
		{domain.OfferPaused, domain.OfferActive}: true,
		{domain.OfferActive, domain.OfferClosed}: true,
		{domain.OfferPaused, domain.OfferClosed}: true,
	}

	// Перебираем все 3×3 пары: всё, чего нет в allowed, должно быть запрещено.
	for _, cur := range statuses {
		for _, next := range statuses {
			t.Run(string(cur)+"->"+string(next), func(t *testing.T) {
				err := TransitionOffer(cur, next)
				if allowed[[2]domain.OfferStatus{cur, next}] {
					if err != nil {
						t.Fatalf("want nil, got %v", err)
					}
					return
				}
				if !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("want ErrInvalidTransition, got %v", err)
				}
			})
		}
	}
}

func TestTransitionResponse(t *testing.T) {
	statuses := []domain.ResponseStatus{domain.ResponsePending, domain.ResponseAccepted, domain.ResponseRejected}
	allowed := map[[2]domain.ResponseStatus]bool{
		{domain.ResponsePending, domain.ResponseAccepted}: true,
		{domain.ResponsePending, domain.ResponseRejected}: true,
	}

	for _, cur := range statuses {
		for _, next := range statuses {
			t.Run(string(cur)+"->"+string(next), func(t *testing.T) {
				err := TransitionResponse(cur, next)
				if allowed[[2]domain.ResponseStatus{cur, next}] {
					if err != nil {
						t.Fatalf("want nil, got %v", err)
					}
					return
				}
				if !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("want ErrInvalidTransition, got %v", err)
				}
			})
		}
	}
}

func TestTransition_UnknownStatus(t *testing.T) {
	if err := TransitionOffer("deleted", domain.OfferActive); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("offer: want ErrInvalidTransition, got %v", err)
	}
	if err := TransitionResponse(domain.ResponsePending, "maybe"); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("response: want ErrInvalidTransition, got %v", err)
	}
}

func offer(status domain.OfferStatus) domain.Offer {
	return domain.Offer{ID: "o1", AuthorID: "author", Status: status}
}

func response(status domain.ResponseStatus) domain.Response {
	return domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: status}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		actorID  string
		offer    domain.Offer
		resp     domain.Response
		decision domain.ResponseStatus
		want     domain.ResponseStatus
		wantErr  error
	}{
		{"accept pending", "author", offer(domain.OfferActive), response(domain.ResponsePending), domain.ResponseAccepted, domain.ResponseAccepted, nil},
		{"reject pending", "author", offer(domain.OfferActive), response(domain.ResponsePending), domain.ResponseRejected, domain.ResponseRejected, nil},
		{"paused offer still decidable", "author", offer(domain.OfferPaused), response(domain.ResponsePending), domain.ResponseAccepted, domain.ResponseAccepted, nil},
		{"not author", "u2", offer(domain.OfferActive), response(domain.ResponsePending), domain.ResponseAccepted, "", ErrNotOfferAuthor},
		{"closed offer", "author", offer(domain.OfferClosed), response(domain.ResponsePending), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"already accepted", "author", offer(domain.OfferActive), response(domain.ResponseAccepted), domain.ResponseRejected, "", domain.ErrInvalidTransition},
		{"already rejected", "author", offer(domain.OfferActive), response(domain.ResponseRejected), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"decision is pending", "author", offer(domain.OfferActive), response(domain.ResponsePending), domain.ResponsePending, "", domain.ErrInvalidTransition},
		{"other offer", "author", domain.Offer{ID: "o2", AuthorID: "author", Status: domain.OfferActive}, response(domain.ResponsePending), domain.ResponseAccepted, "", ErrOfferMismatch},
		{"repeat accept", "author", offer(domain.OfferActive), response(domain.ResponseAccepted), domain.ResponseAccepted, domain.ResponseAccepted, nil},
		{"repeat accept paused", "author", offer(domain.OfferPaused), response(domain.ResponseAccepted), domain.ResponseAccepted, domain.ResponseAccepted, nil},
		{"repeat by stranger", "stranger", offer(domain.OfferActive), response(domain.ResponseAccepted), domain.ResponseAccepted, "", ErrNotOfferAuthor},
		{"repeat by respondent", "u2", offer(domain.OfferActive), response(domain.ResponseAccepted), domain.ResponseAccepted, "", ErrNotOfferAuthor},
		{"repeat on closed offer", "author", offer(domain.OfferClosed), response(domain.ResponseAccepted), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"repeat for other offer", "author", domain.Offer{ID: "o2", AuthorID: "author", Status: domain.OfferActive}, response(domain.ResponseAccepted), domain.ResponseAccepted, "", ErrOfferMismatch},
		{"repeat rejection remains forbidden", "author", offer(domain.OfferActive), response(domain.ResponseRejected), domain.ResponseRejected, "", domain.ErrInvalidTransition},
		{"unknown offer status", "author", offer("deleted"), response(domain.ResponsePending), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"repeat with unknown offer status", "author", offer("deleted"), response(domain.ResponseAccepted), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"unknown response status", "author", offer(domain.OfferActive), response("unknown"), domain.ResponseAccepted, "", domain.ErrInvalidTransition},
		{"unknown decision", "author", offer(domain.OfferActive), response(domain.ResponsePending), "unknown", "", domain.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.offer.WantsInReturn = []domain.Skill{{Name: "Go", Level: domain.LevelBeginner}}
			beforeOffer := tt.offer
			beforeOffer.WantsInReturn = append([]domain.Skill(nil), tt.offer.WantsInReturn...)
			beforeResponse := tt.resp
			got, err := Decide(tt.actorID, tt.offer, tt.resp, tt.decision)
			if !reflect.DeepEqual(tt.offer, beforeOffer) || tt.resp != beforeResponse {
				t.Fatal("Decide modified its input")
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				if got != (domain.Response{}) {
					t.Fatalf("on error result must be zero, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.want {
				t.Fatalf("status: got %s, want %s", got.Status, tt.want)
			}
			if got.ID != tt.resp.ID || got.OfferID != tt.resp.OfferID || got.UserID != tt.resp.UserID {
				t.Fatalf("only status may change: got %+v, input %+v", got, tt.resp)
			}
			if tt.resp.Status == tt.decision && got != tt.resp {
				t.Fatalf("no-op changed response: got %+v, want %+v", got, tt.resp)
			}
		})
	}
}

func TestDecide_DoesNotModifyInput(t *testing.T) {
	o := offer(domain.OfferActive)
	r := response(domain.ResponsePending)

	if _, err := Decide("author", o, r, domain.ResponseAccepted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Status != domain.ResponsePending {
		t.Fatalf("input response changed: %s", r.Status)
	}
	if o.Status != domain.OfferActive {
		t.Fatalf("input offer changed: %s", o.Status)
	}
}

func TestChangeOfferStatus(t *testing.T) {
	tests := []struct {
		name    string
		actorID string
		offer   domain.Offer
		next    domain.OfferStatus
		wantErr error
	}{
		{"pause active", "author", offer(domain.OfferActive), domain.OfferPaused, nil},
		{"resume paused", "author", offer(domain.OfferPaused), domain.OfferActive, nil},
		{"close active", "author", offer(domain.OfferActive), domain.OfferClosed, nil},
		{"not author", "u2", offer(domain.OfferActive), domain.OfferPaused, ErrNotOfferAuthor},
		{"reopen closed", "author", offer(domain.OfferClosed), domain.OfferActive, domain.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChangeOfferStatus(tt.actorID, tt.offer, tt.next)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.next {
				t.Fatalf("status: got %s, want %s", got.Status, tt.next)
			}
		})
	}
}
