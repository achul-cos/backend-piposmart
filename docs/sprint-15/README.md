# API Documentation Update - Sprint 15 Import Profiles, Validation, & Runtime Audit

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 15 |
| Tanggal Update | 30 Juli 2026 |
| Environment | Local Development |
| Fokus | Import profile baru, validasi runtime, error handler, dan audit hasil commit |

## 2. Ringkasan Sprint 15

Sprint 15 pada backend berfokus pada penyelesaian dan validasi modul importing untuk file operasional marketing yang sebelumnya masih dikelola manual dari Excel.

Ruang lingkup yang diverifikasi pada sprint ini:

- import transaksi `NEW_SUBSCRIBE`;
- import histori aktivitas `MONTHLY_ACTIVE`;
- import data bonus/referal mitra `BONUS_MITRA`;
- import target sales `SALES_TARGET`;
- import call & chat sales `SALES_CALL_CHAT`;
- histori batch import;
- histori row import;
- akses file asli upload;
- commit batch import;
- error handler dan pencegahan error untuk frontend.

## 3. Hasil Validasi Utama

Pada tanggal **30 Juli 2026**, pengujian manual dan verifikasi runtime menunjukkan bahwa alur utama import sekarang berjalan untuk profil berikut:

- `NEW_SUBSCRIBE`
- `MONTHLY_ACTIVE`
- `BONUS_MITRA`
- `SALES_TARGET`
- `SALES_CALL_CHAT`

Semua profil di atas berhasil melewati alur:

1. upload file,
2. validasi worker,
3. inspeksi row hasil parsing,
4. commit,
5. verifikasi side-effect pada tabel tujuan.

## 4. Temuan Audit Saat Testing

Selama validasi sprint ini ditemukan beberapa temuan runtime yang penting, lalu diselesaikan sebelum dokumentasi ini ditulis:

### 4.1 Deadlock progress update saat validasi

Gejala:

- batch berhenti pada status `UPLOADED` atau lama sekali tidak bergerak;
- job `IMPORT_VALIDATE` tampak hidup, tetapi progress tidak maju.

Penyebab:

- worker mengunci row `import_batches` di dalam transaction;
- progress update sebelumnya memakai koneksi DB terpisah, sehingga mencoba meng-update row yang sedang dikunci oleh transaction yang sama.

Status:

- **Sudah diperbaiki** dengan membuat progress update memakai executor/transaction yang sama saat validasi berjalan.

### 4.2 Error mapping beberapa validasi impor belum lengkap

Gejala:

- request yang seharusnya `400` bisa jatuh ke `500 INTERNAL_ERROR`.

Kasus yang terdampak:

- `sheet_name` wajib;
- `sheet_name` tanpa `profile`;
- `sheet_name` tidak ditemukan;
- `target_sales_user_id` wajib.

Status:

- **Sudah diperbaiki** dengan error code eksplisit:
  - `SHEET_NAME_REQUIRED`
  - `SHEET_NAME_NEEDS_PROFILE`
  - `SHEET_NOT_FOUND`
  - `TARGET_SALES_USER_REQUIRED`

### 4.3 Collision pada `import_batches.code`

Gejala:

- upload file yang sama pada tanggal yang sama, tetapi dengan konteks berbeda, bisa memicu:

```text
Duplicate entry 'IMPORT-...' for key 'import_batches.uq_import_batches_code'
```

Penyebab:

- `code` batch sebelumnya hanya memakai tanggal + potongan hash file;
- konteks seperti `profile`, `sheet_name`, dan `target_sales_user_id` belum ikut membedakan code.

Status:

- **Sudah diperbaiki** dengan menambahkan hash konteks pada generator code batch.

### 4.4 Migration terbaru belum dijalankan

Gejala:

- `BONUS_MITRA` sempat gagal commit dengan error tabel snapshot belum ada.

Penyebab:

- migration baru sudah dibuat di source code, tetapi belum diterapkan ke database lokal.

Status:

- **Sudah diselesaikan** dengan menjalankan:

```powershell
go run . migrate up
```

Migration yang diterapkan:

- `20260730000200_bonus_mitra_snapshots.sql`

## 5. Endpoint Sprint 15 yang Diverifikasi

### 5.1 Histori Import

- `POST /api/v1/imports`
- `GET /api/v1/imports`
- `GET /api/v1/imports/all`
- `GET /api/v1/imports/{id}`
- `GET /api/v1/imports/{id}/rows`
- `GET /api/v1/imports/{id}/rows/all`
- `POST /api/v1/imports/{id}/commit`

### 5.2 File Asli Import

- `GET /api/v1/imports/{id}/file`
- `GET /api/v1/imports/{id}/file/download`

## 6. Profil Import yang Diverifikasi

| Profile | Tujuan | Hasil |
| --- | --- | --- |
| `NEW_SUBSCRIBE` | Import transaksi subscribe/history aktivasi | PASS |
| `MONTHLY_ACTIVE` | Import histori status aktif bulanan outlet | PASS |
| `BONUS_MITRA` | Import histori referral dan bonus mitra | PASS |
| `SALES_TARGET` | Import target sales dari sheet tertentu | PASS |
| `SALES_CALL_CHAT` | Import histori call/chat sales dari sheet tertentu | PASS |

## 7. Ringkasan Verifikasi Teknis

Command yang dijalankan:

```powershell
go build ./...
go test ./internal/importing/... ./internal/platform/httpserver/... ./internal/platform/migration/...
go run . migrate up
```

Hasil:

- build backend: **PASS**
- test package importing/httpserver/migration: **PASS**
- migration sprint 15 terbaru: **PASS**

Catatan Windows:

- pada environment Windows lokal, setelah package test selesai sukses kadang masih muncul warning cleanup `unlinkat ... Access is denied`;
- assertion test tetap lulus dan package tetap berstatus `ok`.

## 8. Verifikasi Side-Effect Data

Setelah batch sukses di-commit, dilakukan pengecekan ke database lokal.

Hasil yang terverifikasi:

- batch `MONTHLY_ACTIVE` menghasilkan **3 row** pada tabel `outlet_monthly_activity_snapshot`;
- batch `BONUS_MITRA` menghasilkan **1 row** pada tabel `partner_bonus_referral_snapshots`;
- batch `NEW_SUBSCRIBE` menghasilkan **1 row** pada `subscription_orders`;
- batch `SALES_TARGET` menghasilkan **1 target count** dan **1 target omset** untuk sales yang dipilih;
- batch `SALES_CALL_CHAT` menghasilkan **1 row interaction** untuk lead yang sesuai.

## 9. Catatan Penting untuk Frontend

### 9.1 Profile tertentu wajib parameter tambahan

Profile berikut **tidak cukup** hanya kirim file:

- `SALES_TARGET`
- `SALES_CALL_CHAT`

Keduanya wajib:

- `profile`
- `sheet_name`
- `target_sales_user_id`

### 9.2 Commit hanya boleh saat `status=VALIDATED`

Frontend harus memperlakukan status batch sebagai source of truth.

Alur yang aman:

1. upload file;
2. simpan `batch_id`;
3. poll `GET /imports/{id}`;
4. tunggu `status=VALIDATED`;
5. baru tampilkan tombol commit;
6. setelah commit, poll lagi sampai `COMMITTED` atau `COMMIT_FAILED`.

### 9.3 Upload file duplikat bersifat idempoten

Jika file yang sama di-upload lagi dengan kombinasi dedup key yang sama, backend akan mengembalikan batch yang sudah ada, bukan membuat batch baru.

Ini berguna untuk:

- refresh halaman;
- retry upload dari frontend;
- buka ulang histori upload lama tanpa membuat duplikasi data.

## 10. Dokumen Terkait

- [Briefing Frontend Sprint 15](./FRONTEND_BRIEFING.md) — mulai dari sini untuk integrasi frontend,
  termasuk endpoint baru `relink` dan `/imports/summary` (ditambahkan 30 Juli 2026 setelah Sprint 15a).
- [API Testing Sprint 15](./api-testing.md)
- [Sprint Report Sprint 15](./sprint-15.md)

## 11. Kesimpulan

Sprint 15 backend sekarang sudah memiliki:

- coverage manual smoke test untuk profil import utama;
- dokumentasi request/response nyata;
- dokumentasi error case nyata;
- solusi dan pencegahan error untuk frontend;
- audit runtime terhadap worker, migration, dan side-effect commit;
- histori batch import yang bisa dibaca ulang oleh frontend dan tim kantor.
