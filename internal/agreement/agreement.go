// Package agreement реализует фиксацию результата согласования (контракт D):
// единственное место, где меняются статусы Offer и Response.
package agreement

import (
	"errors"
	"fmt"

	"skillswap/domain"
)

var (
	// ErrNotOfferAuthor — действие над предложением пытается выполнить не его автор.
	ErrNotOfferAuthor = errors.New("only offer author can do this")
	// ErrOfferMismatch — отклик относится к другому предложению.
	ErrOfferMismatch = errors.New("response does not belong to offer")
)

// offerTransitions: active ⇄ paused, active/paused → closed. Из closed — никуда.
var offerTransitions = map[domain.OfferStatus][]domain.OfferStatus{
	domain.OfferActive: {domain.OfferPaused, domain.OfferClosed},
	domain.OfferPaused: {domain.OfferActive, domain.OfferClosed},
}

// responseTransitions: pending → accepted | rejected. Оба итоговых статуса финальны.
var responseTransitions = map[domain.ResponseStatus][]domain.ResponseStatus{
	domain.ResponsePending: {domain.ResponseAccepted, domain.ResponseRejected},
}

// TransitionOffer проверяет, допустим ли переход статуса предложения cur → next.
func TransitionOffer(cur, next domain.OfferStatus) error {
	for _, allowed := range offerTransitions[cur] {
		if allowed == next {
			return nil
		}
	}
	return fmt.Errorf("offer %s -> %s: %w", cur, next, domain.ErrInvalidTransition)
}

// TransitionResponse проверяет, допустим ли переход статуса отклика cur → next.
func TransitionResponse(cur, next domain.ResponseStatus) error {
	for _, allowed := range responseTransitions[cur] {
		if allowed == next {
			return nil
		}
	}
	return fmt.Errorf("response %s -> %s: %w", cur, next, domain.ErrInvalidTransition)
}

// Decide фиксирует решение автора предложения по отклику.
// Возвращает изменённую копию отклика; входные значения не меняются.
// Решать можно по активному или приостановленному предложению, но не по закрытому.
func Decide(actorID string, offer domain.Offer, resp domain.Response, decision domain.ResponseStatus) (domain.Response, error) {
	if resp.OfferID != offer.ID {
		return domain.Response{}, fmt.Errorf("response %s, offer %s: %w", resp.ID, offer.ID, ErrOfferMismatch)
	}
	if offer.AuthorID != actorID {
		return domain.Response{}, fmt.Errorf("decide on offer %s: %w", offer.ID, ErrNotOfferAuthor)
	}
	if offer.Status == domain.OfferClosed {
		return domain.Response{}, fmt.Errorf("offer %s is closed: %w", offer.ID, domain.ErrInvalidTransition)
	}
	if err := TransitionResponse(resp.Status, decision); err != nil {
		return domain.Response{}, err
	}
	resp.Status = decision
	return resp, nil
}

// ChangeOfferStatus меняет статус предложения от имени его автора.
// Возвращает изменённую копию предложения; входное значение не меняется.
func ChangeOfferStatus(actorID string, offer domain.Offer, next domain.OfferStatus) (domain.Offer, error) {
	if offer.AuthorID != actorID {
		return domain.Offer{}, fmt.Errorf("change offer %s: %w", offer.ID, ErrNotOfferAuthor)
	}
	if err := TransitionOffer(offer.Status, next); err != nil {
		return domain.Offer{}, err
	}
	offer.Status = next
	return offer, nil
}
