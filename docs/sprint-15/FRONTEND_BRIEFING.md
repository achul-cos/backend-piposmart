# Briefing Frontend — Modul Importing Sprint 15

Tanggal: 30 Juli 2026
Untuk: Tim Frontend `crm_piposmart`
Base path: `/api/v1`

Dokumen ini ringkasan praktis untuk mulai integrasi modul importing. Untuk contoh request/response
lengkap tiap profil dan tiap error case, lihat [`api-testing.md`](./api-testing.md). Untuk laporan
teknis sprint, lihat [`sprint-15.md`](./sprint-15.md).

## 1. Apa yang baru bisa dilakukan

Modul importing mengganti proses input manual Excel jadi: **upload file → validasi otomatis →
preview baris → commit → data masuk ke modul terkait** (subscription, wallet, target, partner,
lead/interaction). Ada 7 profil (2 dari Sprint 14, 5 dari Sprint 15):

| Profile | File asli (contoh) | Sheet | Butuh `sheet_name`? | Butuh `target_sales_user_id`? | Data masuk ke |
| --- | --- | --- | --- | --- | --- |
| `OWNER_OUTLET` | `01. Owner & Outlet` | pertama | tidak | tidak | Owner + Outlet |
| `NON_REGISTER` | `04. Data Belum Registrasi` | pertama | tidak | tidak | Lead non-register |
| `NEW_SUBSCRIBE` | `02. New & Subscribe` | pertama | tidak | tidak | Wallet top-up + Subscription order |
| `MONTHLY_ACTIVE` | `05. Monthly Active` | pertama | tidak | tidak | `outlet_monthly_activity_snapshot` |
| `BONUS_MITRA` | `06. Data Bonus Mitra`, sheet **"Mitra - Referral"** | ke-2 | tidak (auto-detect) | tidak | `partner_bonus_referral_snapshots` |
| `SALES_CALL_CHAT` | file PBGC sales (24 sheet) | contoh `Call & Chat-Lidya` | **ya, wajib** | **ya, wajib** | Interaction / Closing |
| `SALES_TARGET` | file PBGC sales, sheet `TARGET` | `TARGET` | **ya, wajib** | **ya, wajib** | Target Sales bulanan |

**Kenapa `SALES_CALL_CHAT`/`SALES_TARGET` beda sendiri**: file sumbernya (PBGC) punya 24 sheet —
beberapa sales rep berbeda, plus salinan lama/duplikat dengan header nyaris identik. Backend
**sengaja tidak** auto-detect sheet untuk 2 profil ini — kalau salah pilih sheet otomatis, bisa
salah masuk data sales lain tanpa error apapun. Frontend **wajib** kasih dropdown pilih sheet + PIC
sales tujuan secara eksplisit sebelum upload.

## 2. Alur end-to-end (state machine)

```
UPLOADED --(worker)--> VALIDATING --> VALIDATED / VALIDATION_FAILED
                                           |
                                    (user klik commit)
                                           v
                                     COMMITTING --> COMMITTED / COMMIT_FAILED
```

- Upload (`POST /imports`) langsung balik response berisi `batch_id` dan status awal, **tapi
  validasi jalan async di background** (worker job) — jangan asumsikan file sudah divalidasi saat
  response upload diterima.
- **Poll `GET /imports/{id}`** sampai `status` berubah jadi `VALIDATED` (baru boleh tampilkan
  tombol Commit) atau `VALIDATION_FAILED` (tampilkan `error_message`).
- Commit (`POST /imports/{id}/commit`) juga async — poll lagi sampai `COMMITTED`/`COMMIT_FAILED`.
- Re-upload file yang identik (hash sama + profile + sheet_name sama) **tidak membuat batch baru**
  — backend mengembalikan batch yang sudah ada. Aman dipakai untuk retry/refresh halaman.

## 3. Status baris (row) — termasuk yang BARU

`GET /imports/{id}/rows` menampilkan tiap baris hasil parsing dengan salah satu status:

| Status | Arti | Aksi frontend |
| --- | --- | --- |
| `VALID` | Siap commit | tampilkan di preview, tidak perlu aksi |
| `INVALID` | Data di Excel-nya rusak/salah format | tampilkan `validation_errors`, user perbaiki file lalu re-upload |
| `UNMATCHED` **(baru)** | Baris valid secara format, tapi kode owner/outlet/lead yang dirujuk **tidak ditemukan** di database | tampilkan tombol **"Hubungkan manual"** → panggil endpoint relink (lihat §4) |
| `COMMITTED` | Sudah masuk ke tabel tujuan | tampilkan sebagai selesai |
| `COMMIT_FAILED` | Gagal saat commit (jarang, biasanya bug/race condition) | tampilkan `commit_error`, sarankan retry commit |

**Penting soal `UNMATCHED`**: ini BEDA dari Sprint 14 (`OWNER_OUTLET`/`NON_REGISTER`), di mana owner
yang belum ada otomatis dibuat baru. Untuk 5 profil Sprint 15, entitasnya **diasumsikan sudah ada**
dari data sebelumnya (owner/outlet/lead/partner) — kalau kode yang dirujuk baris Excel tidak
ditemukan, itu dianggap kandidat rekonsiliasi yang butuh keputusan admin, bukan auto-create.

## 4. Endpoint BARU: Relink baris UNMATCHED

```
POST /imports/{id}/rows/{row_id}/relink
```

Body (minimal satu field wajib diisi):

```json
{
  "owner_id": 123,
  "outlet_id": 45,
  "lead_id": 67
}
```

- Field yang di-`null`/tidak dikirim akan **mempertahankan** nilai lama pada baris tsb (tidak
  ditimpa kosong).
- Sukses → baris kembali ke status `VALID`, `commit_error` dikosongkan. **Baris belum otomatis
  ter-commit** — user tetap perlu klik commit batch (atau backend akan commit ulang batch yang
  masih `VALIDATED`/pending row).
- Response `200`: object row yang sudah ter-update (sama shape dengan item `GET /imports/{id}/rows`).

Error yang mungkin muncul:

| HTTP | Code | Kapan | Solusi UI |
| --- | --- | --- | --- |
| `409` | `ROW_NOT_UNMATCHED` | Baris yang di-relink bukan status `UNMATCHED` (mis. sudah `COMMITTED`) | Sembunyikan tombol relink kalau status baris bukan `UNMATCHED` |
| `400` | `RELINK_ENTITY_REQUIRED` | Body kosong, tidak ada satupun ID dikirim | Validasi form: minimal 1 field harus diisi sebelum submit |
| `404` | `NOT_FOUND` | `row_id` tidak ada di batch tsb | — |

**Rekomendasi UI**: di halaman detail batch, filter baris `status=UNMATCHED`
(`GET /imports/{id}/rows?status=UNMATCHED`) jadi tab/section terpisah — semacam "perlu tindakan admin"
— dengan form pencarian owner/outlet/lead by code/name untuk mengisi ID yang benar sebelum submit
relink.

## 5. Endpoint BARU: Ringkasan status semua batch

```
GET /imports/summary
```

Response:

```json
{
  "data": {
    "total": 9,
    "counts_by_status": {
      "COMMITTED": 7,
      "VALIDATED": 1,
      "VALIDATION_FAILED": 1
    },
    "needs_attention": 1
  }
}
```

- `needs_attention` = jumlah batch berstatus `VALIDATION_FAILED` + `COMMIT_FAILED`.
- Cocok dipakai untuk badge/notifikasi di menu "Import" (mis. "1 batch perlu perhatian") tanpa
  perlu page melalui `GET /imports?status=...` satu-satu per status.
- `counts_by_status` hanya berisi status yang benar-benar punya batch (status dengan count 0 tidak
  muncul di object) — treat key yang tidak ada sebagai 0 di frontend.

## 6. Upload — parameter per profil (form-data, bukan JSON)

```
POST /imports
Content-Type: multipart/form-data

file: <file .xlsx>
profile: NEW_SUBSCRIBE | MONTHLY_ACTIVE | BONUS_MITRA | SALES_CALL_CHAT | SALES_TARGET | ...
sheet_name: (wajib untuk SALES_CALL_CHAT/SALES_TARGET, kosongkan untuk profil lain)
target_sales_user_id: (wajib untuk SALES_CALL_CHAT/SALES_TARGET, kosongkan untuk profil lain)
```

- `profile` boleh dikosongkan untuk profil yang bisa auto-detect dari header (semua kecuali
  `SALES_CALL_CHAT`/`SALES_TARGET`) — tapi **lebih aman selalu kirim eksplisit** dari dropdown UI,
  supaya user tidak salah pilih file untuk profil yang salah.
- Kalau `sheet_name` dikirim tanpa `profile`, backend menolak dengan `SHEET_NAME_NEEDS_PROFILE` —
  UI harus mewajibkan pilih profile dulu sebelum field sheet_name muncul.

Contoh spesifik per profil ada di [`api-testing.md` §6](./api-testing.md#6-detail-testing-api)
(request form-data lengkap + contoh `raw_payload` hasil parsing tiap profil).

## 7. Endpoint lain yang relevan (semua sudah ada sejak Sprint 14/15 awal)

| Endpoint | Kegunaan |
| --- | --- |
| `GET /imports` / `/imports/all` | List batch (histori upload) |
| `GET /imports/{id}` | Detail 1 batch — progress, status, jumlah row per kategori |
| `GET /imports/{id}/rows` / `/rows/all` | Daftar baris hasil parsing, bisa filter `?status=` |
| `GET /imports/{id}/rejected-rows/export` | Download CSV baris `INVALID` beserta error-nya |
| `GET /imports/{id}/file` / `/file/download` | Lihat/unduh file asli yang di-upload |
| `POST /imports/{id}/commit` | Trigger commit (hanya efektif saat `status=VALIDATED`) |

## 8. Checklist integrasi

- [ ] Dropdown profile eksplisit (jangan andalkan auto-detect di UI, walau backend mendukungnya).
- [ ] Field `sheet_name` + `target_sales_user_id` **muncul kondisional** hanya untuk
      `SALES_CALL_CHAT`/`SALES_TARGET`, dan wajib diisi sebelum submit.
- [ ] Setelah upload, simpan `batch_id`, mulai polling `GET /imports/{id}` (mis. tiap 2-3 detik)
      sampai status stabil (`VALIDATED`/`VALIDATION_FAILED`/`COMMITTED`/`COMMIT_FAILED`).
- [ ] Tombol Commit **disabled** kecuali `status=VALIDATED`.
- [ ] Tab/filter khusus baris `UNMATCHED` di halaman detail batch + form relink manual (§4).
- [ ] Badge "perlu perhatian" di menu Import pakai `GET /imports/summary` → `needs_attention`.
- [ ] Tampilkan `request_id` dari response error di UI (memudahkan tracing ke tim backend).
- [ ] Beri notifikasi non-error kalau upload ulang mengembalikan batch lama ("File ini sudah pernah
      diupload sebelumnya, menampilkan hasil yang ada").
- [ ] Validasi ekstensi `.xlsx` di client sebelum upload (backend tetap validasi ulang, tapi UX
      lebih cepat kasih feedback).

## 9. Pertanyaan terbuka / butuh koordinasi

- Wording/label sheet & target user di dropdown `SALES_CALL_CHAT`/`SALES_TARGET` — perlu daftar
  sales user aktif untuk populate dropdown `target_sales_user_id` (endpoint identity/user sudah ada
  di modul lain, bukan bagian importing).
- Desain UI untuk baris `UNMATCHED` (pencarian owner/outlet/lead saat relink) belum ada mockup —
  koordinasikan dengan desain sebelum implementasi form relink.
