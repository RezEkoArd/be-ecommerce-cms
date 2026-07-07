---
description: Tambah rule atau contoh baru ke rules files berdasarkan kesalahan agent atau pattern baru yang ditemukan. Gunakan saat agent nulis kode yang tidak sesuai pattern project.
allowed-tools: Read, Edit, Glob, Grep
---

# Add Rule / Add Example

Workflow untuk menambah rule baru atau memperkuat contoh di rules yang sudah ada.

## Input yang dibutuhkan

1. **Apa yang agent tulis (salah)** — kode atau deskripsi
2. **Apa yang seharusnya ditulis** — kode atau deskripsi yang benar
3. **Alasan** — kenapa yang benar itu benar

Kalau salah satu belum ada, tanya dulu sebelum lanjut.

## Langkah 1 — Baca rules yang ada

```
@CLAUDE.md
@.claude/rules/backend.md
@.claude/rules/api-design.md
@.claude/rules/logging.md
@.claude/rules/init.md
@.claude/rules/auth.md
@.claude/rules/security.md
```

## Langkah 2 — Tentukan: Prinsip Baru atau Contoh Tambahan?

### A. Apakah prinsip ini SUDAH ADA di rules?
→ **Sudah ada** = **Contoh Tambahan** — tambah ke section ❌/✅ yang relevan, jangan buat section baru

### B. Apakah pattern ini akan muncul di banyak file/layer?
→ **Ya** = **Prinsip Baru** → tambah ringkasan 1 baris di `CLAUDE.md` + detail & contoh di rules file relevan

### C. Apakah ini sementara (masa transisi)?
→ Tambahkan penanda `> ⚠️ TEMP:` dan jelaskan kapan bisa dihapus.

## Langkah 3 — Format rule baru

### Prinsip Baru di rules file

```markdown
## N. [SELALU / JANGAN] [prinsip dalam 1 kalimat]

[1–2 kalimat alasan kenapa rule ini ada.]

### ✅ Prinsip
​```go
// kode yang benar
​```

### ❌ Anti-pattern
​```go
// kode yang salah + komentar kenapa salah
​```
```

### Ringkasan di `CLAUDE.md`
Cukup 1 baris di section yang relevan (misal "Hal yang JANGAN Dilakukan" atau "Coding Conventions") — detail tetap di rules file, jangan duplikasi contoh kode.

## Langkah 4 — File tujuan

| Kasus | File tujuan |
|---|---|
| Layer dependency, interface, error wrap, context, constructor | `.claude/rules/backend.md` |
| Handler, binding, response format, status code, routing, URL | `.claude/rules/api-design.md` |
| Logger, level log, field konteks, data sensitif | `.claude/rules/logging.md` |
| main.go, config, inisialisasi logger/database | `.claude/rules/init.md` |
| Prinsip keamanan: XSS, CSRF, strategi token, cookie flags, CSP | `.claude/rules/security.md` |
| Implementasi auth di Go: JWTAuth middleware, login/refresh/logout, rotation | `.claude/rules/auth.md` |
| Pattern lintas semua area (git, security, testing, migration) | `CLAUDE.md` |

Kalau kasus baru tidak cocok ke file manapun dan diprediksi akan tumbuh (misal: testing, migration), tawarkan buat rules file baru dengan frontmatter `paths:` yang tepat — jangan langsung buat tanpa konfirmasi.

## Langkah 5 — Edit dan report

```
✅ Ditambahkan ke: [nama file]
📍 Section: [nama section]
🔖 Jenis: [Prinsip Baru / Contoh Tambahan]
```
