-- The treasury account holds all coins of the system at the start.
-- Players get their start balance from it through postings.
INSERT INTO accounts (code, balance)
VALUES ('treasury', 10000);

INSERT INTO accounts (code)
SELECT 'acc_' || n
FROM generate_series(1, 10) AS n;

-- Give each player 1000 coins: one posting takes coins from the treasury, one posting adds them to the player.
INSERT INTO postings (account_id, amount)
SELECT a.id, p.amount
FROM generate_series(1, 10) AS n
CROSS JOIN LATERAL (
    VALUES ('treasury', -1000), ('acc_' || n, 1000)
) AS p (code, amount)
JOIN accounts a ON a.code = p.code
ORDER BY n;

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
WHERE transfer_id IS NULL;

DELETE FROM accounts
WHERE code = 'treasury' OR code LIKE 'acc\_%';