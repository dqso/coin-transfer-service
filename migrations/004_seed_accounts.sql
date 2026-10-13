-- The treasury account holds all coins of the system at the start.
-- Players get their start balance from it through normal transfers.
INSERT INTO accounts (code, balance)
VALUES ('treasury', 10000);

INSERT INTO accounts (code)
SELECT 'acc_' || n
FROM generate_series(1, 10) AS n;

-- One transfer of 1000 coins from the treasury to each player.
INSERT INTO transfers (idempotency_key, from_account_id, to_account_id, amount, status)
SELECT 'seed:' || a.code, t.id, a.id, 1000, 'completed'
FROM accounts a
CROSS JOIN (SELECT id FROM accounts WHERE code = 'treasury') AS t
WHERE a.code LIKE 'acc\_%'
ORDER BY a.id;

-- Each transfer has two postings: one takes coins from the treasury, one adds them to the player.
INSERT INTO postings (transfer_id, account_id, amount)
SELECT t.id, p.account_id, p.amount
FROM transfers t
CROSS JOIN LATERAL (
    VALUES (t.from_account_id, -t.amount), (t.to_account_id, t.amount)
) AS p (account_id, amount)
WHERE t.idempotency_key LIKE 'seed:%'
ORDER BY t.id;

-- Apply the postings to the balances.
UPDATE accounts a
SET balance = a.balance + p.total,
    updated_at = now()
FROM (
    SELECT account_id, sum(amount) AS total
    FROM postings
    GROUP BY account_id
) AS p
WHERE p.account_id = a.id;

---- create above / drop below ----

DELETE FROM postings
WHERE transfer_id IN (SELECT id FROM transfers WHERE idempotency_key LIKE 'seed:%');

DELETE FROM transfers
WHERE idempotency_key LIKE 'seed:%';

DELETE FROM accounts
WHERE code = 'treasury' OR code LIKE 'acc\_%';