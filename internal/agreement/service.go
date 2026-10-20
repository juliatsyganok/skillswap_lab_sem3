package agreement

import (
	"fmt"

	"skillswap/domain"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Accept(actorID, responseID string) (domain.Response, error) {
	return s.decide(actorID, responseID, domain.ResponseAccepted)
}

func (s *Service) Reject(actorID, responseID string) (domain.Response, error) {
	return s.decide(actorID, responseID, domain.ResponseRejected)
}

func (s *Service) decide(actorID, responseID string, decision domain.ResponseStatus) (domain.Response, error) {
	resp, err := s.store.GetResponse(responseID)
	if err != nil {
		return domain.Response{}, fmt.Errorf("get response: %w", err)
	}

	offer, err := s.store.GetOffer(resp.OfferID)
	if err != nil {
		return domain.Response{}, fmt.Errorf("get offer: %w", err)
	}
	if resp.Status == decision {
		check := resp
		check.Status = domain.ResponsePending
		if _, err := Decide(actorID, offer, check, decision); err != nil {
			return domain.Response{}, err
		}
		return resp, nil
	}
	updated, err := Decide(actorID, offer, resp, decision)
	if err != nil {
		return domain.Response{}, err
	}

	if err := s.store.SaveResponse(updated); err != nil {
		return domain.Response{}, fmt.Errorf("save response: %w", err)
	}

	return updated, nil
}

func (s *Service) SetOfferStatus(actorID, offerID string, next domain.OfferStatus) (domain.Offer, error) {
	offer, err := s.store.GetOffer(offerID)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("get offer: %w", err)
	}
	updated, err := ChangeOfferStatus(actorID, offer, next)
	if err != nil {
		return domain.Offer{}, err
	}
	if err := s.store.SaveOffer(updated); err != nil {
		return domain.Offer{}, fmt.Errorf("save offer: %w", err)
	}
	return updated, nil
}
