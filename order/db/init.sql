CREATE TABLE IF NOT EXISTS orders (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    item_id         INT          NOT NULL,
    quantity        INT          NOT NULL,
    total_fee       DECIMAL(12,2),
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbox (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50)  NOT NULL,
    aggregate_id   VARCHAR(255) NOT NULL,
    type           VARCHAR(100) NOT NULL,
    payload        JSONB        NOT NULL,
    traceparent    VARCHAR(55)  NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    processed      BOOLEAN      NOT NULL DEFAULT FALSE
);

ALTER TABLE outbox REPLICA IDENTITY FULL;

CREATE INDEX IF NOT EXISTS idx_outbox_processed ON outbox(processed) WHERE processed = FALSE;
