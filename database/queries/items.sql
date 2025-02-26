-- name: CreateItem :one
INSERT INTO items (id, subscription_id, goodwill_id, created_at, auction_end_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
RETURNING *;

-- name: FindItemInSubscription :one
SELECT * FROM items
WHERE subscription_id = ? AND goodwill_id = ?;

-- name: FindItemsEndingSoon :many
SELECT * FROM items
WHERE auction_end_at < ?;
