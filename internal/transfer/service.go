package transfer

import (
	"context"
	"strings"
)

type Repository interface {
	Transfer(ctx context.Context, cmd Command) (Result, error)
	GetAgreement(ctx context.Context, agreementID, actorID string) (Result, error)
	GetCommand(ctx context.Context, commandID, actorID string) (Result, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Transfer(ctx context.Context, cmd Command) (Result, error) {
	if strings.TrimSpace(cmd.ActorID) == "" || strings.TrimSpace(cmd.AgreementID) == "" || strings.TrimSpace(cmd.CommandID) == "" || strings.TrimSpace(cmd.ProposalID) == "" || cmd.ExpectedRevision < 1 {
		return Result{}, ErrInvalidRequest
	}
	return s.repo.Transfer(ctx, cmd)
}
func (s *Service) GetAgreement(ctx context.Context, id, actor string) (Result, error) {
	if id == "" || actor == "" {
		return Result{}, ErrInvalidRequest
	}
	return s.repo.GetAgreement(ctx, id, actor)
}
func (s *Service) GetCommand(ctx context.Context, id, actor string) (Result, error) {
	if id == "" || actor == "" {
		return Result{}, ErrInvalidRequest
	}
	return s.repo.GetCommand(ctx, id, actor)
}
