package httpapi

import "skillswap/domain"

type OfferDTO struct {
	ID       string `json:"id"`
	AuthorID string `json:"author_id"`
	Skill    string `json:"skill"`
	Level    string `json:"level"`
	Format   string `json:"format"`
	Status   string `json:"status"`
}

type OffersResponse struct {
	Items  []OfferDTO `json:"items"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ToOfferDTO(offer domain.Offer) OfferDTO {
	return OfferDTO{
		ID:       offer.ID,
		AuthorID: offer.AuthorID,
		Skill:    offer.Skill.Name,
		Level:    string(offer.Skill.Level),
		Format:   string(offer.Format),
		Status:   string(offer.Status),
	}
}

func PaginateOffers(
	offers []domain.Offer,
	limit int,
	offset int,
) []domain.Offer {
	if offset >= len(offers) {
		return []domain.Offer{}
	}

	end := offset + limit

	if end > len(offers) {
		end = len(offers)
	}

	return offers[offset:end]
}
