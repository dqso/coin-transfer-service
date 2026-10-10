-- A posting changes the balance of one account.
-- A negative amount takes coins from the account, a positive amount adds coins to it.
-- All postings of one operation must add up to zero.
CREATE TABLE postings (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- NULL for internal operations that do not come from the API, for example the seed.
    transfer_id BIGINT REFERENCES transfers (id),
    account_id  BIGINT NOT NULL REFERENCES accounts (id),
    amount      BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT postings_amount_not_zero CHECK (amount <> 0)
);

CREATE INDEX postings_account_id_idx ON postings (account_id);

---- create above / drop below ----

DROP TABLE postings;