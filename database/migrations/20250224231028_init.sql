-- +goose Up
-- +goose StatementBegin
CREATE TABLE subscriptions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  term TEXT NOT NULL,
  min_price INTEGER,
  max_price INTEGER,
  category_id INTEGER,
  last_notified_at DATETIME NOT NULL
);
CREATE INDEX idx_last_notified_at ON subscriptions(last_notified_at);
CREATE UNIQUE INDEX idx_user_id_term ON subscriptions(user_id, term);

CREATE TABLE items (
  id TEXT PRIMARY KEY,
  subscription_id TEXT NOT NULL,
  goodwill_id INTEGER NOT NULL,
  created_at DATETIME NOT NULL,
  started_at DATETIME NOT NULL,
  ends_at DATETIME NOT NULL,
  sent_final BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_subscription_id_goodwill_id ON items(subscription_id, goodwill_id);
CREATE INDEX idx_ends_at_sent_final ON items(ends_at, sent_final);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_last_notified_at;
DROP INDEX IF EXISTS idx_user_id_term;
DROP TABLE IF EXISTS subscriptions;

DROP INDEX IF EXISTS idx_subscription_id;
DROP INDEX IF EXISTS idx_goodwill_id;
DROP INDEX IF EXISTS idx_created_at;
DROP INDEX IF EXISTS idx_ends_at;
DROP TABLE IF EXISTS items;
-- +goose StatementEnd
