package storage

import (
	"errors"
	"testing"

	"skillswap/domain"
)

func TestInMemoryAgreementStore_SaveThenGet(t *testing.T) {
	s := NewInMemoryAgreementStore()
	offer := domain.Offer{ID: "o1", AuthorID: "u1", Status: domain.OfferActive}
	resp := domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponsePending}

	if err := s.SaveOffer(offer); err != nil {
		t.Fatalf("save offer: %v", err)
	}
	if err := s.SaveResponse(resp); err != nil {
		t.Fatalf("save response: %v", err)
	}

	gotOffer, err := s.GetOffer("o1")
	if err != nil || gotOffer.ID != offer.ID || gotOffer.Status != offer.Status {
		t.Fatalf("get offer: got %+v, %v", gotOffer, err)
	}
	gotResp, err := s.GetResponse("r1")
	if err != nil || gotResp != resp {
		t.Fatalf("get response: got %+v, %v", gotResp, err)
	}
}

func TestInMemoryAgreementStore_NotFound(t *testing.T) {
	s := NewInMemoryAgreementStore()

	if _, err := s.GetOffer("missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("offer: want ErrNotFound, got %v", err)
	}
	if _, err := s.GetResponse("missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("response: want ErrNotFound, got %v", err)
	}
}
