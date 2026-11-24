BEGIN;

ALTER TABLE agreements
    ADD COLUMN result_summary text,
    ADD COLUMN result_rating smallint,
    DROP CONSTRAINT agreement_status,
    ADD CONSTRAINT agreement_status CHECK (status IN ('active', 'canceled', 'completed')),
    ADD CONSTRAINT agreement_completion CHECK (
        (status = 'completed' AND result_summary IS NOT NULL AND result_rating IS NOT NULL
            AND char_length(btrim(result_summary)) BETWEEN 1 AND 1000 AND result_rating BETWEEN 1 AND 5)
        OR (status <> 'completed' AND result_summary IS NULL AND result_rating IS NULL)
    );

COMMIT;
