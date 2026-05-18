-- Stress test database: 1M users, 1M products, 3M orders
-- Generated with generate_series() for speed

CREATE TABLE stress_users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    plan VARCHAR(20) DEFAULT 'free',
    country VARCHAR(2) DEFAULT 'US',
    score INT DEFAULT 0,
    last_login TIMESTAMP DEFAULT NOW()
);

CREATE TABLE stress_products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50),
    stock INT DEFAULT 0,
    description TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE stress_orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    ordered_at TIMESTAMP DEFAULT NOW(),
    shipped_at TIMESTAMP,
    notes TEXT DEFAULT ''
);

-- Indexes for realistic workload
CREATE INDEX idx_stress_orders_user ON stress_orders(user_id);
CREATE INDEX idx_stress_orders_product ON stress_orders(product_id);
CREATE INDEX idx_stress_orders_status ON stress_orders(status);
CREATE INDEX idx_stress_orders_ordered_at ON stress_orders(ordered_at);
CREATE INDEX idx_stress_users_created_at ON stress_users(created_at);
CREATE INDEX idx_stress_users_plan ON stress_users(plan);
CREATE INDEX idx_stress_users_country ON stress_users(country);
CREATE INDEX idx_stress_products_category ON stress_products(category);
CREATE INDEX idx_stress_products_price ON stress_products(price);

-- Stress schema tables (for diff testing)
CREATE TABLE stress_config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE stress_audit_log (
    id SERIAL PRIMARY KEY,
    user_id INT,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id INT,
    old_value TEXT,
    new_value TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user ON stress_audit_log(user_id);
CREATE INDEX idx_audit_log_created ON stress_audit_log(created_at);
CREATE INDEX idx_audit_log_action ON stress_audit_log(action);
