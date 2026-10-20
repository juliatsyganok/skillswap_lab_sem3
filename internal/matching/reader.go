package matching

import "skillswap/domain"

type ProfileReader interface {
	Get(id string) (domain.User, error)
	List() ([]domain.User, error)
}

// AgreementReader — чтение общего состояния договорённостей:
// предложения нужны, чтобы по отклику найти автора пары.
type AgreementReader interface {
	GetOffers() ([]domain.Offer, error)
	ListResponses() ([]domain.Response, error)
}
