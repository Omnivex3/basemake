-- Demodb: e-commerce schema for basemake demo & GIF
-- Realistic enough to show the value, simple enough to follow

DROP DATABASE IF EXISTS demodb;
CREATE DATABASE demodb;

\c demodb

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    plan VARCHAR(20) DEFAULT 'free',
    country VARCHAR(2) DEFAULT 'US'
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50),
    stock INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    product_id INT REFERENCES products(id),
    quantity INT NOT NULL DEFAULT 1,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    ordered_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_product ON orders(product_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_ordered_at ON orders(ordered_at);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_products_category ON products(category);

-- Sample users (mix of free/paid, recent/old)
INSERT INTO users (name, email, created_at, plan, country) VALUES
    ('Alice Mokoena', 'alice@example.com', NOW() - INTERVAL '2 days', 'pro', 'ZA'),
    ('Bob Smith', 'bob@example.com', NOW() - INTERVAL '3 days', 'free', 'US'),
    ('Carol Ndlovu', 'carol@example.com', NOW() - INTERVAL '5 hours', 'pro', 'ZA'),
    ('Dave Patel', 'dave@example.com', NOW() - INTERVAL '1 day', 'enterprise', 'IN'),
    ('Eve Johnson', 'eve@example.com', NOW() - INTERVAL '7 days', 'free', 'GB'),
    ('Frank Zulu', 'frank@example.com', NOW() - INTERVAL '30 days', 'pro', 'ZA'),
    ('Grace Kim', 'grace@example.com', NOW() - INTERVAL '60 days', 'free', 'KR'),
    ('Henry Garcia', 'henry@example.com', NOW() - INTERVAL '90 days', 'enterprise', 'MX'),
    ('Irene Okafor', 'irene@example.com', NOW() - INTERVAL '14 days', 'pro', 'NG'),
    ('Jack Williams', 'jack@example.com', NOW() - INTERVAL '45 days', 'free', 'US');

-- Sample products
INSERT INTO products (name, price, category, stock) VALUES
    ('Wireless Mouse', 29.99, 'electronics', 150),
    ('USB-C Hub', 49.99, 'electronics', 80),
    ('Mechanical Keyboard', 89.99, 'electronics', 45),
    ('Standing Desk', 399.99, 'furniture', 12),
    ('Ergonomic Chair', 599.99, 'furniture', 8),
    ('Monitor Arm', 79.99, 'furniture', 30),
    ('Webcam 4K', 129.99, 'electronics', 25),
    ('Noise Canceling Headphones', 249.99, 'electronics', 18),
    ('Desk Lamp', 39.99, 'furniture', 60),
    ('Notebook Set', 14.99, 'office', 200);

-- Sample orders with various statuses and dates
INSERT INTO orders (user_id, product_id, quantity, total, status, ordered_at) VALUES
    (1, 1, 2, 59.98, 'delivered', NOW() - INTERVAL '1 day'),
    (1, 3, 1, 89.99, 'shipped', NOW() - INTERVAL '2 hours'),
    (2, 4, 1, 399.99, 'pending', NOW() - INTERVAL '3 days'),
    (3, 7, 1, 129.99, 'delivered', NOW() - INTERVAL '5 hours'),
    (3, 8, 1, 249.99, 'processing', NOW() - INTERVAL '4 hours'),
    (4, 5, 2, 1199.98, 'delivered', NOW() - INTERVAL '1 day'),
    (5, 2, 1, 49.99, 'cancelled', NOW() - INTERVAL '6 days'),
    (5, 9, 1, 39.99, 'delivered', NOW() - INTERVAL '7 days'),
    (6, 10, 5, 74.95, 'delivered', NOW() - INTERVAL '20 days'),
    (7, 6, 1, 79.99, 'pending', NOW() - INTERVAL '55 days'),
    (8, 4, 1, 399.99, 'delivered', NOW() - INTERVAL '85 days'),
    (9, 8, 1, 249.99, 'shipped', NOW() - INTERVAL '10 days'),
    (10, 1, 3, 89.97, 'delivered', NOW() - INTERVAL '30 days'),
    (10, 9, 2, 79.98, 'pending', NOW() - INTERVAL '40 days'),
    (10, 10, 10, 149.90, 'delivered', NOW() - INTERVAL '42 days');
