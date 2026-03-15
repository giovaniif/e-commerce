CREATE TABLE IF NOT EXISTS items (
    id INT PRIMARY KEY,
    price DECIMAL(10,2) NOT NULL,
    initial_stock BIGINT NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS reservation_id_seq;

CREATE TABLE IF NOT EXISTS stock_events (
    id BIGSERIAL PRIMARY KEY,
    reservation_id BIGINT NOT NULL,
    item_id INT NOT NULL REFERENCES items(id),
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('reserved', 'released', 'completed')),
    quantity INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_events_item_id ON stock_events(item_id);
CREATE INDEX IF NOT EXISTS idx_stock_events_reservation_id ON stock_events(reservation_id);

INSERT INTO items (id, price, initial_stock) VALUES
    (1,  10.00, 1000000000),
    (2,  25.00, 1000000000),
    (3,  49.99, 1000000000),
    (4,   5.00, 1000000000),
    (5,  99.99, 1000000000),
    (6,  15.00, 1000000000),
    (7,  30.00, 1000000000),
    (8,  75.00, 1000000000),
    (9,   8.50, 1000000000),
    (10, 19.99, 1000000000)
ON CONFLICT DO NOTHING;
