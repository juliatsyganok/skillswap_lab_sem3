package agreement

import "skillswap/domain"

// Store — то, что контракту D нужно от хранилища: прочитать сущность и сохранить новый статус.
type Store interface {
	GetOffer(id string) (domain.Offer, error)
	GetResponse(id string) (domain.Response, error)
	SaveOffer(offer domain.Offer) error
	SaveResponse(resp domain.Response) error
}
