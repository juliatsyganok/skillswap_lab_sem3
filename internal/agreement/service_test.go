package agreement

import (
	"errors"
	"testing"

	"skillswap/domain"
	"skillswap/internal/storage"
)

// In-memory хранилище должно подходить сервису без адаптеров.
var _ Store = (*storage.InMemoryAgreementStore)(nil)

// storeStub отдаёт заранее заданные значения/ошибки и запоминает, что сохраняли.
type storeStub struct {
	offer    domain.Offer
	resp     domain.Response
	offerErr error
	respErr  error
	saveErr  error

	savedOffers    []domain.Offer
	savedResponses []domain.Response
}

func (s *storeStub) GetOffer(string) (domain.Offer, error)       { return s.offer, s.offerErr }
func (s *storeStub) GetResponse(string) (domain.Response, error) { return s.resp, s.respErr }

func (s *storeStub) SaveOffer(o domain.Offer) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.savedOffers = append(s.savedOffers, o)
	return nil
}

func (s *storeStub) SaveResponse(r domain.Response) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.savedResponses = append(s.savedResponses, r)
	return nil
}

func TestService_Accept(t *testing.T) {
	store := &storeStub{offer: offer(domain.OfferActive), resp: response(domain.ResponsePending)}

	got, err := NewService(store).Accept("author", "r1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.ResponseAccepted {
		t.Fatalf("status: got %s, want accepted", got.Status)
	}
	if len(store.savedResponses) != 1 || store.savedResponses[0] != got {
		t.Fatalf("saved %v, want exactly [%v]", store.savedResponses, got)
	}
}

func TestService_Reject(t *testing.T) {
	store := &storeStub{offer: offer(domain.OfferActive), resp: response(domain.ResponsePending)}

	got, err := NewService(store).Reject("author", "r1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.ResponseRejected {
		t.Fatalf("status: got %s, want rejected", got.Status)
	}
}

func TestService_Decide_Errors(t *testing.T) {
	tests := []struct {
		name    string
		store   *storeStub
		actorID string
		wantErr error
	}{
		{
			name:    "response storage unavailable",
			store:   &storeStub{respErr: domain.ErrStorageUnavailable},
			actorID: "author",
			wantErr: domain.ErrStorageUnavailable,
		},
		{
			name:    "response not found",
			store:   &storeStub{respErr: domain.ErrNotFound},
			actorID: "author",
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "offer storage unavailable",
			store:   &storeStub{resp: response(domain.ResponsePending), offerErr: domain.ErrStorageUnavailable},
			actorID: "author",
			wantErr: domain.ErrStorageUnavailable,
		},
		{
			name:    "rule violated: not author",
			store:   &storeStub{offer: offer(domain.OfferActive), resp: response(domain.ResponsePending)},
			actorID: "u2",
			wantErr: ErrNotOfferAuthor,
		},
		{
			name:    "rule violated: already accepted",
			store:   &storeStub{offer: offer(domain.OfferActive), resp: response(domain.ResponseAccepted)},
			actorID: "author",
			wantErr: domain.ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.store).Accept(tt.actorID, "r1")

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			// Ни при отказе чтения, ни при нарушении правила ничего не сохраняем.
			if len(tt.store.savedResponses) != 0 {
				t.Fatalf("nothing should be saved, got %v", tt.store.savedResponses)
			}
		})
	}
}

func TestService_Accept_SaveFails(t *testing.T) {
	store := &storeStub{
		offer:   offer(domain.OfferActive),
		resp:    response(domain.ResponsePending),
		saveErr: domain.ErrStorageUnavailable,
	}

	got, err := NewService(store).Accept("author", "r1")

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("want ErrStorageUnavailable, got %v", err)
	}
	if got != (domain.Response{}) {
		t.Fatalf("on error result must be zero, got %+v", got)
	}
}

func TestService_SetOfferStatus(t *testing.T) {
	store := &storeStub{offer: offer(domain.OfferActive)}

	got, err := NewService(store).SetOfferStatus("author", "o1", domain.OfferClosed)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.OfferClosed {
		t.Fatalf("status: got %s, want closed", got.Status)
	}
	if len(store.savedOffers) != 1 || store.savedOffers[0].Status != domain.OfferClosed {
		t.Fatalf("saved %v, want one closed offer", store.savedOffers)
	}
}

func TestService_SetOfferStatus_Errors(t *testing.T) {
	tests := []struct {
		name    string
		store   *storeStub
		next    domain.OfferStatus
		wantErr error
	}{
		{"storage unavailable", &storeStub{offerErr: domain.ErrStorageUnavailable}, domain.OfferPaused, domain.ErrStorageUnavailable},
		{"not found", &storeStub{offerErr: domain.ErrNotFound}, domain.OfferPaused, domain.ErrNotFound},
		{"reopen closed", &storeStub{offer: offer(domain.OfferClosed)}, domain.OfferActive, domain.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.store).SetOfferStatus("author", "o1", tt.next)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if len(tt.store.savedOffers) != 0 {
				t.Fatalf("nothing should be saved, got %v", tt.store.savedOffers)
			}
		})
	}
}
