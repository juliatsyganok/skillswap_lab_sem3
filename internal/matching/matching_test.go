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

func cloneUser(u domain.User) domain.User {
	return domain.User{
		ID:     u.ID,
		Offers: append([]domain.Skill(nil), u.Offers...),
		Needs:  append([]domain.Skill(nil), u.Needs...),
		Format: u.Format,
	}
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
	subject := user("u1", []string{"go", "python"}, []string{"go", "python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u1", []string{"go", "python"}, []string{"go", "python"}, domain.FormatOnline),
	}
	want := []string{}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

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

func TestFindMatches_OneSided_CandidateOffersWhatSubjectNeeds(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u2", []string{"python"}, []string{"rust"}, domain.FormatOnline),
	}
	want := []string{}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func TestFindMatches_OneSided_SubjectOffersWhatCandidateNeeds(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u2", []string{"java"}, []string{"go"}, domain.FormatOnline),
	}
	want := []string{}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func TestFindMatches_FormatAny(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatAny)
	candidates := []domain.User{
		user("u2", []string{"python"}, []string{"go"}, domain.FormatOnline),
	}
	want := []string{"u2"}

	got := FindMatch(subject, candidates)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFindMatches_EmptyCandidates(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	want := []string{}

	got := FindMatch(subject, []domain.User{})

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func TestFindMatches_DoesNotMutateInput(t *testing.T) {
	subject := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	candidates := []domain.User{
		user("u2", []string{"python"}, []string{"go"}, domain.FormatOnline),
		user("u3", []string{"java"}, []string{"go"}, domain.FormatOnline),
	}

	subjectBefore := cloneUser(subject)
	candidatesBefore := make([]domain.User, len(candidates))
	for i, c := range candidates {
		candidatesBefore[i] = cloneUser(c)
	}

	FindMatch(subject, candidates)

	if !reflect.DeepEqual(subject, subjectBefore) {
		t.Fatalf("subject was mutated: got %v, want %v", subject, subjectBefore)
	}
	if !reflect.DeepEqual(candidates, candidatesBefore) {
		t.Fatalf("candidates were mutated: got %v, want %v", candidates, candidatesBefore)
	}
}
