package agreement_test

import (
	"reflect"
	"sort"
	"testing"

	"skillswap/domain"
	"skillswap/internal/agreement"
	"skillswap/internal/discovery"
	"skillswap/internal/matching"
	"skillswap/internal/storage"
)

// Команда и поиск работают с одним и тем же хранилищем состояния.
var (
	_ agreement.Store          = (*storage.InMemoryAgreementStore)(nil)
	_ matching.AgreementReader = (*storage.InMemoryAgreementStore)(nil)
	_ discovery.OfferReader    = (*storage.InMemoryAgreementStore)(nil)
	_ discovery.ResponseReader = (*storage.InMemoryAgreementStore)(nil)
)

func skills(names ...string) []domain.Skill {
	result := make([]domain.Skill, 0, len(names))
	for _, n := range names {
		result = append(result, domain.Skill{Name: n})
	}
	return result
}

type world struct {
	state     *storage.InMemoryAgreementStore
	command   *agreement.Service
	matching  *matching.Service
	discovery *discovery.Service
	learner   domain.User
}

// author учит go и хочет python; learner и other — наоборот, поэтому оба
// совместимы с author. У learner есть отклик в pending на предложение author.
func newWorld(t *testing.T) world {
	t.Helper()
	author := domain.User{ID: "author", Offers: skills("go"), Needs: skills("python"), Format: domain.FormatOnline}
	learner := domain.User{ID: "learner", Offers: skills("python"), Needs: skills("go"), Format: domain.FormatOnline}
	other := domain.User{ID: "other", Offers: skills("python"), Needs: skills("go"), Format: domain.FormatAny}

	profiles := storage.NewInMemoryMatchingStore()
	for _, u := range []domain.User{author, learner, other} {
		profiles.AddUser(u)
	}

	state := storage.NewInMemoryAgreementStore()
	offer := domain.Offer{ID: "o1", AuthorID: "author", Skill: domain.Skill{Name: "go"},
		WantsInReturn: skills("python"), Format: domain.FormatOnline, Status: domain.OfferActive}
	if err := state.SaveOffer(offer); err != nil {
		t.Fatal(err)
	}
	if err := state.SaveResponse(domain.Response{ID: "r1", OfferID: "o1", UserID: "learner", Status: domain.ResponsePending}); err != nil {
		t.Fatal(err)
	}

	return world{
		state:     state,
		command:   agreement.NewService(state),
		matching:  matching.NewService(profiles, state),
		discovery: discovery.NewService(state, state),
		learner:   learner,
	}
}

// matches — выдача подбора; порядок хранилище не обещает, поэтому сортируем.
func (w world) matches(t *testing.T, userID string) []string {
	t.Helper()
	ids, err := w.matching.FindMatchFor(userID)
	if err != nil {
		t.Fatalf("FindMatchFor(%s): %v", userID, err)
	}
	sort.Strings(ids)
	return ids
}

func (w world) relevant(t *testing.T) []string {
	t.Helper()
	offers, err := w.discovery.FindRelevant(w.learner)
	if err != nil {
		t.Fatalf("FindRelevant: %v", err)
	}
	ids := make([]string, 0, len(offers))
	for _, o := range offers {
		ids = append(ids, o.ID)
	}
	return ids
}

func check(t *testing.T, step, what string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s, %s: got %v, want %v", step, what, got, want)
	}
}

func TestReadAfterWrite_AcceptChangesSearch(t *testing.T) {
	w := newWorld(t)

	// До подтверждения: пара доступна обеим сторонам, предложение видно.
	check(t, "before", "matches of author", w.matches(t, "author"), []string{"learner", "other"})
	check(t, "before", "matches of learner", w.matches(t, "learner"), []string{"author"})
	check(t, "before", "relevant for learner", w.relevant(t), []string{"o1"})

	if _, err := w.command.Accept("author", "r1"); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// После: подтверждённая пара и предложение больше не предлагаются.
	check(t, "after accept", "matches of author", w.matches(t, "author"), []string{"other"})
	check(t, "after accept", "matches of learner", w.matches(t, "learner"), []string{})
	check(t, "after accept", "relevant for learner", w.relevant(t), []string{})

	// Повтор — no-op: состояние и выдача те же.
	if _, err := w.command.Accept("author", "r1"); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	stored, err := w.state.GetResponse("r1")
	if err != nil || stored.Status != domain.ResponseAccepted {
		t.Fatalf("stored after repeat: %+v, %v", stored, err)
	}
	check(t, "after repeat", "matches of author", w.matches(t, "author"), []string{"other"})
	check(t, "after repeat", "relevant for learner", w.relevant(t), []string{})
}

func TestReadAfterWrite_RefusedCommandKeepsSearch(t *testing.T) {
	w := newWorld(t)

	// Чужой пользователь получает отказ — ничего не записано, выдача прежняя.
	if _, err := w.command.Accept("stranger", "r1"); err == nil {
		t.Fatal("stranger must be refused")
	}
	check(t, "after refusal", "matches of author", w.matches(t, "author"), []string{"learner", "other"})
	check(t, "after refusal", "relevant for learner", w.relevant(t), []string{"o1"})

	// Отклонение — не договорённость: пара по-прежнему доступна.
	if _, err := w.command.Reject("author", "r1"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	check(t, "after reject", "matches of author", w.matches(t, "author"), []string{"learner", "other"})
	check(t, "after reject", "relevant for learner", w.relevant(t), []string{"o1"})
}
