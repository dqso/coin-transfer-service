CREATE TABLE accounts (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       TEXT NOT NULL,
    balance    BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT accounts_code_unique UNIQUE (code),
    CONSTRAINT accounts_balance_not_negative CHECK (balance >= 0)
);

---- create above / drop below ----

DROP TABLE accounts;