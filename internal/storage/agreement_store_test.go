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

func TestInMemoryAgreementStore_Contract(t *testing.T) {
	runStoreContract(t, func() Store { return NewInMemoryAgreementStore() })
}

func TestInMemoryAgreementStore_Reads(t *testing.T) {
	s := NewInMemoryAgreementStore()

	offers, err := s.GetOffers()
	if err != nil || offers == nil || len(offers) != 0 {
		t.Fatalf("empty store offers: %#v, %v", offers, err)
	}
	all, err := s.ListResponses()
	if err != nil || all == nil || len(all) != 0 {
		t.Fatalf("empty store responses: %#v, %v", all, err)
	}
	mine, err := s.GetResponsesByUser("u2")
	if err != nil || mine == nil || len(mine) != 0 {
		t.Fatalf("no responses for user: %#v, %v", mine, err)
	}

	_ = s.SaveOffer(domain.Offer{ID: "o1", AuthorID: "u1"})
	_ = s.SaveResponse(domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponsePending})
	_ = s.SaveResponse(domain.Response{ID: "r2", OfferID: "o1", UserID: "u3", Status: domain.ResponsePending})
	// Замена по ID не даёт дубля в чтениях.
	_ = s.SaveResponse(domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponseAccepted})

	offers, _ = s.GetOffers()
	if len(offers) != 1 || offers[0].ID != "o1" {
		t.Fatalf("offers: %+v", offers)
	}
	all, _ = s.ListResponses()
	if len(all) != 2 {
		t.Fatalf("all responses: %+v", all)
	}
	mine, _ = s.GetResponsesByUser("u2")
	want := domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponseAccepted}
	if len(mine) != 1 || mine[0] != want {
		t.Fatalf("responses of u2: %+v, want [%+v]", mine, want)
	}
}

func TestInMemoryAgreementStore_OfferIsolation(t *testing.T) {
	s := NewInMemoryAgreementStore()
	offer := domain.Offer{ID: "o1", WantsInReturn: []domain.Skill{{Name: "go"}}}
	_ = s.SaveOffer(offer)

	// Изменение аргумента после Save не меняет хранилище.
	offer.WantsInReturn[0].Name = "changed"
	// Изменение результатов чтения тоже.
	got, _ := s.GetOffer("o1")
	got.WantsInReturn[0].Name = "changed"
	list, _ := s.GetOffers()
	list[0].WantsInReturn[0].Name = "changed"

	stored, _ := s.GetOffer("o1")
	if stored.WantsInReturn[0].Name != "go" {
		t.Fatalf("store shares nested slice: %+v", stored.WantsInReturn)
	}
}
