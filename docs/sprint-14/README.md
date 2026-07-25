# API Testing Report - Sprint 14 Import Framework dan Data Customer

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 14 - Import Framework dan Data Customer |
| Tanggal Testing | 26 Juli 2026 |
| Environment | Local Development, terisolasi (`test_sprint14`, port `8093`) |
| API Base URL | `http://localhost:8093/api/v1` |
| Database | `test_sprint14` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via `curl`, **memakai file Excel asli kantor** (bukan fixture buatan) |
| Database Migration | `go run . migrate up` (`20260726000100_import_framework.sql`) |
| Seeder | `go run . seed master` (data referensi) + `go run . seed demo --preset=minimal` (akun Sales untuk uji RBAC) |
| Worker | `go run . worker` (`WORKER_POLL_INTERVAL=2s`) |
| File uji nyata | `c:/piposmart/data_admin/01. Owner & Outlet 2026 (Copy).xlsx` (105 baris) dan `04. Data Belum Registrasi 2026 - User Temp (Copy).xlsx` (58 baris) — salinan workspace kantor sesungguhnya |

## 2. Header Pengujian

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data (untuk upload)
```

Akun demo yang digunakan:

| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin@piposmart.id` | `ChangeMe123!` (bootstrap-admin) |
| Sales (uji RBAC) | `sales.001@demo.piposmart.id` | `Password123!` |

## 3. Scope Pengujian

- Upload file Excel (.xlsx) dengan dedup berbasis SHA-256.
- Deteksi profil otomatis dari header kolom (dibangun dari header asli, bukan tebakan) dan deklarasi profil manual sebagai fallback.
- Validasi async lewat `job_queue` (worker) — normalisasi telepon, tanggal serial Excel, dan strip kolom OTP.
- Alur staging: upload → validasi (preview) → commit, dengan guard status yang benar.
- Commit yang membuat Owner/Outlet/Lead nyata lewat service yang sudah ada (`customer.Service`, `lead.Service`), bukan SQL baru.
- Export baris invalid sebagai CSV.
- Role gating: seluruh endpoint import ADMIN saja.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Upload | Upload `01. Owner & Outlet 2026.xlsx` tanpa `profile` param | POST `/imports` | 201, `profile=PENDING_DETECTION` awal | PASS |
| Validate | Worker mendeteksi profil otomatis dari header kolom asli | (async) | `profile=OWNER_OUTLET`, `status=VALIDATED` | PASS |
| Validate | Split baris valid/invalid dari data nyata (105 baris sumber, beberapa baris kosong) | (async) | `total_rows=96`, `valid_rows=91`, `invalid_rows=5` | PASS |
| Upload | Upload `04. Data Belum Registrasi.xlsx` dengan `profile=NON_REGISTER` eksplisit | POST `/imports` | 201, header row terdeteksi di row 2 (row 1 judul gabungan) | PASS |
| Validate | Split baris valid/invalid pada data kotor (nomor telepon `"Tidak Tersedia"`, dll) | (async) | `total_rows=56`, `valid_rows=48`, `invalid_rows=8` | PASS |
| Security | Kolom `Kode OTP`/`Created Date Kode OTP` tidak pernah masuk `raw_payload` | Query SQL langsung | 0 baris mengandung nilai OTP asli (`0264`, `0137`, dll) dari **kedua** batch | PASS |
| Upload | Re-upload file yang identik (SHA-256 sama) | POST `/imports` | Mengembalikan batch yang sama (id tidak berubah), tidak ada duplikasi | PASS |
| Commit | Commit batch sebelum tervalidasi / sudah COMMITTED | POST `/imports/{id}/commit` | 400 `INVALID_BATCH_STATUS` | PASS |
| Commit | Commit batch `OWNER_OUTLET` VALIDATED | POST `/imports/{id}/commit` | Async, `COMMITTING` → `COMMITTED`, `committed_rows=91/91` | PASS |
| Commit | Commit batch `NON_REGISTER` VALIDATED | POST `/imports/{id}/commit` | Async, `COMMITTED`, `committed_rows=48/48` | PASS |
| RBAC | Sales mencoba upload | POST `/imports` (Sales) | 403 FORBIDDEN | PASS |
| Export | Unduh baris invalid sebagai CSV | GET `/imports/{id}/rejected-rows/export` | File CSV valid berisi `row_index, raw_payload, validation_errors` | PASS |

## 5. Detail Skenario Pengujian

### 5.1 Deteksi Profil Otomatis dari Header Asli

Marker header diambil langsung dari kedua file asli (bukan diasumsikan):

- `OWNER_OUTLET`: kolom `Kode Owner` + `Nama Owner` + `Nama Outlet` harus ada bersamaan.
- `NON_REGISTER`: kolom `Nomor Telepon` + `Kode OTP` + `Status Akun` harus ada bersamaan.

File `04. Data Belum Registrasi...xlsx` punya baris judul gabungan ("DATA NON REGISTER BULAN JULI") di row 1 — header sungguhan ada di row 2. Sistem memindai 5 baris pertama untuk menemukan baris manapun yang cocok, bukan mengasumsikan header selalu di row 1. Terverifikasi lewat unit test (`TestDetectProfile_NonRegister_HeaderNotOnFirstRow`) dan smoke test nyata.

### 5.2 Keamanan OTP — Diverifikasi terhadap Nilai Asli

File `04. Data Belum Registrasi...xlsx` memuat kode OTP **sungguhan** di kolom `Kode OTP` (`0264`, `0137`, `0438`, `0572`, dst — terlihat langsung saat inspeksi file sebelum implementasi). Setelah commit kedua batch:

```sql
SELECT COUNT(*) FROM import_rows
WHERE raw_payload LIKE '%0264%' OR raw_payload LIKE '%0137%'
   OR raw_payload LIKE '%0438%' OR raw_payload LIKE '%0572%';
-- hasil: 0
```

`JSON_KEYS(raw_payload)` untuk batch Non-Register hanya pernah berisi `phone`, `remarks`, `status_akun`, `date_of_work` — tidak pernah `kode_otp` atau sejenisnya. Mekanismenya dua lapis: (1) index header yang dipakai untuk ekstraksi nilai (`buildHeaderIndex`) secara eksplisit membuang entri OTP, terpisah dari index yang dipakai untuk deteksi profil (`buildFullHeaderIndex`) yang butuh tahu OTP ada tapi tidak pernah membaca isinya. ✅

### 5.3 Bug Ditemukan Saat Testing dengan Data Nyata (dan Perbaikannya)

Testing terhadap file asli (bukan fixture buatan) menemukan **empat bug nyata** yang tidak akan ketahuan dari data sintetis:

1. **CHECK constraint menolak placeholder profil.** Migration awal hanya mengizinkan `profile IN ('OWNER_OUTLET', 'NON_REGISTER')`, padahal upload tanpa `profile` eksplisit butuh nilai sementara sebelum worker mendeteksinya. Diperbaiki: tambah `PENDING_DETECTION` ke daftar yang diizinkan (bukan mengubah ke nullable, supaya kode Go tetap sederhana).
2. **`buildHeaderIndex` yang membuang kolom OTP juga dipakai untuk deteksi profil** — akibatnya "Kode OTP" (penanda profil Non-Register) tidak pernah bisa ditemukan sama sekali, deteksi selalu gagal. Diperbaiki dengan memisah dua index: satu lengkap khusus pencocokan penanda profil, satu lagi (aman, tanpa OTP) untuk ekstraksi nilai — ditangkap oleh unit test sebelum sempat lolos ke smoke test.
3. **`json.RawMessage` tidak bisa men-scan `NULL`** dari kolom `validation_errors` (kosong untuk baris valid) — job `IMPORT_COMMIT` gagal retry 5x dengan error `unsupported Scan ... into type *json.RawMessage`. Diperbaiki: ganti tipe field jadi `sql.NullString`, konversi eksplisit ke `json.RawMessage` hanya saat membangun response.
4. **Format tanggal nyata (`"01/07/26"`, DD/MM/YY 2-digit tahun) tidak dikenali** parser awal (yang cuma expect ISO/serial number) — menyebabkan SEMUA baris Owner & Outlet gagal validasi. Diperbaiki dua arah: (a) tambah layout `DD/MM/YY` ke parser tanggal, (b) `date_of_work` dijadikan **informasional, non-fatal** — gagal parse tanggal tidak lagi menggagalkan seluruh baris, karena field ini tidak pernah dipakai untuk membuat Owner/Outlet/Lead.
5. **`customer.Service.CreateOwner` ternyata sudah otomatis membuat Lead** sebagai bagian dari pembuatan Owner (perilaku existing sejak Sprint 4) — pemanggilan eksplisit `lead.Service.CreateLead` pada baris Non-Register SELALU bentrok kode (`LEAD-%06d` diturunkan dari `owner_id`, jadi satu owner cuma boleh punya satu lead). Diperbaiki: hapus pemanggilan `CreateLead` yang mustahil berhasil, cukup cari lead yang sudah otomatis dibuat dan catat ID-nya — sekaligus membuat baris dengan nomor telepon duplikat antar-baris menjadi idempoten secara alami.

Semua lima ditemukan **karena** testing memakai file Excel produksi asli, bukan data buatan sendiri — data buatan tidak akan pernah punya kombinasi format tanggal, kolom kosong, dan pola bisnis (satu lead per owner) yang persis seperti ini.

### 5.4 Hasil Akhir Setelah Perbaikan

```text
Batch 1 (Owner & Outlet): total=96, valid=91, invalid=5, committed=91/91
Batch 2 (Non-Register):   total=56, valid=48, invalid=8, committed=48/48

Database setelah commit:
  owners: 138 (90 dari batch 1 + 48 dari batch 2)
  outlets: 91 (termasuk 1 owner dengan 2 outlet — reuse kode owner terverifikasi)
  customer_leads: 138 (1 per owner, sesuai perilaku CreateOwner)
```

5 baris invalid batch 1: 2 nomor telepon owner kosong, 3 nomor telepon outlet berisi placeholder `"-"` — semuanya masalah data asli, bukan bug. 8 baris invalid batch 2: nomor telepon rusak (termasuk literal `"Tidak Tersedia"`).

## 6. Kesimpulan Pengujian

Seluruh skenario inti Sprint 14 (upload, dedup SHA-256, deteksi profil otomatis dari header asli, validasi async, strip OTP, commit async yang membuat entitas nyata, guard status, RBAC, export CSV) tervalidasi PASS terhadap **file Excel produksi asli** — bukan fixture sintetis. `go build`, `go vet`, `go test ./...` bersih; migration reversibel (up/down/up teruji). Lima bug nyata ditemukan dan diperbaiki selama testing (§5.3), seluruhnya berasal dari perbedaan antara asumsi desain awal dan bentuk data/perilaku sistem yang sesungguhnya — nilai konkret dari kebijakan "test dengan data asli, bukan cuma fixture buatan".
