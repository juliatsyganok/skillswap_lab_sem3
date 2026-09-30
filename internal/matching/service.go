package matching

import "fmt"

type Service struct {
	reader ProfileReader
}

func NewService(reader ProfileReader) *Service {
	return &Service{reader: reader}
}

func (s *Service) FindMatchFor(userID string) ([]string, error) {
	subject, err := s.reader.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}
	candidates, err := s.reader.List()
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	return FindMatch(subject, candidates), nil
}
