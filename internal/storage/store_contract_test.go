package storage

import (
	"errors"
	"testing"

	"skillswap/domain"
)

func runStoreContract(t *testing.T, newStore func() Store) {
	t.Helper()
	t.Run("missing returns ErrNotFound", func(t *testing.T) {
		if _, err := newStore().GetResponse("missing"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
	t.Run("save load replace and instance isolation", func(t *testing.T) {
		a, b := newStore(), newStore()
		want := domain.Response{ID: "r1", OfferID: "o1", UserID: "u1", Status: domain.ResponsePending}
		if err := a.SaveResponse(want); err != nil {
			t.Fatal(err)
		}
		got, err := a.GetResponse(want.ID)
		if err != nil || got != want {
			t.Fatalf("got %+v, %v; want %+v", got, err, want)
		}
		if _, err := b.GetResponse(want.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("independent store: %v", err)
		}
		want.Status = domain.ResponseAccepted
		if err := a.SaveResponse(want); err != nil {
			t.Fatal(err)
		}
		got, err = a.GetResponse(want.ID)
		if err != nil || got != want {
			t.Fatalf("replacement got %+v, %v; want %+v", got, err, want)
		}
	})
}

func TestInMemoryOfferStore_Contract(t *testing.T) {
	runStoreContract(t, func() Store { return NewInMemoryOfferStore() })
}
