-- name: CreateItem :one
INSERT INTO items (id, subscription_id, goodwill_id, created_at, started_at, ends_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?)
RETURNING *;

-- name: IsItemTracked :one
SELECT EXISTS (
  SELECT 1
  FROM items
  WHERE subscription_id = ? AND goodwill_id = ?
) AS is_tracked;

-- name: FindItemsEndingSoon :many
SELECT i.* FROM items i
JOIN subscriptions s ON i.subscription_id = s.id
WHERE i.ends_at < datetime('now', '+' || s.notify_minutes || ' minutes') AND i.sent_final = FALSE
LIMIT 100;

-- name: SetItemSentFinal :exec
UPDATE items
SET sent_final = TRUE
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteExpiredItems :exec
DELETE FROM items
WHERE ends_at < datetime('now', '-1 day');

-- name: DeleteItemsInSubscriptions :exec
DELETE FROM items
WHERE subscription_id IN (sqlc.slice('ids'));
