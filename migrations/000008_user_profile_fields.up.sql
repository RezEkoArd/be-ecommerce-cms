-- Data profil tambahan untuk storefront. Keduanya opsional.
ALTER TABLE users ADD COLUMN phone VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN birth_date DATE;
