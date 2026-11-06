package agreement

import (
	"context"

	"skillswap/domain"
)

// Store serializes each update across all callers sharing the same state.
// Decision callbacks run on isolated values and must complete without I/O.
type Store interface {
	// UpdateResponse loads the response and its offer, applies the decision, and
	// stores only a changed status. On error it returns zero values and leaves
	// state unchanged. A canceled context is checked before committing.
	UpdateResponse(ctx context.Context, id string, decide func(domain.Offer, domain.Response) (domain.ResponseStatus, error)) (domain.Response, bool, error)
	UpdateOffer(id string, decide func(domain.Offer) (domain.OfferStatus, error)) (domain.Offer, error)
}
