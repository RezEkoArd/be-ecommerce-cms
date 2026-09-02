-- Ganti ikon teks dengan gambar sampul kategori (URL objek di MinIO).
ALTER TABLE categories DROP COLUMN icon;
ALTER TABLE categories ADD COLUMN image_url TEXT NOT NULL DEFAULT '';
