# API Testing Report — Sprint 16a

## Informasi Testing

| Item | Nilai |
| --- | --- |
| Sprint | 16a |
| Tanggal Testing | 3 Agustus 2026 |
| Environment | Local Development |
| Fokus | historical created date, filter `created_from/created_to`, filter `uploaded_from/uploaded_to`, export PDF, dan template report admin awal |

## Summary

| Area | Status | Catatan |
| --- | --- | --- |
| Build backend | PASS | `go build .` berhasil |
| Test package inti yang berubah | PASS | reporting, partner, importing |
| Validasi OpenAPI | PASS | YAML valid setelah patch |
| Known environment note | WARN | Windows kadang menahan file `.exe` hasil `go test` |

## Command Verifikasi

```bash
go build .
go test ./internal/partner ./internal/importing ./internal/reporting
npx -y @apidevtools/swagger-cli validate internal/platform/httpserver/openapi.yaml
```

## Skenario Utama

### 1. Create Partner historis

Request:

```http
POST /api/v1/partner-types
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "code": "COMMUNITY",
  "name": "Komunitas Laundry",
  "commission_mode": "PERCENTAGE",
  "commission_value": "5.00",
  "description": "Partner komunitas daerah",
  "created_at": "2021-01-10T00:00:00Z"
}
```

Ekspektasi:

- partner type tersimpan;
- `created_at` historis dipakai sesuai payload.

### 2. List Partner dengan filter tanggal dibuat

Request:

```http
GET /api/v1/partners?search=mitra&created_from=2021-01-01&created_to=2021-12-31&limit=10&offset=0
Authorization: Bearer {access_token}
```

Ekspektasi:

- hanya partner yang `created_at`-nya masuk ke rentang itu yang tampil;
- search tetap bekerja bersamaan dengan filter tanggal dibuat.

### 3. List history import berdasarkan tanggal upload

Request:

```http
GET /api/v1/imports?profile=OWNER_OUTLET&uploaded_from=2026-07-01&uploaded_to=2026-07-31
Authorization: Bearer {access_token}
```

Ekspektasi:

- batch import terfilter berdasarkan tanggal file di-upload;
- frontend bisa membuat histori upload lintas sesi dengan filter yang konsisten.

### 4. List rows import berdasarkan tanggal dibuat

Request:

```http
GET /api/v1/imports/12/rows?status=VALID&created_from=2026-07-01&created_to=2026-07-31
Authorization: Bearer {access_token}
```

Ekspektasi:

- row import hanya mengembalikan data yang `created_at`-nya sesuai rentang.

### 5. Report generic dengan filter created date

Request:

```http
GET /api/v1/reports/owners_outlets?date_from=2026-07-01&date_to=2026-07-31&created_from=2020-01-01&created_to=2023-12-31
Authorization: Bearer {access_token}
```

Ekspektasi:

- report tetap memakai rentang report period;
- backend juga mempersempit data berdasarkan `created_at`.

### 6. Export PDF

Request:

```http
POST /api/v1/reports/exports
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "report_key": "owners_outlets",
  "format": "PDF",
  "filters": {
    "date_from": "2026-07-01",
    "date_to": "2026-07-31",
    "created_from": "2020-01-01",
    "created_to": "2023-12-31"
  }
}
```

Ekspektasi:

- job export dibuat;
- file dapat diunduh dalam mime type `application/pdf`.

### 7. Report admin nasabah baru per provinsi

Request:

```http
GET /api/v1/reports/admin_nasabah_baru_provinsi?date_from=2026-01-01&date_to=2026-12-31&province=JAWA%20TIMUR
Authorization: Bearer {access_token}
```

Ekspektasi:

- backend mengembalikan kolom rekap owner baru per provinsi;
- struktur kolom lebih dekat ke workbook admin wilayah/provinsi.

## Error Handler yang Diverifikasi

### A. `created_from` tanpa `created_to`

Request:

```http
GET /api/v1/reports/owners_outlets?created_from=2021-01-01
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "reporting: filter tidak valid: created_from dan created_to harus diisi berpasangan"
  }
}
```

Solusi frontend:

- selalu kirim `created_from` dan `created_to` sekaligus.

### B. `uploaded_from` tanpa `uploaded_to`

Request:

```http
GET /api/v1/imports?uploaded_from=2026-07-01
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "uploaded_from dan uploaded_to harus diisi berpasangan"
  }
}
```

Solusi frontend:

- buat komponen filter upload date selalu mengisi start dan end sekaligus.

### C. format tanggal salah

Request:

```http
GET /api/v1/partners?created_from=01-01-2021&created_to=2021-12-31
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "format tanggal harus YYYY-MM-DD"
  }
}
```

Solusi frontend:

- gunakan formatter tunggal `YYYY-MM-DD`.

### D. format export tidak valid

Request:

```json
{
  "report_key": "owners_outlets",
  "format": "DOCX"
}
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "reporting: format export tidak didukung"
  }
}
```

Solusi frontend:

- batasi pilihan ke `CSV`, `XLSX`, atau `PDF`.

### E. report key tidak dikenal

Request:

```http
GET /api/v1/reports/admin_unknown_template?date_from=2026-01-01&date_to=2026-12-31
```

Response:

```json
{
  "error": {
    "code": "REPORT_NOT_FOUND",
    "message": "reporting: report key tidak dikenal"
  }
}
```

Solusi frontend:

- gunakan dropdown berbasis enum report key dari OpenAPI atau hardcoded whitelist yang sama.

## Kesimpulan

Sprint 16a sudah valid untuk tiga tujuan utama:

- backfill data historis melalui `created_at`;
- filtering data historis lintas modul, termasuk histori importing;
- fondasi export admin bertahap dengan template report yang makin mendekati workbook kantor.
