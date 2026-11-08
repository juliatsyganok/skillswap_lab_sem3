package engagementapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"skillswap/domain"
	"skillswap/internal/agreement"
)

type UseCase interface {
	Accept(ctx context.Context, actorID, responseID string) (agreement.Engagement, bool, error)
	Get(ctx context.Context, actorID, responseID string) (agreement.Engagement, error)
}

type Handler struct {
	UseCase   UseCase
	Actor     func(context.Context) string
	RequestID func(context.Context) string
	Logger    *slog.Logger
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("PUT /v1/engagements/{id}/acceptance", h.accept)
	mux.HandleFunc("GET /v1/engagements/{id}", h.get)
}

var idFormat = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.parse(w, r)
	if !ok {
		return
	}
	engagement, changed, err := h.UseCase.Accept(r.Context(), actor, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, AcceptanceDTO{EngagementDTO: toDTO(engagement), Changed: changed})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.parse(w, r)
	if !ok {
		return
	}
	engagement, err := h.UseCase.Get(r.Context(), actor, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(engagement))
}

func (h *Handler) parse(w http.ResponseWriter, r *http.Request) (actor, id string, ok bool) {
	actor = h.Actor(r.Context())
	if !idFormat.MatchString(actor) {
		writeError(w, http.StatusBadRequest, "invalid_demo_user", "Укажите корректный X-Demo-User")
		return "", "", false
	}
	id = r.PathValue("id")
	if !idFormat.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid_id", "Некорректный ID договорённости")
		return "", "", false
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		writeError(w, http.StatusBadRequest, "query_not_allowed", "Маршрут не принимает query-параметры")
		return "", "", false
	}
	if hasBody(r) {
		writeError(w, http.StatusBadRequest, "body_not_allowed", "Маршрут не принимает тело запроса")
		return "", "", false
	}
	return actor, id, true
}

func hasBody(r *http.Request) bool {
	if r.ContentLength > 0 {
		return true
	}
	if r.Body == nil || r.Body == http.NoBody {
		return false
	}
	n, err := io.ReadFull(r.Body, make([]byte, 1))
	return n > 0 || (err != nil && !errors.Is(err, io.EOF))
}

type failure struct {
	status        int
	code, message string
}

func classify(err error) failure {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return failure{http.StatusGatewayTimeout, "dependency_timeout", "Истёк бюджет ожидания зависимости"}
	case errors.Is(err, context.Canceled):
		return failure{http.StatusServiceUnavailable, "request_canceled", "Запрос отменён"}
	case errors.Is(err, agreement.ErrOfferMissing):
		return failure{http.StatusNotFound, "offer_not_found", "Предложение не найдено"}
	case errors.Is(err, domain.ErrNotFound):
		return failure{http.StatusNotFound, "engagement_not_found", "Договорённость не найдена"}
	case errors.Is(err, agreement.ErrNotOfferAuthor):
		return failure{http.StatusForbidden, "not_offer_author", "Подтвердить может только автор предложения"}
	case errors.Is(err, agreement.ErrNotParticipant):
		return failure{http.StatusForbidden, "not_participant", "Договорённость доступна только её участникам"}
	case errors.Is(err, agreement.ErrOfferUnavailable):
		return failure{http.StatusConflict, "offer_unavailable", "Предложение сейчас недоступно"}
	case errors.Is(err, domain.ErrInvalidTransition):
		return failure{http.StatusConflict, "invalid_transition", "Переход запрещён текущим состоянием"}
	case errors.Is(err, agreement.ErrDirectoryFailed):
		return failure{http.StatusBadGateway, "dependency_failed", "Зависимость вернула ошибку"}
	default:
		return failure{http.StatusInternalServerError, "internal_error", "Внутренняя ошибка"}
	}
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	f := classify(err)
	if h.Logger != nil {
		level := slog.LevelInfo
		if f.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		h.Logger.Log(r.Context(), level, "engagement request failed",
			"request_id", h.requestID(r.Context()), "code", f.code, "error", err.Error())
	}
	writeError(w, f.status, f.code, f.message)
}

func (h *Handler) requestID(ctx context.Context) string {
	if h.RequestID == nil {
		return ""
	}
	return h.RequestID(ctx)
}

type errorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorDTO{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
