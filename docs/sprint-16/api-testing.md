# API Testing Report - Sprint 16 Reporting & Export

## 1. Informasi Testing

| Item | Nilai |
| --- | --- |
| Sprint | 16 |
| Tanggal Testing | 2 Agustus 2026 |
| Environment | Local Development |
| Fokus | Validasi compile, vet, wiring route, worker export, dan sanitasi spreadsheet |

## 2. Cakupan Verifikasi

Sprint 16 pada turn ini diverifikasi terutama pada level backend compile/runtime contract:

- route reporting berhasil terdaftar di codebase;
- service reporting berhasil di-wire ke router;
- job export berhasil di-wire ke worker registry;
- builder CSV/XLSX berhasil dikompilasi;
- proteksi formula injection memiliki unit test;
- seluruh repository tetap lulus `go test ./...`.

## 3. Hasil Testing Teknis

### 3.1 Unit / Package Test

Command:

```bash
go test ./...
```

Hasil:

- Seluruh package test lulus, termasuk package baru:
  - `internal/reporting`
- Catatan: pada environment Windows, Go masih gagal menghapus file binary test sementara
  (`*.test.exe`) sehingga process berakhir dengan pesan `Access is denied`, walau seluruh hasil test
  package menunjukkan `ok`.

### 3.2 Vet

Command:

```bash
go vet ./...
```

Hasil:

- PASS

### 3.2b Focused Sprint 16 Package Test

Command:

```bash
go test ./internal/reporting ./internal/platform/httpserver
```

Hasil:

- PASS
- Membuktikan package baru Sprint 16 dan route OpenAPI/httpserver terkait berhasil dikompilasi dan dites secara terfokus.

### 3.3 Build

Command:

```bash
go build .
```

Hasil:

- PASS

### 3.4 OpenAPI Validation

Command:

```bash
npx -y @apidevtools/swagger-cli validate internal/platform/httpserver/openapi.yaml
```

Hasil:

- PASS
- Spec OpenAPI Sprint 16 valid dan dapat dirender Swagger/OpenAPI viewer.

## 4. Endpoint yang Ditambahkan

| Endpoint | Method | Status Verifikasi |
| --- | --- | --- |
| `/api/v1/reports/dashboard` | GET | Ter-wire |
| `/api/v1/reports/owners_outlets` | GET | Ter-wire |
| `/api/v1/reports/activities` | GET | Ter-wire |
| `/api/v1/reports/topups` | GET | Ter-wire |
| `/api/v1/reports/closings` | GET | Ter-wire |
| `/api/v1/reports/subscriptions` | GET | Ter-wire |
| `/api/v1/reports/partners` | GET | Ter-wire |
| `/api/v1/reports/targets_kpi` | GET | Ter-wire |
| `/api/v1/reports/exports` | POST | Ter-wire |
| `/api/v1/reports/exports` | GET | Ter-wire |
| `/api/v1/reports/exports/{id}` | GET | Ter-wire |
| `/api/v1/reports/exports/{id}/download` | GET | Ter-wire |

## 5. Proteksi Error yang Sudah Ada

Error penting yang sudah di-handle:

- `FORBIDDEN`
- `VALIDATION_ERROR`
- `NOT_FOUND`
- `EXPORT_NOT_READY`
- `INTERNAL_ERROR`

## 6. Contoh Error Scenario

### 6.1 Report key tidak didukung

Contoh request:

```http
GET /api/v1/reports/unknown_report
Authorization: Bearer {access_token}
```

Ekspektasi:

- `400 VALIDATION_ERROR`

### 6.2 Format export tidak valid

Contoh request:

```json
{
  "report_key": "closings",
  "format": "PDF"
}
```

Ekspektasi:

- `400 VALIDATION_ERROR`

### 6.3 Download export sebelum selesai

Contoh request:

```http
GET /api/v1/reports/exports/5/download
Authorization: Bearer {access_token}
```

Dengan kondisi:

- export status masih `PENDING` atau `PROCESSING`.

Ekspektasi:

- `409 EXPORT_NOT_READY`

## 7. Kesimpulan

Sprint 16 backend pada turn ini sudah valid pada level:

- schema;
- route wiring;
- service wiring;
- worker wiring;
- compile/test/vet;
- keamanan export dasar.

Testing fungsional HTTP end-to-end dengan database hidup dan file export nyata dapat diperluas lagi
sebagai langkah dokumentasi lanjutan bila dibutuhkan.
