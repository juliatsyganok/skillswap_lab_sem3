package matching

import (
	"errors"
	"reflect"
	"testing"

	"skillswap/domain"
)

// readerStub отдаёт заданные значения/ошибки и считает вызовы.
type readerStub struct {
	users     map[string]domain.User
	list      []domain.User
	offers    []domain.Offer
	responses []domain.Response

	getErr, listErr, offersErr, responsesErr error

	calls []string
}

func (r *readerStub) Get(id string) (domain.User, error) {
	r.calls = append(r.calls, "Get")
	if r.getErr != nil {
		return domain.User{}, r.getErr
	}
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (r *readerStub) List() ([]domain.User, error) {
	r.calls = append(r.calls, "List")
	return r.list, r.listErr
}

func (r *readerStub) GetOffers() ([]domain.Offer, error) {
	r.calls = append(r.calls, "GetOffers")
	return r.offers, r.offersErr
}

func (r *readerStub) ListResponses() ([]domain.Response, error) {
	r.calls = append(r.calls, "ListResponses")
	return r.responses, r.responsesErr
}

// u1 совместим и с u2, и с u3. Предложение o1 принадлежит u1.
func newReader(responses ...domain.Response) *readerStub {
	u1 := user("u1", []string{"go"}, []string{"python"}, domain.FormatOnline)
	u2 := user("u2", []string{"python"}, []string{"go"}, domain.FormatOnline)
	u3 := user("u3", []string{"python"}, []string{"go"}, domain.FormatAny)
	return &readerStub{
		users:     map[string]domain.User{"u1": u1, "u2": u2, "u3": u3},
		list:      []domain.User{u1, u2, u3},
		offers:    []domain.Offer{{ID: "o1", AuthorID: "u1"}},
		responses: responses,
	}
}

func TestService_FindMatchFor_ExcludesConfirmedPair(t *testing.T) {
	r := newReader(domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponseAccepted})

	got, err := NewService(r, r).FindMatchFor("u1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"u3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestService_FindMatchFor_ExcludesConfirmedPairForSecondSide(t *testing.T) {
	r := newReader(domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponseAccepted})

	got, err := NewService(r, r).FindMatchFor("u2")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestService_FindMatchFor_PendingKeepsPair(t *testing.T) {
	r := newReader(domain.Response{ID: "r1", OfferID: "o1", UserID: "u2", Status: domain.ResponsePending})

	got, err := NewService(r, r).FindMatchFor("u1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"u2", "u3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestService_FindMatchFor_SubjectNotFound(t *testing.T) {
	r := newReader()

	got, err := NewService(r, r).FindMatchFor("missing")

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if got != nil {
		t.Fatalf("on error result must be nil, got %v", got)
	}
}

func TestService_FindMatchFor_ReadErrors(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(r *readerStub)
		wantCalls []string
	}{
		{"get subject", func(r *readerStub) { r.getErr = domain.ErrStorageUnavailable }, []string{"Get"}},
		{"list users", func(r *readerStub) { r.listErr = domain.ErrStorageUnavailable }, []string{"Get", "List"}},
		{"get offers", func(r *readerStub) { r.offersErr = domain.ErrStorageUnavailable }, []string{"Get", "List", "GetOffers"}},
		{"list responses", func(r *readerStub) { r.responsesErr = domain.ErrStorageUnavailable }, []string{"Get", "List", "GetOffers", "ListResponses"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newReader()
			tt.setup(r)

			got, err := NewService(r, r).FindMatchFor("u1")

			if !errors.Is(err, domain.ErrStorageUnavailable) {
				t.Fatalf("want ErrStorageUnavailable, got %v", err)
			}
			if got != nil {
				t.Fatalf("on error result must be nil, got %v", got)
			}
			// После отказа следующие чтения не выполняются.
			if !reflect.DeepEqual(r.calls, tt.wantCalls) {
				t.Fatalf("calls: got %v, want %v", r.calls, tt.wantCalls)
			}
		})
	}
}
