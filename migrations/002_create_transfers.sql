CREATE TABLE transfers (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    idempotency_key TEXT NOT NULL,
    from_account_id BIGINT NOT NULL REFERENCES accounts (id),
    to_account_id   BIGINT NOT NULL REFERENCES accounts (id),
    amount          BIGINT NOT NULL,
    status          TEXT NOT NULL,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT transfers_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT transfers_amount_positive CHECK (amount > 0),
    CONSTRAINT transfers_not_self CHECK (from_account_id <> to_account_id),
    CONSTRAINT transfers_status_valid CHECK (status IN ('completed', 'rejected')),
    -- A rejected transfer must have a reason. A completed transfer must not have one.
    CONSTRAINT transfers_reason_valid CHECK ((status = 'rejected') = (reason IS NOT NULL))
);

---- create above / drop below ----

DROP TABLE transfers;