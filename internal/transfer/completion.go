package transfer

import (
	"context"
	"database/sql"
	"fmt"
)

type CompletionCommand struct {
	AgreementID string `json:"-"`
	ActorID     string `json:"-"`
	Summary     string `json:"summary"`
	Rating      int    `json:"rating"`
}

type Completion struct {
	Summary string `json:"summary"`
	Rating  int    `json:"rating"`
}

type CompletionResult struct {
	Result
	Changed bool `json:"changed"`
}

func (r *PostgresRepository) Complete(ctx context.Context, cmd CompletionCommand) (CompletionResult, error) {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return CompletionResult{}, classify(err)
	}
	defer tx.Rollback()

	a, err := lockAgreement(ctx, tx, cmd.AgreementID, cmd.ActorID)
	if err != nil {
		return CompletionResult{}, err
	}
	if a.Status == "completed" {
		saved := Completion{}
		err := tx.QueryRowContext(ctx, `SELECT result_summary, result_rating FROM agreements WHERE id = $1`, a.ID).
			Scan(&saved.Summary, &saved.Rating)
		if err != nil {
			return CompletionResult{}, classify(err)
		}
		return CompletionResult{Result: Result{
			AgreementID: a.ID, Status: a.Status, Revision: a.Revision, Completion: &saved,
		}}, nil
	}
	if a.Status != "active" {
		return CompletionResult{}, ErrInvalidTransition
	}

	reservations, err := currentReservations(ctx, tx, a)
	if err != nil {
		return CompletionResult{}, err
	}
	if err := lockSlots(ctx, tx, reservations[a.InitiatorID], reservations[a.PartnerID]); err != nil {
		return CompletionResult{}, err
	}
	if err := releaseReservations(ctx, tx, a.ID); err != nil {
		return CompletionResult{}, err
	}

	var revision int64
	err = tx.QueryRowContext(ctx, `
UPDATE agreements
SET status = 'completed', revision = revision + 1, result_summary = $2, result_rating = $3, updated_at = now()
WHERE id = $1 RETURNING revision`, a.ID, cmd.Summary, cmd.Rating).Scan(&revision)
	if err != nil {
		return CompletionResult{}, classify(err)
	}
	if err := tx.Commit(); err != nil {
		return CompletionResult{}, fmt.Errorf("%w: %w", ErrCommitUnconfirmed, err)
	}
	return CompletionResult{
		Result: Result{
			AgreementID: a.ID, Status: "completed", Revision: revision,
			Completion: &Completion{Summary: cmd.Summary, Rating: cmd.Rating},
		},
		Changed: true,
	}, nil
}
