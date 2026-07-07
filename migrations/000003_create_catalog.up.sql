CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(120) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
 
CREATE TABLE products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    name        VARCHAR(200)  NOT NULL,
    slug        VARCHAR(220)  NOT NULL UNIQUE,
    description TEXT,
    price       NUMERIC(12,2) NOT NULL,
    stock       INT           NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
 
CREATE TABLE product_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        TEXT    NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
 
CREATE INDEX idx_products_category_id      ON products(category_id);
CREATE INDEX idx_products_slug             ON products(slug);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);
