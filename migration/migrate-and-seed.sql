-- Clean up old tables
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS inventory;

-- Enable extension for generate UUID
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Order status ENUM
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
        CREATE TYPE order_status AS ENUM ('PENDING','PAID','CANCELLED');
    END IF;
END$$;

-- reusable function for update updated_at column
CREATE OR REPLACE FUNCTION  updated_updated_at_column()
RETURNs TRIGGER AS $$
BEGIN 
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';


CREATE TABLE products (
    id BIGSERIAL  PRIMARY KEY,
    public_id UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE, -- external ID
    name VARCHAR(100) NOT NULL,
    base_price DECIMAL(10,2) NOT NULL, -- the current actual price
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inventory (
    id BIGSERIAL  PRIMARY KEY,
    public_id UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE, -- external ID
    product_id BIGINT REFERENCES products(id) ON DELETE CASCADE UNIQUE NOT NULL,
    quantity INT NOT NULL CHECK (quantity >= 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE orders (
    id BIGSERIAL  PRIMARY KEY,
    public_id UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE, -- external ID
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    user_id BIGINT, -- Placeholder for auth layer later,
    -- AUDITABLE SNAPSHOT --
    quantity_bought INT DEFAULT 1 NOT NULL CHECK(quantity_bought > 0),
    price_per_unit DECIMAL(10,2) NOT NULL, -- AUDITABLE SNAPSHOT
    total_amount DECIMAL(10,2) NOT NULL CHECK(total_amount = ROUND(quantity_bought * price_per_unit, 2)), -- quanityu_bought * price_per_unit
    ------------------------
    status order_status DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_product_id ON orders(product_id);

-- attach the function to products table
CREATE TRIGGER update_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE FUNCTION updated_updated_at_column();

-- attach the function to orders table
CREATE TRIGGER update_orders_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE FUNCTION updated_updated_at_column();

-- attach the function to inventory table
CREATE TRIGGER update_inventory_updated_at
BEFORE UPDATE ON inventory
FOR EACH ROW
WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE FUNCTION updated_updated_at_column();