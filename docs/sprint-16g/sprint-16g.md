# Report Laporan Sprint 16g

## Fokus

Melanjutkan pekerjaan migrasi data admin dan menutup gap bisnis/teknis pada:

- persistensi `Kode Baris`
- persistensi dan pengelolaan `Akun Testing`
- export admin owner/outlet yang sebelumnya masih membaca file Excel mentah
- pemblokiran akun testing dari pipeline lead/sales

## Perubahan Utama

### 1. Skema database baru

Migration baru: `20260808000100_add_testing_account_and_row_code.sql`

- `owners.is_testing_account`
- `owners.testing_marked_by_user_id`
- `owners.testing_marked_at`
- `outlets.row_code`

Tujuan:

- `is_testing_account` menjadi penanda resmi bahwa owner tersebut bukan prospek nyata.
- `testing_marked_by_user_id/testing_marked_at` menjadi audit trail saat admin menandai owner.
- `row_code` menyimpan `Kode Baris` asli dari file admin per outlet-row.

### 2. Seeder real dan archive subscribe

Perubahan pada `internal/platform/seeder/seeder_real.go` dan `seeder_subscribe.go`:

- `Kategori Akun = "Akun Testing"` dari file `01. Owner & Outlet` sekarang meng-set
  `owners.is_testing_account = true`.
- `Kode Baris` dari file `01. Owner & Outlet` sekarang masuk ke `outlets.row_code`.
- owner testing yang berasal dari import **tidak lagi dibuatkan `customer_leads`**.
- arsip `New & Subscribe` 2021–2026 tetap memprioritaskan relasi ke owner/outlet existing:
  1. by `Kode Owner`
  2. fallback by normalized phone
  3. baru create owner/outlet jika memang tidak ditemukan

### 3. Endpoint admin untuk Akun Testing

Endpoint baru:

- `PATCH /api/v1/owners/:owner_id/testing-account`

Body:

```json
{
  "is_testing_account": true
}
```

Perilaku:

- hanya role yang bisa manage owner yang boleh mengubah flag ini
- saat `true`, `testing_marked_by_user_id` dan `testing_marked_at` terisi
- saat `false`, audit mark dibersihkan kembali

### 4. Guard pipeline lead/sales

Perubahan pada `internal/lead` dan `internal/customer`:

- akun testing **tidak muncul di list lead**
- akun testing **tidak bisa dibuat lead manual**
- akun testing **tidak bisa di-assign ke supervisor/sales**
- akun testing **tidak terlihat oleh supervisor/sales di modul owner**

Error baru:

- `akun testing tidak boleh masuk pipeline lead/sales`

Catatan audit:

- saat menghubungkan flag testing ke `lockLead`, ditemukan bug nyata pada 9 Agustus 2026:
  query `assign-supervisor` gagal `500` karena kolom `status` menjadi ambigu setelah join ke `owners`.
  Ini diperbaiki dengan meng-qualify kolom `cl.status`, `cl.stage`, `cl.active_sales_id`, dan
  `cl.current_score`.

### 5. Export owner/outlet DB-driven

Perubahan pada `internal/customer` + `internal/reporting`:

- handler download Excel tidak lagi membaca file Excel sumber dari disk
- export mengambil data langsung dari DB (`ExportOwnerOutlets`)
- builder styling admin lama tetap dipakai (`BuildAdminOwnerOutletXLSX`)
- `internal/reporting/export_from_excel.go` dihapus karena sudah menjadi jalur lama yang bertentangan
  dengan state database CRM

Perubahan isi export:

- kolom `Kode Baris` sekarang diambil dari `outlets.row_code`
- `Kategori Akun` sekarang:
  - `Akun Testing` jika owner flagged testing
  - selain itu tetap derived seperti pola admin lama (`Akun Baru` / `Outlet Baru`)

## File yang Diubah

- `migrations/20260808000100_add_testing_account_and_row_code.sql`
- `internal/platform/factory/factory.go`
- `internal/platform/seeder/seeder_real.go`
- `internal/platform/seeder/seeder_subscribe.go`
- `internal/customer/handler.go`
- `internal/customer/repository.go`
- `internal/customer/service.go`
- `internal/customer/types.go`
- `internal/customer/export_admin_format.go`
- `internal/lead/errors.go`
- `internal/lead/repository.go`
- `internal/lead/handler.go`
- `internal/reporting/export.go`
- `internal/reporting/export_from_excel.go` (dihapus)

## Testing dan Verifikasi

### Build/Test

Dijalankan pada 9 Agustus 2026:

- `go build ./...` — bersih
- `go test ./internal/customer ./internal/lead ./internal/platform/seeder ./internal/reporting` — seluruh
  suite lolos
- `go test ./...` secara fungsional seluruh package lolos, tetapi pada Windows proses `go test` kadang
  berakhir non-zero karena gagal menghapus file exe temporary (`unlinkat ... Access is denied`) setelah
  test selesai

### Smoke test API

DB integrasi:

- `piposmart_testing_api_20260809`

Hasil:

- migrasi sampai `20260808000100_add_testing_account_and_row_code.sql` sukses
- `PATCH /owners/1/testing-account` mengembalikan `is_testing_account = true`
- lead owner tersebut langsung hilang dari `GET /leads?all=true`
- `GET /owners/1` sebagai supervisor menjadi `404`
- `POST /leads/1/assign-sales` sebagai supervisor menjadi `400`
- `GET /owners/export/download` menghasilkan `.xlsx` valid yang memuat:
  - string `Akun Testing`
  - string `RB-API-001` pada kolom `Kode Baris`

## Catatan Operasional

Verifikasi penuh `seed demo --preset=real` ke DB audit terpisah `piposmart_testing_audit_20260809`
dijalankan pada 9 Agustus 2026, tetapi selama jendela verifikasi turn ini seed masih berada dalam
transaksi panjang dan belum commit saat dicek dari session lain. Karena itu:

- perubahan seeder tetap divalidasi lewat build + package test + audit kode
- verifikasi HTTP/live dilakukan pada DB integrasi minimal yang selesai penuh
- apabila tim ingin bukti commit akhir corpus real 2021–2026, seed audit perlu dibiarkan selesai di
  proses terpisah atau dipecah menjadi job verifikasi tersendiri
