package storage

import (
	"slices"
	"sync"

	"skillswap/domain"
)

type InMemoryMatchingStore struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewInMemoryMatchingStore() *InMemoryMatchingStore {
	return &InMemoryMatchingStore{
		users: make(map[string]domain.User),
	}
}

func (s *InMemoryMatchingStore) AddUser(u domain.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.ID] = cloneProfile(u)
}

func (s *InMemoryMatchingStore) Get(id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return cloneProfile(user), nil
}

func (s *InMemoryMatchingStore) List() ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, cloneProfile(u))
	}
	return result, nil
}

func cloneProfile(user domain.User) domain.User {
	user.Offers = slices.Clone(user.Offers)
	user.Needs = slices.Clone(user.Needs)
	return user
}
