package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"skillswap/internal/transfer"
)

type TransferService interface {
	Complete(context.Context, transfer.CompletionCommand) (transfer.CompletionResult, error)
	Transfer(context.Context, transfer.Command) (transfer.Result, error)
	GetAgreement(context.Context, string, string) (transfer.Result, error)
	GetCommand(context.Context, string, string) (transfer.Result, error)
}

type TransferHandler struct {
	Service  TransferService
	Actor    func(context.Context) (string, bool)
	Timeout  time.Duration
	LogError func(context.Context, error)
}

func NewTransferHandler(s TransferService, actor func(context.Context) (string, bool)) *TransferHandler {
	return &TransferHandler{Service: s, Actor: actor, Timeout: 5 * time.Second}
}
func (h *TransferHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/agreements/{id}/transfers", h.transfer)
	mux.HandleFunc("POST /v1/agreements/{id}/completion", h.complete)
	mux.HandleFunc("GET /v1/agreements/{id}", h.agreement)
	mux.HandleFunc("GET /v1/transfer-commands/{command_id}", h.command)
}
func (h *TransferHandler) actor(w http.ResponseWriter, r *http.Request) (string, bool) {
	if h.Actor == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "actor resolver is not configured")
		return "", false
	}
	id, ok := h.Actor(r.Context())
	if !ok || id == "" {
		writeError(w, http.StatusForbidden, "forbidden", "actor is required")
		return "", false
	}
	return id, true
}
func (h *TransferHandler) context(r *http.Request) (context.Context, context.CancelFunc) {
	d := h.Timeout
	if d <= 0 {
		d = 5 * time.Second
	}
	return context.WithTimeout(r.Context(), d)
}
func (h *TransferHandler) transfer(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var cmd transfer.Command
	if !decodeAgreementCommand(w, r, &cmd) {
		return
	}
	cmd.AgreementID = r.PathValue("id")
	cmd.ActorID = actor
	ctx, cancel := h.context(r)
	defer cancel()
	result, err := h.Service.Transfer(ctx, cmd)
	if err != nil {
		h.writeOperationError(ctx, w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (h *TransferHandler) complete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var cmd transfer.CompletionCommand
	if !decodeAgreementCommand(w, r, &cmd) {
		return
	}
	cmd.AgreementID, cmd.ActorID = r.PathValue("id"), actor
	ctx, cancel := h.context(r)
	defer cancel()
	result, err := h.Service.Complete(ctx, cmd)
	if err != nil {
		h.writeOperationError(ctx, w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeAgreementCommand(w http.ResponseWriter, r *http.Request, payload any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "expected exactly one JSON object")
		return false
	}
	return true
}

func (h *TransferHandler) agreement(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	ctx, cancel := h.context(r)
	defer cancel()
	result, err := h.Service.GetAgreement(ctx, r.PathValue("id"), actor)
	if err != nil {
		h.writeOperationError(ctx, w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (h *TransferHandler) command(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	ctx, cancel := h.context(r)
	defer cancel()
	result, err := h.Service.GetCommand(ctx, r.PathValue("command_id"), actor)
	if err != nil {
		h.writeOperationError(ctx, w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (h *TransferHandler) writeOperationError(ctx context.Context, w http.ResponseWriter, err error) {
	logError := h.LogError
	if logError == nil {
		logError = func(ctx context.Context, err error) {
			slog.ErrorContext(ctx, "agreement operation failed", "error", err)
		}
	}
	switch {
	case errors.Is(err, transfer.ErrInvalidRequest):
		writeError(w, 400, "invalid_request", "invalid agreement request")
	case errors.Is(err, transfer.ErrNotFound):
		writeError(w, 404, "not_found", "resource not found")
	case errors.Is(err, transfer.ErrForbidden):
		writeError(w, 403, "forbidden", "operation is not permitted")
	case errors.Is(err, transfer.ErrApprovalRequired):
		writeError(w, 409, "approval_required", "both participants must approve")
	case errors.Is(err, transfer.ErrStaleRevision):
		writeError(w, 409, "stale_revision", "agreement revision changed")
	case errors.Is(err, transfer.ErrSlotOccupied):
		writeError(w, 409, "slot_occupied", "target slot is occupied")
	case errors.Is(err, transfer.ErrWrongSlotOwner):
		writeError(w, 409, "wrong_slot_owner", "slot owner does not match participant")
	case errors.Is(err, transfer.ErrPartialOverlap):
		writeError(w, 409, "partial_overlap", "partial slot overlap is not supported")
	case errors.Is(err, transfer.ErrSameSlots):
		writeError(w, 409, "same_slots", "slots are unchanged")
	case errors.Is(err, transfer.ErrCommandConflict):
		writeError(w, 409, "command_conflict", "command ID is already used")
	case errors.Is(err, transfer.ErrInvalidTransition):
		writeError(w, 409, "invalid_transition", "agreement or proposal does not allow the operation")
	case errors.Is(err, transfer.ErrRetryable):
		logError(ctx, err)
		writeError(w, 503, "transaction_retryable", "temporary database conflict")
	case errors.Is(err, transfer.ErrCommitUnconfirmed):
		logError(ctx, err)
		writeError(w, 503, "commit_unconfirmed", "check the saved state before retrying")
	default:
		logError(ctx, err)
		writeError(w, 500, "internal_error", "internal server error")
	}
}
