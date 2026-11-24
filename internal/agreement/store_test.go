package agreement

import (
	"context"
	"fmt"

	"skillswap/domain"
)

// These sequential test doubles preserve failure injection at each read and save.
type responseOperations interface {
	GetResponse(string) (domain.Response, error)
	GetOffer(string) (domain.Offer, error)
	SaveResponse(domain.Response) error
}

func updateResponseForTest(ctx context.Context, store responseOperations, id string, decide func(domain.Offer, domain.Response) (domain.ResponseStatus, error)) (domain.Response, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.Response{}, false, err
	}
	resp, err := store.GetResponse(id)
	if err != nil {
		return domain.Response{}, false, fmt.Errorf("get response: %w", err)
	}
	offer, err := store.GetOffer(resp.OfferID)
	if err != nil {
		return domain.Response{}, false, fmt.Errorf("get offer: %w", err)
	}
	next, err := decide(offer, resp)
	if err != nil {
		return domain.Response{}, false, err
	}
	if resp.Status == next {
		return resp, false, nil
	}
	resp.Status = next
	if err := store.SaveResponse(resp); err != nil {
		return domain.Response{}, false, fmt.Errorf("save response: %w", err)
	}
	return resp, true, nil
}

func (s *storeStub) UpdateResponse(ctx context.Context, id string, decide func(domain.Offer, domain.Response) (domain.ResponseStatus, error)) (domain.Response, bool, error) {
	return updateResponseForTest(ctx, s, id, decide)
}

func (s *storeStub) UpdateOffer(id string, decide func(domain.Offer) (domain.OfferStatus, error)) (domain.Offer, error) {
	offer, err := s.GetOffer(id)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("get offer: %w", err)
	}
	next, err := decide(offer)
	if err != nil {
		return domain.Offer{}, err
	}
	offer.Status = next
	if err := s.SaveOffer(offer); err != nil {
		return domain.Offer{}, fmt.Errorf("save offer: %w", err)
	}
	return offer, nil
}

func (s *failingStore) UpdateResponse(ctx context.Context, id string, decide func(domain.Offer, domain.Response) (domain.ResponseStatus, error)) (domain.Response, bool, error) {
	return updateResponseForTest(ctx, s, id, decide)
}
