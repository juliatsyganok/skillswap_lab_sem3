package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

// ActorFunc получает ID пользователя из контекста.
// Конкретная реализация предоставляется middleware.
type ActorFunc func(context.Context) (string, bool)

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

func (h *OffersHandler) GetOffers(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"Метод не поддерживается",
		)
		return
	}

	// Для этого GET не принимаем тело запроса.
	if r.ContentLength != 0 {
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
				"Обязательный объект не найден",
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

	writeJSON(w, http.StatusOK, OffersResponse{
		Items:  items,
		Limit:  query.Limit,
		Offset: query.Offset,
	})
}
