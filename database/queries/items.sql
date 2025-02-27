-- name: CreateItem :one
INSERT INTO items (id, subscription_id, goodwill_id, created_at, started_at, ends_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?)
RETURNING *;

-- name: FindItemInSubscription :one
SELECT * FROM items
WHERE subscription_id = ? AND goodwill_id = ?;

-- name: FindItemsEndingSoon :many
SELECT * FROM items
WHERE ends_at < datetime('now', '+5 minutes') AND sent_final = FALSE
LIMIT 100;

-- name: SetItemSentFinal :exec
UPDATE items
SET sent_final = TRUE
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteExpiredItems :one
DELETE FROM items
WHERE ends_at < datetime('now', '-1 day')
LIMIT 1000
RETURNING COUNT(*);
