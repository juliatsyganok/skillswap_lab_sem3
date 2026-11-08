package agreement

import (
	"context"

	"skillswap/domain"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Accept(actorID, responseID string) (domain.Response, error) {
	response, _, err := s.AcceptContext(context.Background(), actorID, responseID)
	return response, err
}

// AcceptContext commits the local transition. changed is true only if a status
// was persisted. Call external dependencies before entering this operation.
func (s *Service) AcceptContext(ctx context.Context, actorID, responseID string) (response domain.Response, changed bool, err error) {
	return s.decide(ctx, actorID, responseID, domain.ResponseAccepted)
}

func (s *Service) Reject(actorID, responseID string) (domain.Response, error) {
	response, _, err := s.decide(context.Background(), actorID, responseID, domain.ResponseRejected)
	return response, err
}

func (s *Service) decide(ctx context.Context, actorID, responseID string, decision domain.ResponseStatus) (domain.Response, bool, error) {
	return s.store.UpdateResponse(ctx, responseID, func(offer domain.Offer, resp domain.Response) (domain.ResponseStatus, error) {
		// Preserve the service's LR2 repeat behavior and recheck permissions.
		if resp.Status == domain.ResponseRejected && decision == domain.ResponseRejected {
			resp.Status = domain.ResponsePending
		}
		updated, err := Decide(actorID, offer, resp, decision)
		return updated.Status, err
	})
}

func (s *Service) SetOfferStatus(actorID, offerID string, next domain.OfferStatus) (domain.Offer, error) {
	return s.store.UpdateOffer(offerID, func(offer domain.Offer) (domain.OfferStatus, error) {
		updated, err := ChangeOfferStatus(actorID, offer, next)
		return updated.Status, err
	})
}
