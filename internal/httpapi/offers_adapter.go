package httpapi

import (
	"context"
	"sort"
	"strings"

	"skillswap/domain"
	"skillswap/internal/discovery"
)

type UserReader interface {
	Get(id string) (domain.User, error)
}

type DiscoveryService interface {
	FindRelevant(user domain.User) ([]domain.Offer, error)
}

type OffersAdapter struct {
	users     UserReader
	discovery DiscoveryService
}

func NewOffersAdapter(
	users UserReader,
	discoveryService DiscoveryService,
) *OffersAdapter {
	return &OffersAdapter{
		users:     users,
		discovery: discoveryService,
	}
}
func (a *OffersAdapter) SearchOffers(
	ctx context.Context,
	actorID string,
	skill string,
) ([]domain.Offer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, err := a.users.Get(actorID)
	if err != nil {
		return nil, err
	}
	offers, err := a.discovery.FindRelevant(user)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Offer, 0, len(offers))

	for _, offer := range offers {
		if skill != "" &&
			!strings.EqualFold(offer.Skill.Name, skill) {
			continue
		}

		result = append(result, offer)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

var _ OfferSearcher = (*OffersAdapter)(nil)
var _ DiscoveryService = (*discovery.Service)(nil)
