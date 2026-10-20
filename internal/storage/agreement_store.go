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
	return cloneOffer(offer), nil
}

func (s *InMemoryAgreementStore) GetResponse(id string) (domain.Response, error) {
	resp, ok := s.responses[id]
	if !ok {
		return domain.Response{}, domain.ErrNotFound
	}
	return resp, nil
}

func (s *InMemoryAgreementStore) SaveOffer(offer domain.Offer) error {
	s.offers[offer.ID] = cloneOffer(offer)
	return nil
}

func (s *InMemoryAgreementStore) SaveResponse(resp domain.Response) error {
	s.responses[resp.ID] = resp
	return nil
}

// GetOffers — все предложения (порядок не гарантируется).
func (s *InMemoryAgreementStore) GetOffers() ([]domain.Offer, error) {
	offers := make([]domain.Offer, 0, len(s.offers))
	for _, offer := range s.offers {
		offers = append(offers, cloneOffer(offer))
	}
	return offers, nil
}

// ListResponses — все отклики (порядок не гарантируется).
func (s *InMemoryAgreementStore) ListResponses() ([]domain.Response, error) {
	responses := make([]domain.Response, 0, len(s.responses))
	for _, resp := range s.responses {
		responses = append(responses, resp)
	}
	return responses, nil
}

// GetResponsesByUser — отклики пользователя; нет откликов — пустой non-nil срез.
func (s *InMemoryAgreementStore) GetResponsesByUser(userID string) ([]domain.Response, error) {
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
