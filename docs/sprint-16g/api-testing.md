# API Testing — Sprint 16g

Tanggal eksekusi: **9 Agustus 2026**

## Environment

- Backend binary lokal: `.tmp/crm.exe`
- DB smoke test: `piposmart_testing_api_20260809`
- Port API: `8099`

## Langkah Verifikasi

1. Jalankan migrasi penuh.
2. Jalankan `seed master`.
3. Jalankan `seed demo --preset=minimal --seed=1 --from=2026-01-01 --to=2026-01-01`.
4. Start API pada port `8099`.
5. Login sebagai:
   - `admin.001@demo.piposmart.id`
   - `supervisor.001@demo.piposmart.id`
6. Set `outlets.row_code = 'RB-API-001'` untuk satu outlet uji.
7. Panggil `PATCH /api/v1/owners/1/testing-account` dengan body `{ "is_testing_account": true }`.
8. Verifikasi:
   - `GET /api/v1/leads?all=true` tidak lagi memuat lead `id=1`
   - `GET /api/v1/owners/1` sebagai supervisor mengembalikan `404`
   - `POST /api/v1/leads/1/assign-sales` sebagai supervisor mengembalikan `400`
   - `GET /api/v1/owners/export/download` mengembalikan file `.xlsx` valid
9. Buka zip `.xlsx` dan cek `xl/sharedStrings.xml` / `xl/worksheets/sheet1.xml`.

## Hasil

- `PATCH /owners/1/testing-account` → `200 OK`
- `GET /leads?all=true` → lead owner testing tidak muncul lagi
- `GET /owners/1` sebagai supervisor → `404`
- `POST /leads/1/assign-sales` sebagai supervisor → `400`
- export `.xlsx` mengandung:
  - `Akun Testing`
  - `RB-API-001`

## Bug Audit yang Ditemukan Saat Smoke Test

Saat pengujian pertama, `POST /leads/1/assign-supervisor` sempat gagal `500` dengan error:

`Column 'status' in field list is ambiguous`

Penyebab:

- query `lockLead` baru join ke tabel `owners`
- kolom `status` tidak di-prefix `cl.`

Status:

- diperbaiki pada turn yang sama
- smoke test sesudah patch berjalan lanjut tanpa error server pada jalur testing-account
