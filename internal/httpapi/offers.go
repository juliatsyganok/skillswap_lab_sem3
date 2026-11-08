package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"skillswap/domain"
)

type OfferSearcher interface {
	SearchOffers(
		ctx context.Context,
		actorID string,
		skill string,
	) ([]domain.Offer, error)
}

type ActorFunc func(context.Context) (string, bool)

// OffersHandler обрабатывает HTTP-запросы поиска.
type OffersHandler struct {
	searcher OfferSearcher
	actor    ActorFunc
}

func NewOffersHandler(
	searcher OfferSearcher,
	actor ActorFunc,
) *OffersHandler {
	return &OffersHandler{
		searcher: searcher,
		actor:    actor,
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	writeJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func hasRequestBody(r *http.Request) (bool, error) {
	if r.Body == nil {
		return false, nil
	}

	var buf [1]byte

	n, err := r.Body.Read(buf[:])

	if err != nil && err != io.EOF {
		return false, err
	}

	return n > 0, nil
}
func (h *OffersHandler) GetOffers(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/v1/offers" {
		writeError(
			w,
			http.StatusNotFound,
			"not_found",
			"Маршрут не найден",
		)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)

		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"Метод не поддерживается",
		)
		return
	}
	hasBody, bodyErr := hasRequestBody(r)

	if bodyErr != nil || hasBody {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_body",
			"Тело запроса запрещено",
		)
		return
	}

	query, err := ParseOffersQuery(r.URL.Query())

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_query",
			"Некорректные параметры поиска",
		)
		return
	}
	if h.actor == nil || h.searcher == nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"Сервис не настроен",
		)
		return
	}

	actorID, ok := h.actor(r.Context())

	if !ok || actorID == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_actor",
			"Не указан пользователь",
		)
		return
	}

	offers, err := h.searcher.SearchOffers(
		r.Context(),
		actorID,
		query.Skill,
	)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"not_found",
				"Пользователь не найден",
			)

		case errors.Is(err, domain.ErrStorageUnavailable):
			writeError(
				w,
				http.StatusServiceUnavailable,
				"storage_unavailable",
				"Хранилище временно недоступно",
			)

		default:
			writeError(
				w,
				http.StatusInternalServerError,
				"internal_error",
				"Ошибка поиска предложений",
			)
		}

		return
	}
	page := PaginateOffers(
		offers,
		query.Limit,
		query.Offset,
	)
	items := make([]OfferDTO, 0, len(page))

	for _, offer := range page {
		items = append(items, ToOfferDTO(offer))
	}
	writeJSON(
		w,
		http.StatusOK,
		OffersResponse{
			Items:  items,
			Limit:  query.Limit,
			Offset: query.Offset,
		},
	)
}
