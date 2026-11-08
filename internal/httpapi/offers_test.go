package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"skillswap/domain"
	"strings"
	"testing"
)

type fakeSearcher struct {
	offers []domain.Offer
	calls  int
}

func (f *fakeSearcher) SearchOffers(
	ctx context.Context,
	actorID string,
	skill string,
) ([]domain.Offer, error) {
	f.calls++

	result := make([]domain.Offer, 0)

	for _, offer := range f.offers {
		if skill == "" ||
			strings.EqualFold(offer.Skill.Name, skill) {
			result = append(result, offer)
		}
	}

	return result, nil
}

func testActor(context.Context) (string, bool) {
	return "user-1", true
}

func testOffers() []domain.Offer {
	return []domain.Offer{
		{ID: "1", Skill: domain.Skill{Name: "Go"}},
		{ID: "2", Skill: domain.Skill{Name: "Python"}},
		{ID: "3", Skill: domain.Skill{Name: "Go"}},
		{ID: "4", Skill: domain.Skill{Name: "Go"}},
		{ID: "5", Skill: domain.Skill{Name: "Java"}},
		{ID: "6", Skill: domain.Skill{Name: "Go"}},
	}
}

func performRequest(
	h *OffersHandler,
	path string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	h.GetOffers(rec, req)
	return rec
}

func decodeOffers(
	t *testing.T,
	rec *httptest.ResponseRecorder,
) OffersResponse {
	t.Helper()

	var response OffersResponse

	if err := json.Unmarshal(
		rec.Body.Bytes(), &response,
	); err != nil {
		t.Fatal(err)
	}

	return response
}

func TestGetOffersPagination(t *testing.T) {
	searcher := &fakeSearcher{offers: testOffers()}
	handler := NewOffersHandler(searcher, testActor)

	first := performRequest(
		handler,
		"/v1/offers?skill=go&limit=2&offset=0",
	)
	second := performRequest(
		handler,
		"/v1/offers?skill=go&limit=2&offset=2",
	)

	if first.Code != http.StatusOK ||
		second.Code != http.StatusOK {
		t.Fatal("expected HTTP 200")
	}

	page1 := decodeOffers(t, first)
	page2 := decodeOffers(t, second)

	if len(page1.Items) != 2 ||
		len(page2.Items) != 2 {
		t.Fatal("expected two items on each page")
	}

	if page1.Items[0].ID != "1" ||
		page1.Items[1].ID != "3" {
		t.Fatal("incorrect first page")
	}

	if page2.Items[0].ID != "4" ||
		page2.Items[1].ID != "6" {
		t.Fatal("incorrect second page")
	}
}

func TestGetOffersEmpty(t *testing.T) {
	searcher := &fakeSearcher{offers: testOffers()}
	handler := NewOffersHandler(searcher, testActor)

	rec := performRequest(
		handler,
		"/v1/offers?skill=rust",
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	response := decodeOffers(t, rec)

	if response.Items == nil {
		t.Fatal("items must not be nil")
	}

	if len(response.Items) != 0 {
		t.Fatal("expected empty items")
	}
}

func TestGetOffersInvalidQuery(t *testing.T) {
	cases := []string{
		"/v1/offers?limit=0",
		"/v1/offers?limit=51",
		"/v1/offers?limit=abc",
		"/v1/offers?offset=-1",
		"/v1/offers?offset=abc",
		"/v1/offers?limit=2&limit=3",
		"/v1/offers?unknown=value",
		"/v1/offers?skill=",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			searcher := &fakeSearcher{
				offers: testOffers(),
			}
			handler := NewOffersHandler(searcher, testActor)

			rec := performRequest(handler, path)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected 400, got %d",
					rec.Code,
				)
			}

			if searcher.calls != 0 {
				t.Fatal("searcher must not be called")
			}
		})
	}
}

func TestGetOffersDefaults(t *testing.T) {
	searcher := &fakeSearcher{offers: testOffers()}
	handler := NewOffersHandler(searcher, testActor)

	rec := performRequest(handler, "/v1/offers")
	response := decodeOffers(t, rec)

	if response.Limit != 20 || response.Offset != 0 {
		t.Fatal("incorrect default pagination")
	}
}

func TestGetOffersOffsetBeyondEnd(t *testing.T) {
	searcher := &fakeSearcher{offers: testOffers()}
	handler := NewOffersHandler(searcher, testActor)

	rec := performRequest(
		handler,
		"/v1/offers?limit=2&offset=100",
	)
	response := decodeOffers(t, rec)

	if response.Items == nil || len(response.Items) != 0 {
		t.Fatal("expected non-nil empty items")
	}
}

func TestGetOffersStablePagination(t *testing.T) {
	searcher := &fakeSearcher{offers: testOffers()}
	handler := NewOffersHandler(searcher, testActor)

	first := decodeOffers(t, performRequest(
		handler, "/v1/offers?skill=go&limit=2&offset=0",
	))
	second := decodeOffers(t, performRequest(
		handler, "/v1/offers?skill=go&limit=2&offset=0",
	))

	if !reflect.DeepEqual(first, second) {
		t.Fatal("same data must produce same page")
	}
}
