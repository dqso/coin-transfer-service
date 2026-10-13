CREATE TYPE transfer_status AS ENUM ('in_progress', 'completed', 'rejected');

CREATE TABLE transfers (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    idempotency_key   TEXT NOT NULL,
    -- If the client sends an unknown account, the id is NULL and the code from the request is saved instead.
    from_account_id   BIGINT REFERENCES accounts (id),
    from_account_code TEXT,
    to_account_id     BIGINT REFERENCES accounts (id),
    to_account_code   TEXT,
    amount            BIGINT NOT NULL,
    status            transfer_status NOT NULL,
    reason            TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT transfers_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT transfers_amount_positive CHECK (amount > 0),
    -- Each side has either an account id or an unknown account code, not both.
    CONSTRAINT transfers_from_account_valid CHECK ((from_account_id IS NULL) <> (from_account_code IS NULL)),
    CONSTRAINT transfers_to_account_valid CHECK ((to_account_id IS NULL) <> (to_account_code IS NULL)),
    -- A rejected transfer must have a reason. Other transfers must not have one.
    CONSTRAINT transfers_reason_valid CHECK ((status = 'rejected') = (reason IS NOT NULL))
);

---- create above / drop below ----

DROP TABLE transfers;
DROP TYPE transfer_status;