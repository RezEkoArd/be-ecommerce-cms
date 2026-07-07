---
description: Cek konvensi yang berlaku sebelum mulai coding — baca rules yang relevan dan tampilkan checklist ringkas sebagai reminder.
allowed-tools: Read, Glob, Grep
---

# Check Convention

Workflow proaktif — dijalankan **sebelum** mulai coding untuk memastikan agent tahu pattern yang berlaku.

## Input yang dibutuhkan

```
Sebutkan:
1. Apa yang akan dibuat/diedit? (endpoint, service, repository, middleware, migration, dll)
2. Nama file atau path-nya kalau sudah tahu
3. Modul mana? (posts, category, user, dll)
```

## Langkah 1 — Baca rules

```
@CLAUDE.md
@.claude/rules/backend.md
@.claude/rules/api-design.md
@.claude/rules/logging.md
@.claude/rules/init.md
```

## Langkah 2 — Identifikasi rules relevan

| Jenis task | Rules yang relevan |
|---|---|
| Endpoint CRUD baru (handler + service + repo) | semua kecuali `init.md` |
| Handler / route / request-response | `api-design.md` + `logging.md` |
| Service + unit test | `backend.md` + section Testing di `CLAUDE.md` |
| Repository / query GORM | `backend.md` + `logging.md` + section Database di `CLAUDE.md` |
| Middleware (auth, rate limit) | `api-design.md` + `logging.md` |
| Migration baru | section Database & Migration di `CLAUDE.md` |
| Perubahan di main.go / config | `init.md` |
| Model / struct entity | `backend.md` + section Database di `CLAUDE.md` |

## Langkah 3 — Format output

```
# Convention Check — [nama task / nama file]

## [Kategori 1, misal: Layer & Dependency]
✅ [convention yang harus diikuti]
❌ Jangan: [anti-pattern yang relevan]

## [Kategori 2, misal: Logging]
✅ [convention yang harus diikuti]

---
Siap mulai. Semua convention di atas akan diterapkan.
Ada convention yang perlu disesuaikan?
```

**Aturan output:**
- Kalimat singkat dan actionable — bukan copy-paste isi rules
- Maksimal 5–7 poin per kategori
- Sertakan nama package/fungsi spesifik (`ShouldBindJSON`, `response.NewResponse`, `logger.Errorf`, `WithContext(ctx)`, dll)
- Sebutkan nama file yang benar kalau bisa ditentukan

## Contoh output

### Input: "bikin endpoint CRUD untuk testimonial"

```
# Convention Check — CRUD Testimonial

## Layer & Dependency
✅ handler → service → repository, satu arah, semua via interface
✅ Constructor: NewTestimonialHandler(svc), NewTestimonialService(repo), NewTestimonialRepository(db)
✅ ctx diteruskan dari handler sampai repository
❌ Jangan: import gin/gorm di service, logic bisnis di handler

## Handler & API
✅ ShouldBindJSON + validasi binding tag, bukan BindJSON
✅ Response: response.NewResponse(code, pesan, data) — konsisten
✅ Route group: /api/v1/testimonials (plural, kebab-case)
❌ Jangan: return gorm.ErrRecordNotFound atau error mentah ke client

## Repository & GORM
✅ db.WithContext(ctx) sebelum semua query
✅ Soft delete: field DeletedAt gorm.DeletedAt
✅ Error: log via logger.Errorf lalu wrap fmt.Errorf("testimonialRepository.FindByID: %w", err)

## Logging
✅ Log hanya di handler dan repository, tidak di service
✅ Sertakan field konteks (testimonial_id, dll), jangan log data sensitif

## Testing & Migration
✅ Unit test service dengan mock repository manual (testify/mock)
✅ Migration SQL via make migrate-create, bukan AutoMigrate

---
Siap mulai. Semua convention di atas akan diterapkan.
Ada convention yang perlu disesuaikan?
```
