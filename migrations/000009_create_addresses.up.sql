CREATE TABLE addresses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label        VARCHAR(50)  NOT NULL,
    recipient    VARCHAR(100) NOT NULL,
    phone        VARCHAR(20)  NOT NULL,
    street       TEXT         NOT NULL,
    city         VARCHAR(100) NOT NULL,
    postal_code  VARCHAR(10)  NOT NULL,
    is_primary   BOOLEAN      NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);

-- Hanya boleh ada satu alamat utama per user. Dijaga di level DB
-- supaya tidak bergantung pada urutan operasi di aplikasi.
CREATE UNIQUE INDEX idx_addresses_one_primary
    ON addresses(user_id) WHERE is_primary;
