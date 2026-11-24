package engagementapi

import "skillswap/internal/agreement"

type EngagementDTO struct {
	ID        string `json:"id"`
	OfferID   string `json:"offer_id"`
	AuthorID  string `json:"author_id"`
	LearnerID string `json:"learner_id"`
	Status    string `json:"status"`
}

type AcceptanceDTO struct {
	EngagementDTO
	Changed bool `json:"changed"`
}

func toDTO(e agreement.Engagement) EngagementDTO {
	return EngagementDTO{
		ID:        e.Response.ID,
		OfferID:   e.Response.OfferID,
		AuthorID:  e.Offer.AuthorID,
		LearnerID: e.Response.UserID,
		Status:    string(e.Response.Status),
	}
}
