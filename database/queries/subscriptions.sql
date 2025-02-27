-- name: CreateSubscription :one
INSERT INTO subscriptions (id, user_id, term, last_notified_at, min_price, max_price)
VALUES (?, ?, ?, 0, ?, ?)
RETURNING *;

-- name: FindSubscription :one
SELECT * FROM subscriptions
WHERE id = ?;

-- name: FindUserSubscriptions :many
SELECT * FROM subscriptions
WHERE user_id = ?;

-- name: DeleteUserSubscriptions :exec
DELETE FROM subscriptions
WHERE user_id = ? AND id IN (sqlc.slice('ids'));

-- name: FindSubscriptionsToNotify :many
SELECT * FROM subscriptions
WHERE last_notified_at < datetime('now', '-5 minutes')
LIMIT 100;

-- name: SetSubscriptionLastNotifiedAt :exec
UPDATE subscriptions
SET last_notified_at = CURRENT_TIMESTAMP
WHERE id = ?;
