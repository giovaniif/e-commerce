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

CREATE TABLE IF NOT EXISTS processed_events (
    event_id     UUID        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
