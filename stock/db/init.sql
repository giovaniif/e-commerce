CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY,
    price DECIMAL(10,2) NOT NULL,
    initial_stock INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS reservations (
    id SERIAL PRIMARY KEY,
    item_id INTEGER NOT NULL REFERENCES items(id),
    total_fee DECIMAL(10,2) NOT NULL,
    quantity INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'reserved'
);

INSERT INTO items (id, price, initial_stock) VALUES (1, 10.00, 10)
ON CONFLICT (id) DO NOTHING;
