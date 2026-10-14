-- The reason text is replaced with a code. The values are described by the ReasonCode constants in the code.
ALTER TABLE transfers
    DROP CONSTRAINT transfers_reason_valid,
    ADD COLUMN reason_code INTEGER;

-- If some reason has no code, reason_code stays NULL and the new constraint below fails.
UPDATE transfers
SET reason_code = CASE reason
    WHEN 'self-transfer not allowed' THEN 422001
    WHEN 'account not found' THEN 422002
    WHEN 'insufficient funds' THEN 422003
END
WHERE reason IS NOT NULL;

ALTER TABLE transfers
    DROP COLUMN reason,
    -- A rejected transfer must have a reason code. Other transfers must not have one.
    ADD CONSTRAINT transfers_reason_code_valid CHECK ((status = 'rejected') = (reason_code IS NOT NULL));

---- create above / drop below ----

ALTER TABLE transfers
    DROP CONSTRAINT transfers_reason_code_valid,
    ADD COLUMN reason TEXT;

UPDATE transfers
SET reason = CASE reason_code
    WHEN 422001 THEN 'self-transfer not allowed'
    WHEN 422002 THEN 'account not found'
    WHEN 422003 THEN 'insufficient funds'
END
WHERE reason_code IS NOT NULL;

ALTER TABLE transfers
    DROP COLUMN reason_code,
    ADD CONSTRAINT transfers_reason_valid CHECK ((status = 'rejected') = (reason IS NOT NULL));