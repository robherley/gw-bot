-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscriptions
ADD COLUMN notify_minutes INTEGER NOT NULL DEFAULT 10;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscriptions
DROP COLUMN notify_minutes;
-- +goose StatementEnd
