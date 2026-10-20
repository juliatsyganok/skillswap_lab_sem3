package discovery

import (
	"fmt"

	"skillswap/domain"
)

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

// ExcludeConfirmed убирает предложения, по которым у пользователя уже
// подтверждена договорённость (его отклик принят). Порядок сохраняется,
// пустой результат — непустой (non-nil) срез, как у FindRelevant.
func ExcludeConfirmed(offers []domain.Offer, userID string, responses []domain.Response) []domain.Offer {
	confirmed := make(map[string]bool)
	for _, resp := range responses {
		if resp.UserID == userID && resp.Status == domain.ResponseAccepted {
			confirmed[resp.OfferID] = true
		}
	}

	result := make([]domain.Offer, 0, len(offers))
	for _, offer := range offers {
		if confirmed[offer.ID] {
			continue
		}
		result = append(result, offer)
	}

	return result
}

type OfferReader interface {
	GetOffers() ([]domain.Offer, error)
}

type ResponseReader interface {
	GetResponsesByUser(userID string) ([]domain.Response, error)
}

type Service struct {
	reader    OfferReader
	responses ResponseReader
}

func NewService(reader OfferReader, responses ResponseReader) *Service {
	return &Service{
		reader:    reader,
		responses: responses,
	}
}

// FindRelevant возвращает релевантные предложения без тех, по которым
// договорённость пользователя уже подтверждена.
func (s *Service) FindRelevant(user domain.User) ([]domain.Offer, error) {
	offers, err := s.reader.GetOffers()
	if err != nil {
		return nil, fmt.Errorf("get offers: %w", err)
	}

	responses, err := s.responses.GetResponsesByUser(user.ID)
	if err != nil {
		return nil, fmt.Errorf("get responses: %w", err)
	}

	return ExcludeConfirmed(FindRelevant(user, offers), user.ID, responses), nil
}
