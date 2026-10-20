package agreement

import (
	"errors"
	"testing"

	"skillswap/domain"
	"skillswap/internal/storage"
)

type failingStore struct {
	*storage.InMemoryAgreementStore
	failResponseLoad bool
	failOfferLoad    bool
	failSave         bool
	saveCalls        int
}

func (s *failingStore) GetResponse(id string) (domain.Response, error) {
	if s.failResponseLoad {
		return domain.Response{}, domain.ErrStorageUnavailable
	}
	return s.InMemoryAgreementStore.GetResponse(id)
}
func (s *failingStore) GetOffer(id string) (domain.Offer, error) {
	if s.failOfferLoad {
		return domain.Offer{}, domain.ErrStorageUnavailable
	}
	return s.InMemoryAgreementStore.GetOffer(id)
}
func (s *failingStore) SaveResponse(r domain.Response) error {
	s.saveCalls++
	if s.failSave {
		return domain.ErrStorageUnavailable
	}
	return s.InMemoryAgreementStore.SaveResponse(r)
}

func scenarioStore(t *testing.T) *storage.InMemoryAgreementStore {
	t.Helper()
	s := storage.NewInMemoryAgreementStore()
	if err := s.SaveOffer(domain.Offer{ID: "offer-1", AuthorID: "author", Status: domain.OfferActive}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveResponse(domain.Response{ID: "response-1", OfferID: "offer-1", UserID: "learner", Status: domain.ResponsePending}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestScenario_AcceptRepeatAndRead(t *testing.T) {
	s := scenarioStore(t)
	svc := NewService(s)
	got, err := svc.Accept("author", "response-1")
	if err != nil || got.Status != domain.ResponseAccepted {
		t.Fatalf("accept: %+v, %v", got, err)
	}
	stored, err := s.GetResponse("response-1")
	if err != nil || stored.Status != domain.ResponseAccepted {
		t.Fatalf("stored after accept: %+v, %v", stored, err)
	}
	got, err = svc.Accept("author", "response-1")
	if err != nil || got.Status != domain.ResponseAccepted {
		t.Fatalf("repeat: %+v, %v", got, err)
	}
	read, err := s.GetResponse("response-1")
	if err != nil || read.Status != domain.ResponseAccepted {
		t.Fatalf("read: %+v, %v", read, err)
	}
}

func TestScenario_RefusalAndReadFailures(t *testing.T) {
	for _, tc := range []struct {
		name            string
		response, offer bool
	}{
		{"response", true, false}, {"offer", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := scenarioStore(t)
			s := &failingStore{InMemoryAgreementStore: base, failResponseLoad: tc.response, failOfferLoad: tc.offer}
			got, err := NewService(s).Accept("author", "response-1")
			if !errors.Is(err, domain.ErrStorageUnavailable) || got != (domain.Response{}) || s.saveCalls != 0 {
				t.Fatalf("got %+v, %v, saves %d", got, err, s.saveCalls)
			}
		})
	}
	t.Run("save refusal preserves data and retry works", func(t *testing.T) {
		base := scenarioStore(t)
		s := &failingStore{InMemoryAgreementStore: base, failSave: true}
		got, err := NewService(s).Accept("author", "response-1")
		if !errors.Is(err, domain.ErrStorageUnavailable) || got != (domain.Response{}) {
			t.Fatalf("got %+v, %v", got, err)
		}
		stored, _ := base.GetResponse("response-1")
		if stored.Status != domain.ResponsePending {
			t.Fatalf("failed save mutated storage: %+v", stored)
		}
		s.failSave = false
		got, err = NewService(s).Accept("author", "response-1")
		if err != nil || got.Status != domain.ResponseAccepted {
			t.Fatalf("retry: %+v, %v", got, err)
		}
	})
}

func TestScenario_ForeignAndInvalidTransitionDoNotWrite(t *testing.T) {
	for _, tc := range []struct {
		name, actor string
		status      domain.ResponseStatus
		want        error
	}{
		{"foreign user", "stranger", domain.ResponsePending, ErrNotOfferAuthor},
		{"invalid state", "author", domain.ResponseRejected, domain.ErrInvalidTransition},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := scenarioStore(t)
			if tc.status != domain.ResponsePending {
				_ = s.SaveResponse(domain.Response{ID: "response-1", OfferID: "offer-1", UserID: "learner", Status: tc.status})
			}
			before, _ := s.GetResponse("response-1")
			got, err := NewService(s).Accept(tc.actor, "response-1")
			after, _ := s.GetResponse("response-1")
			if !errors.Is(err, tc.want) || got != (domain.Response{}) || before != after {
				t.Fatalf("got %+v, %v; before %+v after %+v", got, err, before, after)
			}
		})
	}
}
