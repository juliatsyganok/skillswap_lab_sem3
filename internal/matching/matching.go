package matching

import "skillswap/domain"

func FindMatch(subject domain.User, candidates []domain.User) []string {
	result := make([]string, 0)
	for _, candidate := range candidates {
		if candidate.ID == subject.ID {
			continue
		}
		if !mutuallyCompatible(subject, candidate) {
			continue
		}
		result = append(result, candidate.ID)
	}
	return result
}

func mutuallyCompatible(a, b domain.User) bool {
	if !skill(a.Offers, b.Needs) {
		return false
	}
	if !skill(b.Offers, a.Needs) {
		return false
	}
	if !formats(a.Format, b.Format) {
		return false
	}
	return true
}

func skill(offers, needs []domain.Skill) bool {
	for _, offer := range offers {
		for _, need := range needs {
			if offer.Name == need.Name {
				return true
			}
		}
	}
	return false
}

func formats(a, b domain.Format) bool {
	return a == b || a == domain.FormatAny || b == domain.FormatAny
}
