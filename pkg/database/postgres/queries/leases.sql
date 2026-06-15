-- name: FindLeaseById :one
SELECT id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at
FROM leases 
WHERE id = @lease_id;

-- name: InsertLease :one
INSERT INTO leases (id, identifier, cluster_id, acquired_at, renewed_at)
VALUES (@id, @identifier, @cluster_id, @acquired_at, @renewed_at)
RETURNING id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at;

-- name: UpdateLease :one
UPDATE leases 
SET 
    identifier = @identifier,
    cluster_id = @cluster_id,
    acquired_at = @acquired_at,
    renewed_at = @renewed_at,
    updated_at = NOW()
WHERE id = @id
  -- Only renew/take over if, at write time, the lease is still ours or has expired.
  -- This makes the update atomic with the ownership/expiry decision and prevents a
  -- stalled holder from overwriting a newer leader after failover.
  AND (identifier = @identifier OR renewed_at < @stale_threshold)
RETURNING id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at;
