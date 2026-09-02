-- Ikon kategori: karakter Jepang atau emoji (mis. 上, 羽織, 👕).
-- VARCHAR(16) cukup untuk beberapa karakter multibyte.
ALTER TABLE categories ADD COLUMN icon VARCHAR(16) NOT NULL DEFAULT '';
