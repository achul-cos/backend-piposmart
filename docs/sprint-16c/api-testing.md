# API Testing Report — Sprint 16c

## Informasi Testing

| Item | Nilai |
| --- | --- |
| Sprint | 16c |
| Tanggal Testing | 4 Agustus 2026 |
| Environment | Local Development |
| Fokus | Arsitektur Lead per-outlet, Multi-Outlet support, SQL Count FIX, dan Filter Search |

## Summary

| Area | Status | Catatan |
| --- | --- | --- |
| Build backend | PASS | `go build ./...` berhasil tanpa error |
| Type check frontend | PASS | `npx tsc --noEmit` berhasil tanpa error (0 errors) |
| Multi-Outlet Lead query | PASS | 1 Owner dengan 2 outlet mengembalikan 2 baris lead terpisah |
| SQL Count Query Fix | PASS | `SELECT COUNT(*)` mengembalikan jumlah total outlet aktif dengan tepat |

## Command Verifikasi

```bash
# Verify Go build
cd backend/backend_crm_piposmart
go build ./...

# Verify Frontend TypeScript
cd frontend/crm_piposmart
npx tsc --noEmit
```

## Skenario Utama API Testing

### 1. Mengambil Daftar Lead per Outlet

Request:

```http
GET /api/v1/leads?page=1&limit=10
Authorization: Bearer {access_token}
Content-Type: application/json
```

Response Skenario (Owner "deny" memiliki 2 outlet: "deny mart" dan "deny pos"):

```json
{
  "items": [
    {
      "id": 1,
      "code": "LEAD-000001",
      "owner": {
        "id": 12,
        "code": "OWN-0012",
        "name": "deny",
        "phone": "08123456789",
        "brand_name": "Deny Group"
      },
      "outlet_id": 21,
      "outlet": {
        "id": 21,
        "code": "OUT-0021",
        "name": "deny mart",
        "phone": "08123456789"
      },
      "stage": "NEW",
      "status": "OPEN",
      "current_score": 1
    },
    {
      "id": 2,
      "code": "LEAD-000002",
      "owner": {
        "id": 12,
        "code": "OWN-0012",
        "name": "deny",
        "phone": "08123456789",
        "brand_name": "Deny Group"
      },
      "outlet_id": 22,
      "outlet": {
        "id": 22,
        "code": "OUT-0022",
        "name": "deny pos",
        "phone": "08123456789"
      },
      "stage": "NEW",
      "status": "OPEN",
      "current_score": 1
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 2
  }
}
```

Ekspektasi:
- Setiap outlet yang dimiliki oleh owner tampil sebagai entitas Lead tersendiri.
- `total` pagination mencerminkan jumlah total outlet aktif.

### 2. Pencarian Lead berdasarkan Nama Outlet / Kode Outlet

Request:

```http
GET /api/v1/leads?q=deny+mart
Authorization: Bearer {access_token}
```

Ekspektasi:
- Query pencarian mencocokkan kata kunci pada `ot.name` dan `ot.code`.
- Mengembalikan Lead yang memiliki nama outlet "deny mart".

## Error Handling

Jika terjadi kesalahan internal query database, handler mencatat `slog.Error` dan mengembalikan response terstruktur:

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Terjadi kesalahan pada server"
  }
}
```
