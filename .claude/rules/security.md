---
paths:
  - "internal/handler/**/*.go"
  - "internal/midleware/**/*.go"
  - "internal/service/**/*.go"
---

# Keamanan — ichiba

> Dokumen ini adalah aturan & rujukan keamanan untuk auth di ichiba.
> Backend: Go/Gin. Frontend: Next.js. Setiap keputusan di sini disertai alasannya.

## Prinsip inti

1. **XSS adalah masalah output, bukan input.** Pertahanan utama = escaping saat render, bukan validasi saat input.
2. **Apa pun yang bisa dibaca JavaScript, bisa dicuri XSS.** Jangan simpan token sensitif di tempat yang terbaca JS (`localStorage`, cookie non-HttpOnly).
3. **Pertahanan berlapis.** Cookie flags + CSP + output encoding + rotasi token. Jangan bergantung pada satu lapisan.

## Strategi penyimpanan token (KEPUTUSAN)

- **Access token** → JWT berumur pendek (15 menit), disimpan **hanya di memory** (variabel JS). Tidak di localStorage, tidak di cookie.
- **Refresh token** → string acak berumur panjang (7 hari), disimpan di **cookie `HttpOnly; Secure; SameSite=Strict`**.

**Alasan:** access token di memory + refresh di cookie HttpOnly = aman dari XSS (tidak ada yang persisten & terbaca JS) sekaligus aman dari CSRF (SameSite mencegah cookie ikut lintas-situs).

### Access token vs refresh token

| | Access token | Refresh token |
|---|---|---|
| Jenis | JWT (stateless) | String acak (dicatat di DB) |
| Dicek | Tiap request | Hanya saat `/refresh` |
| Umur | 15 menit | 7 hari |
| Disimpan (klien) | Memory | Cookie HttpOnly |
| Bisa dicabut | Tidak perlu (umur pendek) | Ya — wajib (tabel `refresh_tokens`) |

## Konfigurasi cookie (refresh token)

Set dengan flag: `HttpOnly; Secure; SameSite=Strict; Path=/api/auth/refresh`.
Contoh kode Go untuk `SetCookie` ada di [`auth.md`](auth.md) §1.

| Flag | Mencegah | TIDAK mencegah |
|---|---|---|
| `HttpOnly` | Pencurian token oleh XSS (JS tak bisa baca) | CSRF |
| `Secure` | Penyadapan di jaringan (hanya HTTPS) | XSS / CSRF |
| `SameSite` | CSRF (cookie tak ikut lintas-situs) | XSS |

## Alur auth (ringkas)

- **Login** → verifikasi kredensial → keluarkan access token (body) + refresh token (cookie HttpOnly) → simpan hash refresh di DB.
- **Request** → kirim `Authorization: Bearer <access>` dari memory.
- **401 (access kedaluwarsa)** → interceptor panggil `POST /refresh` → validasi cookie → access token baru + rotasi refresh → ulangi request asli.
- **App load (page reload)** → panggil `/refresh` sekali untuk memulihkan access token di memory.
- **Refresh gagal** → redirect ke `/login`.
- **Logout** → cabut refresh token di DB (isi `revoked_at`) + hapus cookie.

## Aturan backend (Go / Gin)

> Contoh kode konkret (middleware `JWTAuth`, handler login/refresh/logout, service rotation, migration `refresh_tokens`) ada di [`auth.md`](auth.md).

- ✅ Simpan **hash** refresh token di DB (`token_hash`), bukan token mentah.
- ✅ **Rotasi** refresh token tiap kali `/refresh` dipakai: cabut yang lama, terbitkan yang baru.
- ✅ Cek `revoked_at` dan `expires_at` sebelum menerima refresh token.
- ✅ Access token: JWT, verifikasi tanda tangan di middleware auth (tanpa query DB).
- ✅ Set header CSP lewat middleware (batasi sumber skrip; larang inline script bila memungkinkan).
- ✅ Set `SameSite` sebelum `SetCookie` (`c.SetSameSite(http.SameSiteStrictMode)`).
- ❌ Jangan pernah kirim refresh token di body respons — hanya lewat cookie.
- ❌ Jangan buat refresh token sebagai JWT stateless (menghapus kemampuan revocation).
- ❌ Jangan telan error validasi token diam-diam; balas 401 yang jelas.

## Aturan frontend (Next.js)

- ✅ Access token **hanya di memory** (variabel modul / state), tidak persisten.
- ✅ Sertakan `credentials: "include"` di **setiap** request agar cookie ikut.
- ✅ Logika refresh terpusat di **satu interceptor**, bukan disebar per pemanggilan.
- ✅ Batasi retry (flag `retry=false` saat mengulang) untuk mencegah loop tak terhingga.
- ✅ Panggil `/refresh` saat app load untuk memulihkan sesi setelah page reload.
- ❌ Jangan simpan token apa pun di `localStorage` / `sessionStorage`.
- ❌ Jangan render HTML mentah dari data user (`dangerouslySetInnerHTML`) tanpa sanitasi.

## Pencegahan XSS

- **Output encoding (utama):** escape data sesuai konteks (HTML, atribut, URL, JS). Andalkan auto-escape framework (React/Next `{}`).
- **CSP:** header `Content-Security-Policy` sebagai lapisan kedua — bahkan bila payload lolos, CSP bisa mencegah eksekusi.
- **Sanitasi:** kalau harus render HTML kaya (mis. deskripsi produk berformat), pakai library sanitizer allowlist. Jangan bikin sendiri.
- **Validasi input:** lapisan bantu untuk integritas data, BUKAN pertahanan utama XSS.

## Pencegahan CSRF

- `SameSite=Strict` (atau `Lax`) pada cookie refresh = pertahanan utama.
- Endpoint `/refresh` bisa ditambah CSRF token untuk jaminan ekstra bila perlu.
- Ingat: `HttpOnly` TIDAK mencegah CSRF — itu tugas `SameSite`.

## Checklist sebelum rilis

- [ ] Access token JWT umur pendek, hanya di memory
- [ ] Refresh token string acak, cookie `HttpOnly; Secure; SameSite`
- [ ] Hash refresh token tersimpan di DB, bukan token mentah
- [ ] Rotasi + revocation refresh token berfungsi (logout menghapus sesi)
- [ ] `credentials: "include"` di semua request frontend
- [ ] Interceptor refresh punya guard anti-loop
- [ ] Header CSP terpasang via middleware
- [ ] Semua HTTPS di produksi (flag `Secure` aktif)
