package matching

import "skillswap/domain"

// ConfirmedPartners возвращает ID пользователей, с которыми у userID есть
// подтверждённая договорённость. Пара — автор предложения и тот, чей отклик
// принят (ResponseAccepted); порядок участников не важен. Отклик на
// неизвестное предложение пропускается: без автора пару не определить.
func ConfirmedPartners(userID string, offers []domain.Offer, responses []domain.Response) map[string]bool {
	authors := make(map[string]string, len(offers))
	for _, offer := range offers {
		authors[offer.ID] = offer.AuthorID
	}

	partners := make(map[string]bool)
	for _, resp := range responses {
		if resp.Status != domain.ResponseAccepted {
			continue
		}
		author, ok := authors[resp.OfferID]
		if !ok {
			continue
		}
		switch userID {
		case resp.UserID:
			partners[author] = true
		case author:
			partners[resp.UserID] = true
		}
	}
	return partners
}

// ExcludePartners убирает из ids подтверждённых партнёров, сохраняя порядок.
// Пустой результат — непустой (non-nil) срез, как у FindMatch.
func ExcludePartners(ids []string, partners map[string]bool) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if partners[id] {
			continue
		}
		result = append(result, id)
	}
	return result
}
