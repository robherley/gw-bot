-- name: CreateSubscription :one
INSERT INTO subscriptions (id, user_id, term, last_notified_at, min_price, max_price)
VALUES (?, ?, ?, 0, ?, ?)
RETURNING *;

-- name: FindUserSubscriptions :many
SELECT * FROM subscriptions
WHERE user_id = ?;

-- name: DeleteUserSubscriptions :exec
DELETE FROM subscriptions
WHERE user_id = ? AND id IN (sqlc.slice('ids'));
