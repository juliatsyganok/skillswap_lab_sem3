package response

import (
	"errors"
	"testing"

	"skillswap/domain"
)



type failingReader struct{}

func (f *failingReader) GetOffer(offerID string) (domain.Offer, error) {
	return domain.Offer{}, domain.ErrStorageUnavailable
}

func (f *failingReader) GetResponsesByUser(userID string) ([]domain.Response, error) {
	return nil, domain.ErrStorageUnavailable
}


type fakeReader struct {
	offer     domain.Offer
	responses []domain.Response
}

func (f *fakeReader) GetOffer(offerID string) (domain.Offer, error) {
	if offerID != f.offer.ID {
		return domain.Offer{}, domain.ErrNotFound
	}
	return f.offer, nil
}

func (f *fakeReader) GetResponsesByUser(userID string) ([]domain.Response, error) {
	return f.responses, nil
}



func TestCheckCanRespond_Allowed(t *testing.T) {
	offer := domain.Offer{ID: "offer-1", AuthorID: "author-1", Status: domain.OfferActive}
	svc := NewService(&fakeReader{offer: offer})

	ok, err := svc.CheckCanRespond(domain.User{ID: "user-1"}, "offer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected true, got false")
	}
}



func TestCheckCanRespond_OfferNotActive(t *testing.T) {
	offer := domain.Offer{ID: "offer-1", AuthorID: "author-1", Status: domain.OfferClosed}
	svc := NewService(&fakeReader{offer: offer})

	ok, err := svc.CheckCanRespond(domain.User{ID: "user-1"}, "offer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected false for closed offer")
	}
}

func TestCheckCanRespond_OwnOffer(t *testing.T) {
	offer := domain.Offer{ID: "offer-1", AuthorID: "user-1", Status: domain.OfferActive}
	svc := NewService(&fakeReader{offer: offer})

	ok, err := svc.CheckCanRespond(domain.User{ID: "user-1"}, "offer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected false when user is the author")
	}
}

func TestCheckCanRespond_AlreadyResponded(t *testing.T) {
	offer := domain.Offer{ID: "offer-1", AuthorID: "author-1", Status: domain.OfferActive}
	existing := []domain.Response{{OfferID: "offer-1", UserID: "user-1"}}
	svc := NewService(&fakeReader{offer: offer, responses: existing})

	ok, err := svc.CheckCanRespond(domain.User{ID: "user-1"}, "offer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected false when user already responded")
	}
}


func TestCheckCanRespond_StorageUnavailable(t *testing.T) {
	svc := NewService(&failingReader{})

	_, err := svc.CheckCanRespond(domain.User{ID: "u1"}, "offer-1")

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Errorf("expected ErrStorageUnavailable, got %v", err)
	}
}




func TestCanRespond_DoesNotMutateInput(t *testing.T) {
	offer := domain.Offer{ID: "offer-1", AuthorID: "author-1", Status: domain.OfferActive}
	existing := []domain.Response{{OfferID: "offer-2", UserID: "user-9"}}
	existingCopy := append([]domain.Response{}, existing...)

	_, _ = CanRespond(domain.User{ID: "user-1"}, offer, existing)

	if len(existing) != len(existingCopy) || existing[0] != existingCopy[0] {
		t.Errorf("CanRespond must not mutate its input slice")
	}
}