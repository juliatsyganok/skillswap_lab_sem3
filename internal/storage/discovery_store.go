package storage

import "skillswap/domain"

func (s *InMemoryOfferStore) GetOffers() ([]domain.Offer, error) {
	offers := make([]domain.Offer, 0, len(s.offers))

	for _, offer := range s.offers {
		offers = append(offers, offer)
	}

	return offers, nil
}
