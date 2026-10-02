package response

import "skillswap/domain"

func CanRespond(user domain.User, offer domain.Offer, existing []domain.Response) (bool, error) {
	if offer.Status != domain.OfferActive {
		return false, nil
	}
	if offer.AuthorID == user.ID {
		return false, nil
	}
	for _, r := range existing {
		if r.OfferID == offer.ID && r.UserID == user.ID {
			return false, nil
		}
	}
	return true, nil
}

type ResponseReader interface {
	GetOffer(offerID string) (domain.Offer, error)
	GetResponsesByUser(userID string) ([]domain.Response, error)
}

type Service struct {
	reader ResponseReader
}

func NewService(r ResponseReader) *Service {
	return &Service{reader: r}
}

func (s *Service) CheckCanRespond(user domain.User, offerID string) (bool, error) {
	offer, err := s.reader.GetOffer(offerID)
	if err != nil {
		return false, err
	}
	existing, err := s.reader.GetResponsesByUser(user.ID)

	if err != nil {
		return false, err
	}
	return CanRespond(user, offer, existing)
}
