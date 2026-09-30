package storage

import ("skillswap/domain")

type InMemoryOfferStore struct{
	offers map[string]domain.Offer
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
	return s.responses[userID], nil
}