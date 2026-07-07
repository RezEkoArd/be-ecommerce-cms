# claude-golang-template

Boilerplate konfigurasi Claude Code untuk project backend Go dengan stack:
Gin + GORM v2 + PostgreSQL + godotenv + zerolog (via `pkg/logger`) + Golang-Migrate + Testify.

Diambil dari konfigurasi yang sudah terbukti dipakai di project
`be-cms-company-profile-web-tour`.

## Isi

```
├── CLAUDE.md                    # Template konteks project — ISI PLACEHOLDER-nya
└── .claude/
    ├── rules/                   # Konvensi per topik, dimuat otomatis sesuai path file
    │   ├── backend.md            # Layer dependency, interface, error wrap, context
    │   ├── api-design.md         # ShouldBindJSON, response format, status code, routing
    │   ├── logging.md            # pkg/logger, level log, field konteks, data sensitif
    │   └── init.md               # Urutan inisialisasi main.go, config, logger, database
    └── commands/                # Dipanggil manual dengan /nama
        ├── add-rule.md           # /add-rule — tambah rule saat agent salah pattern
        ├── check-convention.md   # /check-convention — checklist sebelum coding
        └── audit-rules.md        # /audit-rules — bersihkan rules usang/duplikat
```

## Cara pakai di project baru

```bash
# dari root project baru:
cp -r ~/development/claude-golang-template/.claude .
cp ~/development/claude-golang-template/CLAUDE.md .
```

Lalu:

1. **Isi placeholder di `CLAUDE.md`** — `<NAMA_PROJECT>`, `<NAMA_BINARY>`,
   `<DESKRIPSI_SINGKAT_PROJECT>`. Bisa juga jalankan `/init` di Claude Code lalu
   minta Claude merge hasilnya ke template ini.
2. **Cek struktur project** — rules mengasumsikan layout `cmd/` + `internal/` +
   `pkg/logger` + `pkg/response`. Kalau layout berbeda, sesuaikan `paths:` di
   frontmatter tiap file rules.
3. Jalankan `/check-convention` sebagai tes bahwa rules terbaca.

## Update rules ke depannya

Rules berevolusi lewat siklus di masing-masing project:

- `/add-rule` — saat agent menulis kode yang salah pattern, tambahkan sebagai
  contoh atau prinsip baru.
- `/audit-rules` — jalankan setiap selesai feature besar atau 2 minggu untuk
  memangkas duplikasi, rule TEMP kadaluarsa, dan section gemuk.

Kalau hasil perbaikan di sebuah project layak dipakai project lain, salin balik
perubahannya ke template ini supaya project baru berikutnya kebagian.
