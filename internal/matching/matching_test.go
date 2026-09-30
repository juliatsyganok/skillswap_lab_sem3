package matching

import (
	"reflect"
	"skillswap/domain"
	"testing"
)

func user(id string, offers, needs []string, format domain.Format) domain.User {
	return domain.User{
		ID:     id,
		Offers: toSkills(offers),
		Needs:  toSkills(needs),
		Format: format,
	}
}

func toSkills(names []string) []domain.Skill {
	skills := make([]domain.Skill, len(names))
	for i, n := range names {
		skills[i] = domain.Skill{Name: n}
	}
	return skills
}

func TestFindMatches_HappyPath(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u2", []string{"python"}, []string{"go"}, domain.FormatOnline),
	}
	want := []string{"u2"}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFindMatches_SelfExcluded(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline),
	}
	want := []string{}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// по трбованию краевого теста, котрый упадёт, если убрать проверку на format
func TestFindMatches_FormatMismatch(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u2", []string{"python"}, []string{"go"}, domain.FormatOffline),
	}
	want := []string{}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
