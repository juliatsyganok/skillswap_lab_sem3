package transfer

import (
	"context"
	"errors"
	"testing"
)

type stubRepo struct {
	calls  int
	result Result
	err    error
}

func (s *stubRepo) Transfer(_ context.Context, _ Command) (Result, error) {
	s.calls++
	return s.result, s.err
}
func (s *stubRepo) GetAgreement(_ context.Context, _, _ string) (Result, error) {
	return s.result, s.err
}
func (s *stubRepo) GetCommand(_ context.Context, _, _ string) (Result, error) { return s.result, s.err }
func TestTransferValidation(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(repo)
	_, err := svc.Transfer(context.Background(), Command{ActorID: "u1", AgreementID: "a1", CommandID: "cmd1", ProposalID: "p1"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatal("repository called for invalid command")
	}
}
func TestTransferCallsRepository(t *testing.T) {
	repo := &stubRepo{result: Result{AgreementID: "a1", Revision: 2}}
	result, err := NewService(repo).Transfer(context.Background(), Command{ActorID: "u1", AgreementID: "a1", CommandID: "cmd1", ProposalID: "p1", ExpectedRevision: 1})
	if err != nil || result.Revision != 2 || repo.calls != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, repo.calls)
	}
}
