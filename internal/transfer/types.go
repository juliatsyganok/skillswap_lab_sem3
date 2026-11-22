package transfer

import "errors"

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidTransition = errors.New("invalid transition")
	ErrApprovalRequired  = errors.New("approval required")
	ErrStaleRevision     = errors.New("stale revision")
	ErrSlotOccupied      = errors.New("slot occupied")
	ErrWrongSlotOwner    = errors.New("wrong slot owner")
	ErrPartialOverlap    = errors.New("partial overlap")
	ErrSameSlots         = errors.New("same slots")
	ErrCommandConflict   = errors.New("command conflict")
	ErrRetryable         = errors.New("transaction retryable")
	ErrCommitUnconfirmed = errors.New("commit unconfirmed")
	ErrInvariant         = errors.New("invariant violation")
)

type Command struct {
	AgreementID      string `json:"-"`
	ActorID          string `json:"-"`
	CommandID        string `json:"command_id"`
	ProposalID       string `json:"proposal_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}

type Result struct {
	AgreementID     string `json:"agreement_id"`
	Status          string `json:"status"`
	Revision        int64  `json:"revision"`
	InitiatorSlotID string `json:"initiator_slot_id"`
	PartnerSlotID   string `json:"partner_slot_id"`
	CommandID       string `json:"command_id"`
}

type Agreement struct {
	ID, InitiatorID, PartnerID, Status string
	Revision                           int64
}

type Proposal struct {
	ID, AgreementID, InitiatorSlotID, PartnerSlotID, Status string
	BaseRevision                                            int64
	InitiatorApproved, PartnerApproved                      bool
}

type StoredCommand struct {
	CommandID, AgreementID, ProposalID, ActorID string
	ExpectedRevision, ResultRevision            int64
	InitiatorSlotID, PartnerSlotID              string
}
