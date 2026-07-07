---
description: Audit semua rules files — cari duplikasi, rules usang, dan section yang bisa dipadatkan. Jalankan setiap selesai feature besar atau setiap 2 minggu.
allowed-tools: Read, Edit, Glob, Grep
---

# Audit Rules

Workflow untuk menjaga rules files tetap lean, akurat, dan tidak redundant.

## Langkah 1 — Baca semua rules files

```
@CLAUDE.md
@.claude/rules/backend.md
@.claude/rules/api-design.md
@.claude/rules/logging.md
@.claude/rules/init.md
@.claude/rules/auth.md
@.claude/rules/security.md
```

> Kalau ada rules file baru di `.claude/rules/` yang belum tercantum di atas, ikut sertakan — daftar ini harus mencakup **semua** file di folder itu.

## Langkah 2 — Jalankan 5 pengecekan

### Cek A — Duplikasi & Overlap
- Prinsip yang sama di lebih dari satu file? (Perhatikan: `CLAUDE.md` sengaja berisi ringkasan — duplikat berarti contoh kode/detail yang ikut disalin, bukan ringkasan 1 baris)
- Contoh kode mirip/identik di beberapa section?

Tandai: `[DUPLIKAT] <deskripsi> → gabung ke <lokasi tujuan>`

### Cek B — Rules TEMP yang bisa dihapus
- Cari baris yang mengandung `TEMP` atau `sementara`.
- Apakah kondisinya masih berlaku?

Tandai: `[TEMP] <lokasi> → tanyakan ke user: masih relevan?`

### Cek C — Prinsip tanpa contoh / contoh tanpa prinsip
- Rule tanpa contoh kode ✅/❌?
- Section dengan contoh panjang tapi prinsipnya tidak eksplisit?

Tandai: `[LEMAH] <lokasi> → tambahkan contoh kode ✅/❌`

### Cek D — Section terlalu panjang
- Lebih dari 3 contoh untuk 1 prinsip yang sama?
- Bisa diringkas ke tabel anti-pattern?

Tandai: `[GEMUK] <lokasi> → kandidat untuk dipadatkan`

### Cek E — Rules vs kondisi kode aktual
- Apakah rules masih sesuai dengan struktur project? (cek path di frontmatter `paths:` masih valid — misal `internal/database/` benar-benar ada)
- Apakah nama package/fungsi di contoh masih sama dengan yang dipakai di kode?

Tandai: `[USANG] <lokasi> → update sesuai kondisi kode`

## Langkah 3 — Format laporan

```
# Audit Rules — [tanggal]

## Ringkasan
Total rules files: N
Temuan: N duplikasi · N TEMP · N lemah · N gemuk · N usang

---

## 🔴 Perlu Tindakan Segera
### [DUPLIKAT] ...
### [TEMP] ...
### [USANG] ...

## 🟡 Perlu Diperhatikan
### [LEMAH] ...
### [GEMUK] ...

## 🟢 Sehat
- <file yang tidak ada temuan>

## Tindakan yang Disarankan
1. [aksi konkret]
2. [aksi konkret]
```

## Langkah 4 — Tanya konfirmasi sebelum edit

**Jangan langsung edit** — tanya dulu:
```
Dari temuan di atas, mau langsung perbaiki yang mana?
Ketik nomor aksi atau "semua".
```

## Kapan Jalankan
- Setelah selesai develop 1 feature besar (misal: 1 modul CRUD lengkap)
- Setiap 2 minggu kalau development aktif
- Kalau CLAUDE.md sudah > 150 baris
