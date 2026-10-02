package discovery

import "skillswap/domain"

func IsRelevant(user domain.User, offer domain.Offer) bool {
	if offer.Status != domain.OfferActive {
		return false
	}

	if offer.AuthorID == user.ID {
		return false
	}

	if !formatCompatible(user.Format, offer.Format) {
		return false
	}

	if !needsSkill(user.Needs, offer.Skill.Name) {
		return false
	}

	if len(offer.WantsInReturn) > 0 && !hasWantedSkill(user.Offers, offer.WantsInReturn) {
		return false
	}

	return true
}

func FindRelevant(user domain.User, offers []domain.Offer) []domain.Offer {
	result := make([]domain.Offer, 0)

	for _, offer := range offers {
		if IsRelevant(user, offer) {
			result = append(result, offer)
		}
	}

	return result
}

func needsSkill(needs []domain.Skill, skillName string) bool {
	for _, skill := range needs {
		if skill.Name == skillName {
			return true
		}
	}

	return false
}

func hasWantedSkill(userOffers []domain.Skill, wants []domain.Skill) bool {
	for _, userSkill := range userOffers {
		for _, wantedSkill := range wants {
			if userSkill.Name == wantedSkill.Name {
				return true
			}
		}
	}

	return false
}

func formatCompatible(userFormat, offerFormat domain.Format) bool {
	return userFormat == offerFormat ||
		userFormat == domain.FormatAny ||
		offerFormat == domain.FormatAny
}

type OfferReader interface {
	GetOffers() ([]domain.Offer, error)
}

type Service struct {
	reader OfferReader
}

func NewService(reader OfferReader) *Service {
	return &Service{
		reader: reader,
	}
}

func (s *Service) FindRelevant(user domain.User) ([]domain.Offer, error) {
	offers, err := s.reader.GetOffers()
	if err != nil {
		return nil, err
	}

	return FindRelevant(user, offers), nil
}
