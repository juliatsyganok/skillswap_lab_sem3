package response


import (
	"errors"
	"testing"

	"skillswap/domain"
)


type failingReader struct{}

func (f *failingReader) GetOffer(offerID string) (domain.Offer, error) {
	return domain.Offer{}, domain.ErrStorageUnavailable
}

func (f *failingReader) GetResponsesByUser(userID string) ([]domain.Response, error) {
	return nil, domain.ErrStorageUnavailable
}