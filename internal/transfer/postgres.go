package transfer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresRepository struct{ DB *sql.DB }

func NewPostgresRepository(db *sql.DB) *PostgresRepository { return &PostgresRepository{DB: db} }

func classify(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "40P01", "40001", "55P03", "57014":
			return fmt.Errorf("%w: %v", ErrRetryable, err)
		case "23505":
			if pg.ConstraintName == "reservations_pkey" {
				return fmt.Errorf("%w: %v", ErrSlotOccupied, err)
			}
			if pg.ConstraintName == "transfer_commands_pkey" || pg.ConstraintName == "transfer_commands_proposal_id_key" {
				return fmt.Errorf("%w: %v", ErrCommandConflict, err)
			}
		}
	}
	return err
}

func loadStored(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (StoredCommand, error) {
	var s StoredCommand
	err := q.QueryRowContext(ctx, `SELECT c.command_id,c.agreement_id,c.proposal_id,c.actor_id,c.expected_revision,c.result_revision,p.initiator_slot_id,p.partner_slot_id
 FROM transfer_commands c JOIN transfer_proposals p ON p.id=c.proposal_id WHERE c.command_id=$1`, id).
		Scan(&s.CommandID, &s.AgreementID, &s.ProposalID, &s.ActorID, &s.ExpectedRevision, &s.ResultRevision, &s.InitiatorSlotID, &s.PartnerSlotID)
	return s, err
}
func permitted(a Agreement, actor string) bool { return actor == a.InitiatorID || actor == a.PartnerID }
func sameCommand(s StoredCommand, c Command) bool {
	return s.AgreementID == c.AgreementID && s.ProposalID == c.ProposalID && s.ActorID == c.ActorID && s.ExpectedRevision == c.ExpectedRevision
}
func storedResult(s StoredCommand) Result {
	return Result{s.AgreementID, "active", s.ResultRevision, s.InitiatorSlotID, s.PartnerSlotID, s.CommandID}
}

func (r *PostgresRepository) Transfer(ctx context.Context, c Command) (Result, error) {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Result{}, classify(err)
	}
	defer tx.Rollback()

	var a Agreement
	err = tx.QueryRowContext(ctx, `SELECT id,initiator_id,partner_id,status,revision FROM agreements WHERE id=$1 FOR UPDATE`, c.AgreementID).
		Scan(&a.ID, &a.InitiatorID, &a.PartnerID, &a.Status, &a.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, classify(err)
	}
	if !permitted(a, c.ActorID) {
		return Result{}, ErrForbidden
	}

	previous, err := loadStored(ctx, tx, c.CommandID)
	if err == nil {
		if !sameCommand(previous, c) {
			return Result{}, ErrCommandConflict
		}
		return storedResult(previous), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Result{}, classify(err)
	}
	if a.Status != "active" {
		return Result{}, ErrInvalidTransition
	}
	if a.Revision != c.ExpectedRevision {
		return Result{}, ErrStaleRevision
	}

	var p Proposal
	err = tx.QueryRowContext(ctx, `SELECT id,agreement_id,base_revision,initiator_slot_id,partner_slot_id,initiator_approved,partner_approved,status
 FROM transfer_proposals WHERE id=$1 AND agreement_id=$2 FOR UPDATE`, c.ProposalID, c.AgreementID).
		Scan(&p.ID, &p.AgreementID, &p.BaseRevision, &p.InitiatorSlotID, &p.PartnerSlotID, &p.InitiatorApproved, &p.PartnerApproved, &p.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, classify(err)
	}
	if p.Status != "pending" {
		return Result{}, ErrInvalidTransition
	}
	if p.BaseRevision != a.Revision {
		return Result{}, ErrStaleRevision
	}
	if !p.InitiatorApproved || !p.PartnerApproved {
		return Result{}, ErrApprovalRequired
	}
	if p.InitiatorSlotID == p.PartnerSlotID {
		return Result{}, ErrInvalidRequest
	}

	rows, err := tx.QueryContext(ctx, `SELECT r.slot_id,r.participant_id FROM reservations r WHERE r.agreement_id=$1`, a.ID)
	if err != nil {
		return Result{}, classify(err)
	}
	old := map[string]string{}
	for rows.Next() {
		var slot, participant string
		if err = rows.Scan(&slot, &participant); err != nil {
			break
		}
		old[participant] = slot
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return Result{}, classify(err)
	}
	if len(old) != 2 || old[a.InitiatorID] == "" || old[a.PartnerID] == "" {
		return Result{}, ErrInvariant
	}
	if old[a.InitiatorID] == p.InitiatorSlotID && old[a.PartnerID] == p.PartnerSlotID {
		return Result{}, ErrSameSlots
	}
	if old[a.InitiatorID] == p.InitiatorSlotID || old[a.PartnerID] == p.PartnerSlotID || old[a.InitiatorID] == p.PartnerSlotID || old[a.PartnerID] == p.InitiatorSlotID {
		return Result{}, ErrPartialOverlap
	}
	ids := []string{old[a.InitiatorID], old[a.PartnerID], p.InitiatorSlotID, p.PartnerSlotID}
	sort.Strings(ids)
	for i, id := range ids {
		if i > 0 && id == ids[i-1] {
			continue
		}
		var locked string
		err = tx.QueryRowContext(ctx, `SELECT id FROM slots WHERE id=$1 FOR UPDATE`, id).Scan(&locked)
		if errors.Is(err, sql.ErrNoRows) {
			return Result{}, ErrNotFound
		}
		if err != nil {
			return Result{}, classify(err)
		}
	}
	
	for _, target := range []struct{ slot, owner string }{{p.InitiatorSlotID, a.InitiatorID}, {p.PartnerSlotID, a.PartnerID}} {
		var owner string
		if err = tx.QueryRowContext(ctx, `SELECT owner_id FROM slots WHERE id=$1`, target.slot).Scan(&owner); err != nil {
			return Result{}, classify(err)
		}
		if owner != target.owner {
			return Result{}, ErrWrongSlotOwner
		}
		var occupant string
		err = tx.QueryRowContext(ctx, `SELECT agreement_id FROM reservations WHERE slot_id=$1`, target.slot).Scan(&occupant)
		if err == nil {
			return Result{}, ErrSlotOccupied
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Result{}, classify(err)
		}
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM reservations WHERE agreement_id=$1`, a.ID)
	if err != nil {
		return Result{}, classify(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Result{}, err
	}
	if n != 2 {
		return Result{}, ErrInvariant
	}
	for _, target := range []struct{ slot, owner string }{{p.InitiatorSlotID, a.InitiatorID}, {p.PartnerSlotID, a.PartnerID}} {
		_, err = tx.ExecContext(ctx, `INSERT INTO reservations(slot_id,agreement_id,participant_id) VALUES($1,$2,$3)`, target.slot, a.ID, target.owner)
		if err != nil {
			return Result{}, classify(err)
		}
	}
	res, err = tx.ExecContext(ctx, `UPDATE agreements SET revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$2 AND status='active'`, a.ID, a.Revision)
	if err != nil {
		return Result{}, classify(err)
	}
	n, err = res.RowsAffected()
	if err != nil {
		return Result{}, err
	}
	if n != 1 {
		return Result{}, ErrStaleRevision
	}
	_, err = tx.ExecContext(ctx, `UPDATE transfer_proposals SET status='executed',executed_at=now() WHERE id=$1`, p.ID)
	if err != nil {
		return Result{}, classify(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO transfer_commands(command_id,agreement_id,proposal_id,actor_id,expected_revision,result_revision) VALUES($1,$2,$3,$4,$5,$6)`, c.CommandID, a.ID, p.ID, c.ActorID, a.Revision, a.Revision+1)
	if err != nil {
		return Result{}, classify(err)
	}
	if err = tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrCommitUnconfirmed, err)
	}
	return Result{a.ID, "active", a.Revision + 1, p.InitiatorSlotID, p.PartnerSlotID, c.CommandID}, nil
}

func (r *PostgresRepository) GetAgreement(ctx context.Context, id, actor string) (Result, error) {
	var a Agreement
	err := r.DB.QueryRowContext(ctx, `SELECT id,initiator_id,partner_id,status,revision FROM agreements WHERE id=$1`, id).Scan(&a.ID, &a.InitiatorID, &a.PartnerID, &a.Status, &a.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, classify(err)
	}
	if !permitted(a, actor) {
		return Result{}, ErrForbidden
	}
	result := Result{AgreementID: a.ID, Status: a.Status, Revision: a.Revision}
	rows, err := r.DB.QueryContext(ctx, `SELECT r.participant_id,r.slot_id FROM reservations r WHERE r.agreement_id=$1`, id)
	if err != nil {
		return Result{}, classify(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var owner, slot string
		if err = rows.Scan(&owner, &slot); err != nil {
			return Result{}, err
		}
		count++
		if owner == a.InitiatorID {
			result.InitiatorSlotID = slot
		} else if owner == a.PartnerID {
			result.PartnerSlotID = slot
		} else {
			return Result{}, ErrInvariant
		}
	}
	if err = rows.Err(); err != nil {
		return Result{}, err
	}
	if a.Status == "active" && (count != 2 || result.InitiatorSlotID == "" || result.PartnerSlotID == "") {
		return Result{}, ErrInvariant
	}
	if a.Status != "active" && count != 0 {
		return Result{}, ErrInvariant
	}
	return result, nil
}

func (r *PostgresRepository) GetCommand(ctx context.Context, id, actor string) (Result, error) {
	s, err := loadStored(ctx, r.DB, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, classify(err)
	}
	var initiator, partner string
	err = r.DB.QueryRowContext(ctx, `SELECT initiator_id,partner_id FROM agreements WHERE id=$1`, s.AgreementID).Scan(&initiator, &partner)
	if err != nil {
		return Result{}, classify(err)
	}
	if actor != initiator && actor != partner {
		return Result{}, ErrForbidden
	}
	return storedResult(s), nil
}
