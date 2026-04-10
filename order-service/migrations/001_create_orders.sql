CREATE TABLE IF NOT EXISTS orders (
    id              TEXT PRIMARY KEY,
    customer_id     TEXT        NOT NULL,
    item_name       TEXT        NOT NULL,
    amount          BIGINT      NOT NULL,
    status          TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
