BEGIN;

CREATE TABLE agreements (
    id text PRIMARY KEY,
    initiator_id text NOT NULL,
    partner_id text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT agreement_participants CHECK (initiator_id <> partner_id),
    CONSTRAINT agreement_status CHECK (status IN ('active', 'canceled'))
);

CREATE TABLE slots (
    id text PRIMARY KEY,
    owner_id text NOT NULL,
    UNIQUE (id, owner_id)
);

CREATE TABLE reservations (
    slot_id text PRIMARY KEY,
    agreement_id text NOT NULL REFERENCES agreements (id),
    participant_id text NOT NULL,
    FOREIGN KEY (slot_id, participant_id) REFERENCES slots (id, owner_id),
    UNIQUE (agreement_id, participant_id)
);

CREATE TABLE transfer_proposals (
    id text PRIMARY KEY,
    agreement_id text NOT NULL REFERENCES agreements (id),
    base_revision bigint NOT NULL CHECK (base_revision > 0),
    initiator_slot_id text NOT NULL REFERENCES slots (id),
    partner_slot_id text NOT NULL REFERENCES slots (id),
    initiator_approved boolean NOT NULL DEFAULT false,
    partner_approved boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'executed')),
    executed_at timestamptz,
    UNIQUE (id, agreement_id),
    CHECK (initiator_slot_id <> partner_slot_id)
);

CREATE TABLE transfer_commands (
    command_id text PRIMARY KEY,
    agreement_id text NOT NULL,
    proposal_id text NOT NULL UNIQUE,
    actor_id text NOT NULL,
    expected_revision bigint NOT NULL CHECK (expected_revision > 0),
    result_revision bigint NOT NULL CHECK (result_revision = expected_revision + 1),
    FOREIGN KEY (proposal_id, agreement_id) REFERENCES transfer_proposals (id, agreement_id)
);

COMMIT;
