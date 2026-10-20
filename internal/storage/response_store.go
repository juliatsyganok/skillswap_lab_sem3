package storage

import (
	"skillswap/domain"
)

type Store interface {
	SaveResponse(domain.Response) error
	GetResponse(string) (domain.Response, error)
}

type InMemoryOfferStore struct {
	offers    map[string]domain.Offer
	responses map[string][]domain.Response
}

func NewInMemoryOfferStore() *InMemoryOfferStore {
	return &InMemoryOfferStore{
		offers:    make(map[string]domain.Offer),
		responses: make(map[string][]domain.Response),
	}
}

func (s *InMemoryOfferStore) AddOffer(o domain.Offer) {
	s.offers[o.ID] = o
}

func (s *InMemoryOfferStore) GetOffer(offerID string) (domain.Offer, error) {
	offer, ok := s.offers[offerID]
	if !ok {
		return domain.Offer{}, domain.ErrNotFound
	}
	return offer, nil
}

func (s *InMemoryOfferStore) GetResponsesByUser(userID string) ([]domain.Response, error) {
	responses := s.responses[userID]
	if responses == nil {
		return make([]domain.Response, 0), nil
	}
	return append([]domain.Response(nil), responses...), nil
}

// SaveResponse replaces the response with the same ID.
func (s *InMemoryOfferStore) SaveResponse(r domain.Response) error {
	for userID, responses := range s.responses {
		filtered := responses[:0]
		for _, existing := range responses {
			if existing.ID != r.ID {
				filtered = append(filtered, existing)
			}
		}
		s.responses[userID] = filtered
	}
	if s.responses[r.UserID] == nil {
		s.responses[r.UserID] = make([]domain.Response, 0)
	}
	responses := s.responses[r.UserID]
	for i := range responses {
		if responses[i].ID == r.ID {
			responses[i] = r
			s.responses[r.UserID] = responses
			return nil
		}
	}
	s.responses[r.UserID] = append(responses, r)
	return nil
}

func (s *InMemoryOfferStore) GetResponse(key string) (domain.Response, error) {
	for _, responses := range s.responses {
		for _, r := range responses {
			if r.ID == key {
				return r, nil
			}
		}
	}
	return domain.Response{}, domain.ErrNotFound
}

func (s *InMemoryOfferStore) Load(key string) (domain.Response, error) { return s.GetResponse(key) }

func (s *InMemoryOfferStore) Save(r domain.Response) error { return s.SaveResponse(r) }
