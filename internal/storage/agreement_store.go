package storage

import "skillswap/domain"

// InMemoryAgreementStore хранит предложения и отклики для контракта D.
type InMemoryAgreementStore struct {
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
	offer, ok := s.offers[id]
	if !ok {
		return domain.Offer{}, domain.ErrNotFound
	}
	return offer, nil
}

func (s *InMemoryAgreementStore) GetResponse(id string) (domain.Response, error) {
	resp, ok := s.responses[id]
	if !ok {
		return domain.Response{}, domain.ErrNotFound
	}
	return resp, nil
}

func (s *InMemoryAgreementStore) SaveOffer(offer domain.Offer) error {
	s.offers[offer.ID] = offer
	return nil
}

func (s *InMemoryAgreementStore) SaveResponse(resp domain.Response) error {
	s.responses[resp.ID] = resp
	return nil
}
