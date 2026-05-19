-- +goose Up

-- Guard table for batch-level idempotency. A UNIQUE constraint on batch_id
-- prevents the TOCTOU race where two concurrent INSERT transactions both pass
-- a SELECT EXISTS check before either commits.
CREATE TABLE ingested_batches (
    batch_id   UUID        NOT NULL PRIMARY KEY,
    org_id     UUID        NOT NULL REFERENCES organizations(id),
    site_id    UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ingested_batches;
