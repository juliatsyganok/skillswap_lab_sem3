package matching

import "fmt"

type Service struct {
	reader     ProfileReader
	agreements AgreementReader
}

func NewService(reader ProfileReader, agreements AgreementReader) *Service {
	return &Service{reader: reader, agreements: agreements}
}

// FindMatchFor возвращает доступные пары: взаимно совместимых кандидатов
// без тех, с кем договорённость уже подтверждена.
func (s *Service) FindMatchFor(userID string) ([]string, error) {
	subject, err := s.reader.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}
	candidates, err := s.reader.List()
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	offers, err := s.agreements.GetOffers()
	if err != nil {
		return nil, fmt.Errorf("get offers: %w", err)
	}
	responses, err := s.agreements.ListResponses()
	if err != nil {
		return nil, fmt.Errorf("list responses: %w", err)
	}
	partners := ConfirmedPartners(subject.ID, offers, responses)
	return ExcludePartners(FindMatch(subject, candidates), partners), nil
}
