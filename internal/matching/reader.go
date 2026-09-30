package matching

import "skillswap/domain"

type ProfileReader interface {
	Get(id string) (domain.User, error)
	List() ([]domain.User, error)
}
