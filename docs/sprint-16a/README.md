# Sprint 16a — Historical Created Date & Export Foundation

## Ringkasan

Sprint 16a menambahkan fondasi agar CRM dapat menampung data historis perusahaan tanpa kehilangan
makna tanggal dibuat asli dari data bisnis, lalu mulai menyempurnakan export admin agar semakin
mendekati workbook operasional kantor.

Prinsip yang dipakai:

- `created_at` tetap menjadi sumber tanggal dibuat utama di backend;
- endpoint create tertentu sekarang dapat menerima `created_at` eksplisit untuk backfill data lama;
- jika `created_at` tidak dikirim, backend tetap memakai perilaku normal saat ini;
- endpoint list utama mendapat filter `created_from` dan `created_to`;
- modul reporting/export sekarang mendukung:
  - filter `created_from` dan `created_to`;
  - export `PDF` selain `CSV` dan `XLSX`;
  - report key admin tambahan yang mulai meniru workbook kantor satu per satu.

## Cakupan Modul Sprint 16a

- Customer
  - owner
  - outlet
  - global outlet
- Lead
- Catalog
  - package
  - plan
  - promotion
- Activity
  - interaction
  - training
- Wallet
  - wallet account
  - payment
  - transaction
- Closing
- Subscription
  - order
  - subscription
  - reconciliation
  - issue
- Partner
  - partner type
  - partner
  - assignment
  - interaction
  - referral
- Importing
  - import batch
  - import rows
- Reporting / Export

## Filter Baru

Filter baru yang sekarang dipakai lintas modul:

- `created_from=YYYY-MM-DD`
- `created_to=YYYY-MM-DD`

Khusus modul importing, ditambahkan juga:

- `uploaded_from=YYYY-MM-DD`
- `uploaded_to=YYYY-MM-DD`

Aturan:

- `created_from` dan `created_to` harus dipakai berpasangan;
- `uploaded_from` dan `uploaded_to` harus dipakai berpasangan;
- `created_to` dan `uploaded_to` diperlakukan inklusif pada level tanggal;
- filter ini menyasar tanggal dibuat aktual data pada tabel utama modul tersebut.

## Request Body Baru

Beberapa endpoint create sekarang menerima field opsional:

- `created_at`

Contoh:

```json
{
  "code": "OWN-HIST-0001",
  "name": "Owner Lama",
  "phone": "081234567890",
  "created_at": "2021-03-14T00:00:00Z"
}
```

Jika field ini tidak dikirim:

- data tetap tersimpan normal;
- tanggal dibuat mengikuti default sistem/database saat request diproses.

## Penyempurnaan Reporting / Export

Perubahan utama:

- `GET /api/v1/reports/{report_key}` sekarang menerima:
  - `created_from`
  - `created_to`
- `POST /api/v1/reports/exports` sekarang mendukung:
  - format `CSV`
  - format `XLSX`
  - format `PDF`

## Report Key Admin yang Sudah Ditambahkan

### `admin_owner_outlet`

Digunakan untuk rekap owner/outlet yang lebih dekat ke format admin kantor.

### `admin_new_subscribe`

Digunakan untuk rekap subscribe/aktivasi yang lebih dekat ke format admin kantor.

### `admin_nasabah_baru_provinsi`

Digunakan untuk rekap owner baru per provinsi dengan struktur yang mulai meniru workbook admin
nasabah baru berdasarkan wilayah/provinsi.

Kolom inti yang sudah disediakan:

- `year_member`
- `month_member`
- `owner_code`
- `owner_name`
- `owner_phone`
- `owner_email`
- `project_outlet`
- `city`
- `address`
- `province`

## Catatan Tentang Workbook Kantor

Sprint 16a belum mencoba menyalin seluruh workbook kantor sekaligus. Pendekatannya sengaja dibuat
bertahap:

- satu report key admin mewakili satu kebutuhan workbook;
- struktur kolom dibuat semakin dekat ke workbook nyata;
- template yang lebih kompleks tetap bisa ditambah bertahap tanpa merusak route reporting yang
  sudah dipakai frontend.

## Verifikasi

Lihat:

- [sprint-16a.md](./sprint-16a.md)
- [api-testing.md](./api-testing.md)
