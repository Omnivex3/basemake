-- Insert 500K users
INSERT INTO stress_users (name, email, created_at, plan, country, score, last_login)
SELECT
    'User_' || n,
    'user_' || n || '@example.com',
    NOW() - (random() * INTERVAL '365 days'),
    CASE WHEN random() < 0.6 THEN 'free' WHEN random() < 0.85 THEN 'pro' ELSE 'enterprise' END,
    (ARRAY['US','ZA','GB','IN','DE','FR','NG','KE','BR','JP'])[1 + (random() * 9)::int],
    (random() * 10000)::int,
    NOW() - (random() * INTERVAL '30 days')
FROM generate_series(1, 500000) n;

-- Insert 500K products
INSERT INTO stress_products (name, price, category, stock, description)
SELECT
    'Product_' || n,
    (random() * 500 + 0.99)::numeric(10,2),
    (ARRAY['electronics','furniture','office','clothing','food','tools','sports','books','music','toys'])[1 + (random() * 9)::int],
    (random() * 1000)::int,
    'Description for product ' || n || '. This is a sample product description for stress testing purposes.'
FROM generate_series(1, 500000) n;

-- Insert 2M orders
INSERT INTO stress_orders (user_id, product_id, quantity, total, status, ordered_at, shipped_at)
SELECT
    1 + (random() * 499999)::int,
    1 + (random() * 499999)::int,
    1 + (random() * 10)::int,
    ((random() * 1000) + 10)::numeric(10,2),
    (ARRAY['pending','processing','shipped','delivered','cancelled','returned'])[1 + (random() * 5)::int],
    NOW() - (random() * INTERVAL '180 days'),
    CASE WHEN random() < 0.7 THEN NOW() - (random() * INTERVAL '170 days') ELSE NULL END
FROM generate_series(1, 2000000) n;

-- Analyze for query planner
ANALYZE stress_users;
ANALYZE stress_products;
ANALYZE stress_orders;
ANALYZE stress_config;
ANALYZE stress_audit_log;

-- Small tables for config/audit
INSERT INTO stress_config (key, value)
SELECT 'config_' || n, 'value_' || n
FROM generate_series(1, 1000) n;

INSERT INTO stress_audit_log (user_id, action, entity_type, entity_id, old_value, new_value)
SELECT
    1 + (random() * 499999)::int,
    (ARRAY['CREATE','UPDATE','DELETE','LOGIN','LOGOUT','EXPORT','IMPORT'])[1 + (random() * 6)::int],
    (ARRAY['user','order','product','config'])[1 + (random() * 3)::int],
    (random() * 100000)::int,
    'old_value_' || (random() * 100)::int,
    'new_value_' || (random() * 100)::int
FROM generate_series(1, 100000) n;
