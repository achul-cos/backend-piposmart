# API Testing Report - Sprint 05

## Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 05 - Lead dan Assignment Sales |
| Tanggal Testing | 23 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:18081/api/v1` saat smoke test; default project tetap mengikuti `APP_PORT` |
| Testing Tool | Manual smoke test via PowerShell |
| Migration Version | `20260723000400` |

## Testing Summary

| Module | Total Case | Passed | Failed | Status |
| --- | ---: | ---: | ---: | --- |
| Lead Ownership Flow | 11 | 11 | 0 | PASS |
| **Total** | **11** | **11** | **0** | **PASS** |

## Header

```http
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

## Endpoint Utama

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Lead | GET | `/leads` | List lead sesuai visibility actor. |
| Create Lead | POST | `/leads` | Membuat lead dari owner yang sudah ada. |
| Detail Lead | GET | `/leads/{lead_id}` | Detail lead sesuai visibility actor. |
| Assignment History | GET | `/leads/{lead_id}/assignment-history` | Riwayat perpindahan ownership. |
| Assign Supervisor | POST | `/leads/{lead_id}/assign-supervisor` | Admin membagikan lead ke Supervisor. |
| Assign Sales | POST | `/leads/{lead_id}/assign-sales` | Supervisor membagikan lead ke Sales. |
| Release | POST | `/leads/{lead_id}/release` | Supervisor/Admin mengembalikan lead ke Admin. |
| Mark Invalid | POST | `/leads/{lead_id}/mark-invalid` | Sales menandai lead invalid dan mengembalikan ke Supervisor. |
| Bulk Assign Supervisor | POST | `/leads/bulk/assign-supervisor` | Bulk assign lead ke Supervisor. |
| Bulk Assign Sales | POST | `/leads/bulk/assign-sales` | Bulk assign lead ke Sales. |
| Bulk Release | POST | `/leads/bulk/release` | Bulk release lead ke Admin. |

## Query Parameter `GET /leads`

| Param | Contoh | Keterangan |
| --- | --- | --- |
| `q` | `Laundry` | Search lead/owner. |
| `ownership` | `SALES` | Filter current owner role. |
| `stage` | `POSSIBLE` | Filter stage. |
| `status` | `OPEN` | Filter status. |
| `score` | `0` | Filter score 0-3. |
| `supervisor_id` | `2` | Filter supervisor. |
| `sales_id` | `3` | Filter active sales. |
| `follow_up_from` | `2026-07-01` | Filter follow-up awal. |
| `follow_up_to` | `2026-07-31` | Filter follow-up akhir. |
| `page` | `1` | Nomor halaman. |
| `limit` | `10` | Jumlah data per halaman. |

## Smoke Test Matrix

| Case | Result |
| --- | --- |
| Admin create owner auto lead | PASS |
| Supervisor cannot see admin-owned owner | PASS |
| Admin assigns lead to supervisor | PASS |
| Supervisor sees own owner | PASS |
| Supervisor assigns lead to sales | PASS |
| Assigned sales sees owner | PASS |
| Other sales cannot see owner | PASS |
| Sales mark invalid returns to supervisor | PASS |
| Sales loses visibility after invalid | PASS |
| Supervisor sees invalid lead | PASS |
| Assignment history recorded | PASS |

## Contoh Request

### Admin Assign Lead ke Supervisor

```http
POST /api/v1/leads/10/assign-supervisor
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "supervisor_id": 2,
  "reason": "Distribusi dari Admin ke Supervisor"
}
```

Response `200 OK`:

```json
{
  "data": {
    "id": 10,
    "current_owner_role": "SUPERVISOR",
    "current_owner": {
      "id": 2,
      "name": "Supervisor Demo 001",
      "role": "SUPERVISOR"
    },
    "stage": "NEW",
    "status": "OPEN",
    "current_score": 1
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Supervisor Assign Lead ke Sales

```http
POST /api/v1/leads/10/assign-sales
Authorization: Bearer {supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "sales_id": 3,
  "reason": "Distribusi dari Supervisor ke Sales"
}
```

Response `200 OK` berisi lead dengan `current_owner_role = SALES`.

### Sales Mark Invalid

```http
POST /api/v1/leads/10/mark-invalid
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "reason": "Customer tidak potensial dari call terakhir"
}
```

Response `200 OK`:

```json
{
  "data": {
    "id": 10,
    "current_owner_role": "SUPERVISOR",
    "status": "INVALID",
    "stage": "INVALID",
    "current_score": 0
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

## Contoh Error

### Sales Lain Membuka Lead Bukan Miliknya

```http
GET /api/v1/leads/10
Authorization: Bearer {other_sales_access_token}
Accept: application/json
```

Response `404 Not Found`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Supervisor Assign Lead yang Bukan Scope-nya

```http
POST /api/v1/leads/10/assign-sales
Authorization: Bearer {supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "sales_id": 3
}
```

Response `403 Forbidden`:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

## Conclusion

Sprint 05 API ownership dan lead assignment berjalan sesuai briefing terbaru.

**Overall API Testing Status:** `PASS`
