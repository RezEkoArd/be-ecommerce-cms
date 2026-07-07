CREATE TYPE discount_type AS ENUM ('percent', 'fixed');
CREATE TYPE order_status  AS ENUM ('draft', 'paid', 'shipped', 'completed', 'cancelled');
 
CREATE TABLE coupons (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           VARCHAR(50)   NOT NULL UNIQUE,
    discount_type  discount_type NOT NULL,
    discount_value NUMERIC(12,2) NOT NULL,
    expires_at     TIMESTAMPTZ,
    is_active      BOOLEAN       NOT NULL DEFAULT true
);
 
CREATE TABLE orders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id)   ON DELETE RESTRICT,
    coupon_id  UUID         REFERENCES coupons(id)          ON DELETE SET NULL,
    status     order_status NOT NULL DEFAULT 'draft',
    subtotal   NUMERIC(12,2) NOT NULL,
    tax        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount   NUMERIC(12,2) NOT NULL DEFAULT 0,
    total      NUMERIC(12,2) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
 
CREATE TABLE order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID NOT NULL REFERENCES orders(id)   ON DELETE CASCADE,
    product_id   UUID REFERENCES products(id)          ON DELETE SET NULL,
    product_name VARCHAR(200)  NOT NULL,   -- snapshot: nama saat dibeli
    price        NUMERIC(12,2) NOT NULL,   -- snapshot: harga saat dibeli
    quantity     INT           NOT NULL CHECK (quantity > 0)
);
 
CREATE INDEX idx_orders_user_id         ON orders(user_id);
CREATE INDEX idx_orders_coupon_id       ON orders(coupon_id);
CREATE INDEX idx_order_items_order_id   ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
