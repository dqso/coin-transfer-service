-- The worker looks for transfers that stay in progress for too long.
-- There are only a few such transfers, so a partial index is small.
CREATE INDEX transfers_in_progress_updated_at_idx ON transfers (updated_at) WHERE status = 'in_progress';

---- create above / drop below ----

DROP INDEX transfers_in_progress_updated_at_idx;