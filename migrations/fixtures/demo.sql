BEGIN;

INSERT INTO agreements (id, initiator_id, partner_id) VALUES
    ('agreement-1', 'author', 'learner'), ('agreement-2', 'author', 'mentor');
INSERT INTO slots (id, owner_id) VALUES
    ('slot-1', 'author'), ('slot-2', 'learner'), ('slot-3', 'author'), ('slot-4', 'learner'),
    ('slot-5', 'author'), ('slot-6', 'mentor');
INSERT INTO reservations (slot_id, agreement_id, participant_id) VALUES
    ('slot-1', 'agreement-1', 'author'), ('slot-2', 'agreement-1', 'learner'),
    ('slot-5', 'agreement-2', 'author'), ('slot-6', 'agreement-2', 'mentor');
INSERT INTO transfer_proposals (id, agreement_id, base_revision, initiator_slot_id, partner_slot_id, initiator_approved, partner_approved)
VALUES ('proposal-1', 'agreement-1', 1, 'slot-3', 'slot-4', true, true);

COMMIT;
