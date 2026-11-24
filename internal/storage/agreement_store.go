package storage

import (
	"context"
	"fmt"
	"sync"

	"skillswap/domain"
)

// InMemoryAgreementStore хранит предложения и отклики для контракта D.
type InMemoryAgreementStore struct {
	mu        sync.RWMutex
	offers    map[string]domain.Offer
	responses map[string]domain.Response
}

func NewInMemoryAgreementStore() *InMemoryAgreementStore {
	return &InMemoryAgreementStore{
		offers:    make(map[string]domain.Offer),
		responses: make(map[string]domain.Response),
	}
}

func (s *InMemoryAgreementStore) GetOffer(id string) (domain.Offer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getOffer(id)
}

func (s *InMemoryAgreementStore) getOffer(id string) (domain.Offer, error) {
	offer, ok := s.offers[id]
	if !ok {
		return domain.Offer{}, domain.ErrNotFound
	}
	return cloneOffer(offer), nil
}

func (s *InMemoryAgreementStore) GetResponse(id string) (domain.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getResponse(id)
}

func (s *InMemoryAgreementStore) getResponse(id string) (domain.Response, error) {
	resp, ok := s.responses[id]
	if !ok {
		return domain.Response{}, domain.ErrNotFound
	}
	return resp, nil
}

func (s *InMemoryAgreementStore) SaveOffer(offer domain.Offer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offers[offer.ID] = cloneOffer(offer)
	return nil
}

func (s *InMemoryAgreementStore) SaveResponse(resp domain.Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[resp.ID] = resp
	return nil
}

// GetOffers — все предложения (порядок не гарантируется).
func (s *InMemoryAgreementStore) GetOffers() ([]domain.Offer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	offers := make([]domain.Offer, 0, len(s.offers))
	for _, offer := range s.offers {
		offers = append(offers, cloneOffer(offer))
	}
	return offers, nil
}

// ListResponses — все отклики (порядок не гарантируется).
func (s *InMemoryAgreementStore) ListResponses() ([]domain.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	responses := make([]domain.Response, 0, len(s.responses))
	for _, resp := range s.responses {
		responses = append(responses, resp)
	}
	return responses, nil
}

// GetResponsesByUser — отклики пользователя; нет откликов — пустой non-nil срез.
func (s *InMemoryAgreementStore) GetResponsesByUser(userID string) ([]domain.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	responses := make([]domain.Response, 0)
	for _, resp := range s.responses {
		if resp.UserID == userID {
			responses = append(responses, resp)
		}
	}
	return responses, nil
}

// cloneOffer копирует вложенный срез, чтобы хранилище не делило его с вызывающим.
func cloneOffer(offer domain.Offer) domain.Offer {
	if offer.WantsInReturn != nil {
		offer.WantsInReturn = append(make([]domain.Skill, 0, len(offer.WantsInReturn)), offer.WantsInReturn...)
	}
	return offer
}

// UpdateResponse keeps the offer and response stable from the first read
// through the status write. The callback must not call back into the store.
func (s *InMemoryAgreementStore) UpdateResponse(ctx context.Context, id string, decide func(domain.Offer, domain.Response) (domain.ResponseStatus, error)) (domain.Response, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.Response{}, false, err
	}

	resp, err := s.getResponse(id)
	if err != nil {
		return domain.Response{}, false, fmt.Errorf("get response: %w", err)
	}
	offer, err := s.getOffer(resp.OfferID)
	if err != nil {
		return domain.Response{}, false, fmt.Errorf("get offer: %w", err)
	}
	next, err := decide(offer, resp)
	if err != nil {
		return domain.Response{}, false, err
	}
	if err := ctx.Err(); err != nil {
		return domain.Response{}, false, err
	}
	if next == resp.Status {
		return resp, false, nil
	}
	resp.Status = next
	s.responses[id] = resp
	return resp, true, nil
}

func (s *InMemoryAgreementStore) UpdateOffer(id string, decide func(domain.Offer) (domain.OfferStatus, error)) (domain.Offer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	offer, err := s.getOffer(id)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("get offer: %w", err)
	}
	next, err := decide(offer)
	if err != nil {
		return domain.Offer{}, err
	}
	if next != offer.Status {
		stored := s.offers[id]
		stored.Status = next
		s.offers[id] = stored
	}
	offer.Status = next
	return offer, nil
}
