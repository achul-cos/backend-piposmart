# API Documentation Update - Sprint 16 Dashboard, Reporting, dan Async Export

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 16 |
| Tanggal Update | 2 Agustus 2026 |
| Environment | Local Development |
| Fokus | Dashboard reporting, report list, export async CSV/XLSX, dan proteksi spreadsheet formula injection |

## 2. Ringkasan Sprint 16

Sprint 16 backend pada update ini berfokus pada pembuatan modul reporting baru yang bisa dipakai
frontend untuk:

- menampilkan dashboard ringkas berdasarkan role user;
- menampilkan list report utama lintas modul;
- membuat export report secara asinkron melalui worker;
- memantau status export;
- mengunduh file hasil export dalam format `CSV` atau `XLSX`;
- mencegah spreadsheet formula injection saat file dibuka di Excel/Spreadsheet.

## 3. Endpoint Baru Sprint 16

### 3.1 Dashboard

- `GET /api/v1/reports/dashboard`

Filter:

- `date_from=YYYY-MM-DD`
- `date_to=YYYY-MM-DD`

Output utama:

- role actor;
- rentang tanggal efektif;
- kartu summary:
  - total owner;
  - total outlet;
  - langganan aktif;
  - jumlah topup diterima;
  - revenue topup;
  - omset closing confirmed;
  - jumlah closing confirmed.

### 3.2 Report List

- `GET /api/v1/reports/owners_outlets`
- `GET /api/v1/reports/activities`
- `GET /api/v1/reports/topups`
- `GET /api/v1/reports/closings`
- `GET /api/v1/reports/subscriptions`
- `GET /api/v1/reports/partners`
- `GET /api/v1/reports/targets_kpi`

Query umum:

- `date_from`
- `date_to`
- `status`
- `q`
- `province`
- `city`
- `page`
- `limit`
- `all=true`

Respons dibuat generik agar frontend mudah membangun tabel:

- `report_key`
- `columns`
- `items`
- `pagination`
- `insight`

### 3.3 Export Center

- `POST /api/v1/reports/exports`
- `GET /api/v1/reports/exports`
- `GET /api/v1/reports/exports/{id}`
- `GET /api/v1/reports/exports/{id}/download`

Format export yang didukung:

- `CSV`
- `XLSX`

Job export diproses lewat worker dengan `job_type`:

- `REPORT_EXPORT_GENERATE`

Status export:

- `PENDING`
- `PROCESSING`
- `COMPLETED`
- `FAILED`

## 4. Report yang Tersedia

| Report Key | Fungsi |
| --- | --- |
| `owners_outlets` | Rekap owner, brand, wilayah, outlet count, dan saldo wallet |
| `activities` | Rekap interaksi customer berdasarkan call/chat/follow up |
| `topups` | Rekap payment topup owner |
| `closings` | Rekap closing sales dan nominal closing |
| `subscriptions` | Rekap status subscription owner/outlet |
| `partners` | Rekap mitra, komisi, dan payout |
| `targets_kpi` | Rekap target KPI dan ranking sales |

## 5. Keamanan Export

Sprint 16 ini menambahkan proteksi formula injection untuk seluruh sel spreadsheet export.

String yang diawali karakter berbahaya seperti:

- `=`
- `+`
- `-`
- `@`

akan diprefix `'` sebelum ditulis ke file export.

Tujuannya agar data yang berasal dari input user tidak dieksekusi sebagai formula saat dibuka di
Excel atau aplikasi spreadsheet lain.

## 6. Artefak Teknis yang Ditambahkan

### Migration

- `migrations/20260802000200_reporting_exports.sql`

Tabel baru:

- `report_exports`

### Package Baru

- `internal/reporting/`

Isi utama:

- repository query reporting;
- service dashboard & export;
- handler HTTP reporting;
- worker handler export async;
- builder CSV/XLSX;
- proteksi formula injection.

## 7. Verifikasi yang Sudah Dilakukan

Pada 2 Agustus 2026, validasi lokal yang sudah berhasil:

- `go test ./...` seluruh package test lulus, tetapi environment Windows masih memunculkan kendala pembersihan file `.test.exe` sementara saat proses selesai
- `go vet ./...` PASS
- `go build .` PASS
- `npx -y @apidevtools/swagger-cli validate internal/platform/httpserver/openapi.yaml` PASS

Catatan:

- pada akhir `go test ./...`, environment Windows sempat menampilkan pesan temp-file unlink
  `Access is denied` dari cache build Go / temp test binary. Secara output, seluruh package test
  dinyatakan PASS, namun proses Go tetap keluar dengan exit non-zero karena gagal menghapus file
  sementara yang sedang terkunci oleh environment Windows.

## 8. Catatan Scope Sprint 16 pada Update Ini

Update ini sudah menutup fondasi utama Sprint 16 backend:

- reporting API;
- dashboard summary;
- export async;
- storage file export;
- worker integration;
- formula injection protection.

Yang masih bisa diperdalam pada turn berikutnya bila diperlukan:

- API testing report Sprint 16 yang lebih panjang dengan contoh request/response/error handler;
- export tambahan ke PDF/PNG bila ingin disamakan dengan ekosistem analytics 14g;
- dashboard summary per role yang lebih kaya dan lebih spesifik KPI bisnis.
