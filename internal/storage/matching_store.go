package storage

import "skillswap/domain"

type InMemoryMatchingStore struct {
	users map[string]domain.User
}

func NewInMemoryMatchingStore() *InMemoryMatchingStore {
	return &InMemoryMatchingStore{
		users: make(map[string]domain.User),
	}
}

func (s *InMemoryMatchingStore) AddUser(u domain.User) {
	s.users[u.ID] = u
}

func (s *InMemoryMatchingStore) Get(id string) (domain.User, error) {
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (s *InMemoryMatchingStore) List() ([]domain.User, error) {
	result := make([]domain.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result, nil
}
