# API Testing Report - Sprint 08 Closing dan Laporan Penjualan

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 08 - Closing dan Laporan Penjualan |
| Tanggal Testing Awal | 23 Juli 2026 |
| Tanggal Update Laporan | 24 Juli 2026 |
| Environment | Local Development |
| Standard API Base URL | `http://localhost:8080/api/v1` |
| Actual Smoke Test Base URL | `http://localhost:18080/api/v1` |
| Testing Tool | Manual smoke test via PowerShell/curl/Postman-compatible request |
| Auth | JWT Bearer Token |
| Seeder | `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` |

## 2. Header Pengujian

Route protected wajib memakai header berikut:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Route yang diuji memakai akun demo:

| Role | Email | Password | Tujuan Pengujian |
| --- | --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` | Confirm/reject, list all, mutation closing. |
| Supervisor | `supervisor.001@demo.piposmart.id` | `Password123!` | Confirm/reject dan visibility team. |
| Sales 1 | `sales.001@demo.piposmart.id` | `Password123!` | Create closing untuk lead sendiri. |
| Sales 2 | `sales.002@demo.piposmart.id` | `Password123!` | Negative test create closing lead milik Sales lain. |

## 3. Scope Pengujian

Sprint 08 memvalidasi bahwa remark score `3` tidak lagi dicatat melalui endpoint interaction biasa, tetapi melalui laporan closing yang menyimpan snapshot package/plan/promo dan perhitungan nominal dalam satu database transaction.

Hal yang divalidasi:

- Sales hanya dapat membuat closing untuk lead yang sedang menjadi miliknya.
- Closing membuat record `sales_closings`, interaction remark score `3`, dan stage history `CLOSING` secara atomic.
- Snapshot package, plan, dan promo tidak bergantung pada perubahan master data setelah transaksi.
- Status closing mendukung `PENDING_RECONCILIATION`, `CONFIRMED`, dan `REJECTED`.
- Pending closing belum dianggap KPI/revenue confirmed.
- Soft delete, restore, force delete, dan bulk mutation berjalan sesuai role.
- Error handler mengembalikan response standar.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Auth | `/auth/login` Sales | POST | 200 OK | PASS |
| Catalog | Lookup plan Business 12 bulan | GET | 200 OK | PASS |
| Catalog | Lookup promo FREE Business 12 bulan | GET | 200 OK | PASS |
| Closing | `/closings` | GET | 200 OK | PASS |
| Closing | `/closings?status=CONFIRMED` | GET | 200 OK | PASS |
| Closing | `/closings?q={keyword}` | GET | 200 OK | PASS |
| Closing | `/closings?closed_from=2026-07-01&closed_to=2026-07-31` | GET | 200 OK | PASS |
| Closing | `/leads/{lead_id}/closings` tanpa promo | POST | 201 Created | PASS |
| Closing | `/leads/{lead_id}/closings` promo FREE | POST | 201 Created | PASS |
| Closing | `/leads/{lead_id}/closings` promo PAID | POST | 201 Created | PASS |
| Closing | `/closings/{closing_id}` | GET | 200 OK | PASS |
| Closing | `/closings/{closing_id}/confirm` | POST | 200 OK | PASS |
| Closing | `/closings/{closing_id}/reject` | POST | 200 OK | PASS |
| Closing | `/closings/{closing_id}` soft delete | DELETE | 200 OK | PASS |
| Closing | `/closings/{closing_id}/restore` | PATCH | 200 OK | PASS |
| Closing | `/closings/{closing_id}/force` | DELETE | 200 OK | PASS |
| Closing | `/closings/bulk` | DELETE | 200 OK | PASS |
| Closing | `/closings/bulk/restore` | PATCH | 200 OK | PASS |
| Closing | `/closings/bulk/force` | DELETE | 200 OK | PASS |
| RBAC | Sales create closing untuk lead Sales lain | POST | 403 FORBIDDEN | PASS |
| Error | Duplicate pending/confirmed closing pada lead sama | POST | 409 LEAD_ALREADY_HAS_CLOSING | PASS |
| Error | Discount membuat final amount negatif | POST | 400 FINAL_AMOUNT_NEGATIVE | PASS |
| Error | Promo tidak eligible untuk plan | POST | 400 INVALID_PROMOTION | PASS |
| Error | Decimal invalid | POST | 400 INVALID_DECIMAL | PASS |
| Error | Unique transfer code di luar 0-999 | POST | 400 VALIDATION_ERROR | PASS |
| Error | Sort invalid | GET | 400 INVALID_SORT | PASS |
| Error | `closed_from` bukan format tanggal | GET | 400 VALIDATION_ERROR | PASS |
| Error | Confirm/reject closing bukan status pending | POST | 400 INVALID_STATUS | PASS |
| Error | Detail closing tidak ditemukan | GET | 404 NOT_FOUND | PASS |
| Activity Guard | Remark score 3 melalui interaction biasa | POST | 400 INVALID_TRANSITION | PASS |

## 5. Route Coverage Sprint 08

| Nama Route | Method | Path | Fungsi | Coverage |
| --- | --- | --- | --- | --- |
| List Closing | GET | `/api/v1/closings` | List closing sesuai visibility role. | Success + filter + invalid query. |
| Detail Closing | GET | `/api/v1/closings/{closing_id}` | Detail closing dan snapshot transaksi. | Success + not found. |
| Create Lead Closing | POST | `/api/v1/leads/{lead_id}/closings` | Sales membuat closing untuk lead miliknya. | Success + RBAC + validation + duplicate. |
| Confirm Closing | POST | `/api/v1/closings/{closing_id}/confirm` | Admin/Supervisor confirm closing pending. | Success + invalid status + forbidden Sales. |
| Reject Closing | POST | `/api/v1/closings/{closing_id}/reject` | Admin/Supervisor reject closing pending. | Success + invalid status + forbidden Sales. |
| Soft Delete Closing | DELETE | `/api/v1/closings/{closing_id}` | Soft delete closing. | Success + forbidden Sales. |
| Restore Closing | PATCH | `/api/v1/closings/{closing_id}/restore` | Restore closing soft-deleted. | Success. |
| Force Delete Closing | DELETE | `/api/v1/closings/{closing_id}/force` | Hard delete closing permanen. | Success pada data test. |
| Bulk Soft Delete | DELETE | `/api/v1/closings/bulk` | Bulk soft delete closing. | Success + invalid body. |
| Bulk Restore | PATCH | `/api/v1/closings/bulk/restore` | Bulk restore closing. | Success. |
| Bulk Force Delete | DELETE | `/api/v1/closings/bulk/force` | Bulk hard delete closing. | Success pada data test. |

## 6. Query Parameter List Closing

Endpoint:

```http
GET /api/v1/closings
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Laundry Cerah` | Search kode closing, kode lead, atau nama owner. |
| `status` | `PENDING_RECONCILIATION` | Filter status closing: `PENDING_RECONCILIATION`, `CONFIRMED`, `REJECTED`. |
| `lead_id` | `4` | Filter closing berdasarkan lead. |
| `owner_id` | `1` | Filter closing berdasarkan owner. |
| `sales_id` | `3` | Filter closing berdasarkan Sales. |
| `supervisor_id` | `2` | Filter closing berdasarkan Supervisor. |
| `plan_id` | `8` | Filter closing berdasarkan plan subscription. |
| `closed_from` | `2026-07-01` | Batas awal tanggal closing. |
| `closed_to` | `2026-07-31` | Batas akhir tanggal closing. |
| `sort` | `-closed_at` | Sort by `closed_at`, `created_at`, `updated_at`, `status`, `final_amount`, atau `code`. Prefix `-` berarti descending. |
| `page` | `1` | Halaman data. |
| `limit` | `10` | Jumlah data per halaman, maksimal 100. |

Contoh request:

```http
GET /api/v1/closings?status=CONFIRMED&closed_from=2026-07-01&closed_to=2026-07-31&sort=-final_amount&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "CLS-20260723100000-000004-000000",
        "status": "CONFIRMED",
        "final_amount": "1698723.00",
        "currency": "IDR",
        "plan_snapshot": {
          "code": "BUSINESS_12_MONTHS",
          "tenure_months": 12,
          "duration_days": 360
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

## 7. Success Case Detail

### Case 7.1 - Create Closing Promo FREE

Request:

```http
POST /api/v1/leads/{lead_id}/closings
Authorization: Bearer {sales_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 8,
  "promotion_id": 1,
  "discount_amount": "0.00",
  "unique_transfer_code": 123,
  "closed_at": "2026-07-23T10:00:00+07:00",
  "interaction_type": "CALL",
  "contact_name": "Owner Laundry",
  "contact_phone": "6281234567890",
  "customer_response": "Customer setuju ambil paket Business 12 bulan",
  "note": "Closing Business 12 + promo free 1 bulan"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "status": "PENDING_RECONCILIATION",
    "tenure_months": 12,
    "duration_days": 360,
    "base_price": "1698600.00",
    "discount_amount": "0.00",
    "additional_charge": "0.00",
    "unique_transfer_code": 123,
    "final_amount": "1698723.00",
    "currency": "IDR",
    "plan_snapshot": {
      "code": "BUSINESS_12_MONTHS"
    },
    "promotion_snapshot": {
      "code": "FREE_1_MONTH_BUSINESS_12",
      "charge_type": "FREE"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 7.2 - Create Closing Promo PAID

Request menggunakan plan Pro 12 bulan dan promo bundle alat berbayar.

```json
{
  "plan_id": 13,
  "promotion_id": 3,
  "discount_amount": "0.00",
  "unique_transfer_code": 321,
  "interaction_type": "CALL",
  "customer_response": "Customer ambil paket Pro 12 bulan dengan bundle alat",
  "note": "Closing Pro 12 + promo alat POS"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "status": "PENDING_RECONCILIATION",
    "additional_charge": "1500000.00",
    "promotion_snapshot": {
      "code": "PRO_12_ANDROID_POS_BUNDLE",
      "charge_type": "PAID"
    }
  }
}
```

Result: PASS.

### Case 7.3 - Detail Closing Snapshot

Request:

```http
GET /api/v1/closings/{closing_id}
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "package_snapshot": {
      "code": "BUSINESS"
    },
    "plan_snapshot": {
      "code": "BUSINESS_12_MONTHS",
      "duration_days": 360
    },
    "promotion_snapshot": {
      "code": "FREE_1_MONTH_BUSINESS_12"
    }
  }
}
```

Result: PASS.

### Case 7.4 - Confirm Closing

Request:

```http
POST /api/v1/closings/{closing_id}/confirm
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "note": "Pembayaran sudah diverifikasi Admin"
}
```

Expected response `200 OK`:

```json
{
  "data": {
    "status": "CONFIRMED",
    "confirmed_at": "2026-07-23T..."
  }
}
```

Result: PASS.

### Case 7.5 - Reject Closing

Request:

```http
POST /api/v1/closings/{closing_id}/reject
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "reason": "Customer batal melakukan pembayaran",
  "note": "Rejected saat smoke test"
}
```

Expected response `200 OK`:

```json
{
  "data": {
    "status": "REJECTED",
    "rejection_reason": "Customer batal melakukan pembayaran",
    "rejected_at": "2026-07-23T..."
  }
}
```

Result: PASS.

### Case 7.6 - Soft Delete, Restore, dan Force Delete

Soft delete:

```http
DELETE /api/v1/closings/{closing_id}
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `200 OK`:

```json
{
  "data": {
    "ids": [1],
    "affected": 1
  }
}
```

Restore:

```http
PATCH /api/v1/closings/{closing_id}/restore
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Force delete hanya digunakan pada data test yang memang boleh dihapus permanen:

```http
DELETE /api/v1/closings/{closing_id}/force
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Result: PASS.

### Case 7.7 - Bulk Closing Mutation

Request bulk soft delete:

```http
DELETE /api/v1/closings/bulk
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "ids": [10, 11]
}
```

Expected response `200 OK`:

```json
{
  "data": {
    "ids": [10, 11],
    "affected": 2
  }
}
```

Result: PASS.

## 8. Error Handling Cases

### Case 8.1 - Request Tanpa Token

Request:

```http
GET /api/v1/closings
Accept: application/json
```

Expected response `401 Unauthorized`:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.2 - Sales Membuat Closing Lead Milik Sales Lain

Request:

```http
POST /api/v1/leads/{lead_id_milik_sales_lain}/closings
Authorization: Bearer {sales_access_token}
Content-Type: application/json
```

```json
{
  "plan_id": 8,
  "discount_amount": "0.00",
  "note": "Negative test forbidden"
}
```

Expected response `403 Forbidden`:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.3 - Lead Sudah Punya Closing Pending atau Confirmed

Request kedua pada lead yang sama:

```json
{
  "plan_id": 8,
  "discount_amount": "0.00",
  "note": "Duplicate closing"
}
```

Expected response `409 Conflict`:

```json
{
  "error": {
    "code": "LEAD_ALREADY_HAS_CLOSING",
    "message": "lead sudah memiliki closing pending atau confirmed",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.4 - Discount Membuat Final Amount Negatif

Request:

```json
{
  "plan_id": 8,
  "discount_amount": "999999999.00",
  "unique_transfer_code": 0
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "FINAL_AMOUNT_NEGATIVE",
    "message": "final amount tidak boleh negatif",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.5 - Promo Tidak Eligible Untuk Plan

Request:

```json
{
  "plan_id": 8,
  "promotion_id": 3,
  "discount_amount": "0.00",
  "note": "Promo Pro dipakai untuk Business"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_PROMOTION",
    "message": "promo tidak eligible untuk plan yang dipilih",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.6 - Decimal Tidak Valid

Request:

```json
{
  "plan_id": 8,
  "discount_amount": "10.123"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_DECIMAL",
    "message": "nilai uang harus decimal valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.7 - Unique Transfer Code Tidak Valid

Request:

```json
{
  "plan_id": 8,
  "unique_transfer_code": 1000
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request closing tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.8 - Invalid Sort

Request:

```http
GET /api/v1/closings?sort=-unknown_field
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "sort tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.9 - Invalid Date Filter

Request:

```http
GET /api/v1/closings?closed_from=23-07-2026
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "closed_from harus format YYYY-MM-DD",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.10 - Confirm Closing Yang Sudah Confirmed

Request:

```http
POST /api/v1/closings/{closing_id_confirmed}/confirm
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "note": "Confirm ulang"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_STATUS",
    "message": "status closing tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.11 - Closing Tidak Ditemukan

Request:

```http
GET /api/v1/closings/999999999
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `404 Not Found`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data closing tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.12 - Bulk Body Invalid

Request:

```http
DELETE /api/v1/closings/bulk
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "ids": []
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.13 - Remark 3 Melalui Interaction Biasa Ditolak

Request:

```http
POST /api/v1/leads/{lead_id}/interactions
Authorization: Bearer {sales_access_token}
Content-Type: application/json
```

```json
{
  "interaction_type": "CALL",
  "remark_score": 3,
  "note": "Mencoba closing tanpa laporan penjualan"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_TRANSITION",
    "message": "remark 3 harus dicatat melalui closing",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

## 9. RBAC Matrix Closing

| Action | Admin | Supervisor | Sales |
| --- | --- | --- | --- |
| List closing | Semua data | Data dalam cakupan supervisor/team | Data milik sendiri |
| Detail closing | Semua data | Data dalam cakupan supervisor/team | Data milik sendiri |
| Create closing | Tidak digunakan untuk closing Sales | Tidak digunakan untuk closing Sales | Hanya lead miliknya |
| Confirm closing | Bisa | Bisa | Tidak bisa |
| Reject closing | Bisa | Bisa | Tidak bisa |
| Soft delete/restore/force delete | Bisa | Bisa sesuai visibility | Tidak bisa |
| Bulk mutation | Bisa | Bisa sesuai visibility | Tidak bisa |

## 10. Quality Gate

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build .` | PASS |
| `go run . migrate up` | PASS |
| `go run . migrate down` lalu `go run . migrate up` | PASS |
| `go run . seed master` | PASS |
| `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` | PASS |
| Smoke test route Sprint 08 | PASS |

## 11. Defect dan Catatan

| Item | Status | Catatan |
| --- | --- | --- |
| Route coverage closing | CLOSED | Semua route closing Sprint 08 terdokumentasi. |
| Error handler coverage | CLOSED | Error utama ditambahkan dengan contoh request dan response. |
| Reconciliation penuh | DEFERRED | Masuk scope Sprint 10: order, subscription, wallet debit, reconciliation final. |

## 12. Conclusion

Sprint 08 menyediakan API closing yang atomic dengan remark score `3`, snapshot catalog, dan perhitungan nominal yang aman untuk uang. Berdasarkan smoke test lokal, migration, seeder, unit/integration test, vet, dan build, seluruh route Sprint 08 yang masuk scope dinyatakan PASS.
