-- Alamat disalin ke pesanan sebagai snapshot, bukan foreign key.
-- Kalau user mengedit/menghapus alamatnya nanti, pesanan lama tetap
-- menunjukkan ke mana barang dikirim saat itu.
ALTER TABLE orders ADD COLUMN shipping_recipient   VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN shipping_phone       VARCHAR(20)  NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN shipping_street      TEXT         NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN shipping_city        VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN shipping_postal_code VARCHAR(10)  NOT NULL DEFAULT '';
