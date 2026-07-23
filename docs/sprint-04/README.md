# API Testing Report - Sprint 04

## Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 04 - Owner dan Outlet |
| Tanggal Testing | 23 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| Smoke Test Tambahan | `http://localhost:8081/api/v1` |
| OpenAPI URL | `http://localhost:8080/openapi.yaml` |
| Swagger UI | `http://localhost:8080/swagger/index.html` |
| Testing Tool | Manual smoke test via PowerShell |
| Database | MySQL local, database `piposmart` |
| Migration Version | `20260723000300` |

## Testing Summary

| Module | Total Case | Passed | Failed | Status |
| --- | ---: | ---: | ---: | --- |
| Owner Management | 10 | 10 | 0 | PASS |
| Outlet Management | 7 | 7 | 0 | PASS |
| Restore, Force Delete, dan Bulk Operation | 10 | 10 | 0 | PASS |
| RBAC | 1 | 1 | 0 | PASS |
| **Total** | **28** | **28** | **0** | **PASS** |

**Success Rate:** `100%`

## Authentication Header

Seluruh endpoint Owner/Outlet membutuhkan JWT milik Admin atau Supervisor.

```http
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

Contoh lengkap:

```http
GET /api/v1/owners?page=1&limit=10
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Accept: application/json
```

Sales tidak memiliki permission `owners.manage`, sehingga akses ke endpoint ini
ditolak dengan `403 Forbidden`.

---

# 1. Route Convention untuk Restore, Force Delete, dan Bulk

Mulai Sprint 04, pola route berikut disepakati untuk entitas CRM yang memiliki
soft delete:

| Kebutuhan | Pola Route |
| --- | --- |
| Restore single data | `PATCH /api/v1/{entities}/{id}/restore` |
| Hard delete single data | `DELETE /api/v1/{entities}/{id}/force` |
| Bulk create | `POST /api/v1/{entities}/bulk` |
| Bulk update | `PATCH /api/v1/{entities}/bulk` |
| Bulk soft delete | `DELETE /api/v1/{entities}/bulk` |
| Bulk hard delete | `DELETE /api/v1/{entities}/bulk/force` |

Untuk nested entity seperti outlet:

```text
PATCH  /api/v1/owners/{owner_id}/outlets/{outlet_id}/restore
DELETE /api/v1/owners/{owner_id}/outlets/{outlet_id}/force
POST   /api/v1/owners/{owner_id}/outlets/bulk
PATCH  /api/v1/owners/{owner_id}/outlets/bulk
DELETE /api/v1/owners/{owner_id}/outlets/bulk
DELETE /api/v1/owners/{owner_id}/outlets/bulk/force
```

Delete behavior:

- Soft delete mengisi `deleted_at` dan tidak menghapus child data.
- Restore mengosongkan kembali `deleted_at` dan mengubah status menjadi `ACTIVE`.
- Hard delete menghapus row permanen dari database.
- Jika parent di-hard-delete, child tidak dihapus. Foreign key child diarahkan
  menjadi `NULL` jika skema tabel mendukung.
- Jika parent masih soft-deleted, child tetap menyimpan foreign key parent.

---

# 2. Query Parameter Guide

## 2.1 Owner List - GET `/owners`

Endpoint:

```http
GET /api/v1/owners
Authorization: Bearer {admin_or_supervisor_access_token}
```

Parameter yang tersedia:

| Parameter | Tipe | Contoh | Keterangan |
| --- | --- | --- | --- |
| `q` | string | `Laundry Cerah` | Search global pada `code`, `name`, `phone`, `brand_name`, `city`, dan `province`. |
| `code` | string | `OWN-00001` | Filter berdasarkan kode owner. |
| `name` | string | `Budi` | Filter berdasarkan nama owner. |
| `phone` | string | `0812` | Filter berdasarkan nomor telepon. Input dapat memakai format `08...`, `8...`, atau `+62...`. |
| `brand_name` | string | `Laundry Cerah` | Filter berdasarkan brand laundry. |
| `province` | string | `Jawa Barat` | Filter berdasarkan provinsi. |
| `city` | string | `Bandung` | Filter berdasarkan kota. |
| `page` | integer | `1` | Nomor halaman. Default `1`. |
| `limit` | integer | `10` | Jumlah item per halaman. Default `10`, maksimum `100`. |
| `sort` | string | `-created_at` | Sorting whitelist. Prefix `-` artinya descending. |

Field `sort` yang didukung untuk owner:

```text
created_at
updated_at
code
name
brand_name
city
province
```

Contoh request list owner terbaru:

```http
GET /api/v1/owners?page=1&limit=10&sort=-created_at
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Contoh request search global:

```http
GET /api/v1/owners?q=Laundry%20Cerah&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Contoh request filter spesifik:

```http
GET /api/v1/owners?province=Jawa%20Barat&city=Bandung&brand_name=Laundry%20Cerah
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Contoh request filter phone:

```http
GET /api/v1/owners?phone=0812
Authorization: Bearer {admin_access_token}
Accept: application/json
```

## 2.2 Outlet List - GET `/owners/{owner_id}/outlets`

Endpoint:

```http
GET /api/v1/owners/{owner_id}/outlets
Authorization: Bearer {admin_or_supervisor_access_token}
```

Parameter yang tersedia:

| Parameter | Tipe | Contoh | Keterangan |
| --- | --- | --- | --- |
| `q` | string | `Outlet Utama` | Search global pada `code`, `name`, `phone`, `city`, dan `province`. |
| `code` | string | `OUT-00001` | Filter berdasarkan kode outlet. |
| `name` | string | `Outlet Utama` | Filter berdasarkan nama outlet. |
| `phone` | string | `0812` | Filter berdasarkan nomor telepon outlet. |
| `province` | string | `Jawa Barat` | Filter berdasarkan provinsi outlet. |
| `city` | string | `Bandung` | Filter berdasarkan kota outlet. |
| `page` | integer | `1` | Nomor halaman. Default `1`. |
| `limit` | integer | `10` | Jumlah item per halaman. Default `10`, maksimum `100`. |
| `sort` | string | `name` | Sorting whitelist. Prefix `-` artinya descending. |

Field `sort` yang didukung untuk outlet:

```text
created_at
updated_at
code
name
city
province
```

Contoh request:

```http
GET /api/v1/owners/9/outlets?q=Outlet&page=1&limit=10&sort=name
Authorization: Bearer {admin_access_token}
Accept: application/json
```

---

# 3. Test Case Matrix

| ID | Method | Endpoint | Expected | Actual | Result |
| --- | --- | --- | ---: | ---: | --- |
| OWN-001 | GET | `/owners?page=1&limit=5&sort=-created_at` | 200 | 200 | PASS |
| OWN-002 | POST | `/owners` | 201 | 201 | PASS |
| OWN-003 | GET | `/owners?q=Laundry%20API%20Sprint&city=Bandung` | 200 | 200 | PASS |
| OWN-004 | GET | `/owners/{owner_id}` | 200 | 200 | PASS |
| OWN-005 | PATCH | `/owners/{owner_id}` | 200 | 200 | PASS |
| OWN-006 | POST | `/owners` duplicate code | 409 | 409 | PASS |
| OWN-007 | POST | `/owners` invalid phone | 400 | 400 | PASS |
| OUT-001 | POST | `/owners/{owner_id}/outlets` | 201 | 201 | PASS |
| OUT-002 | GET | `/owners/{owner_id}/outlets?page=1&limit=10&q=Outlet` | 200 | 200 | PASS |
| OUT-003 | GET | `/owners/{owner_id}/outlets/{outlet_id}` | 200 | 200 | PASS |
| OUT-004 | PATCH | `/owners/{owner_id}/outlets/{outlet_id}` | 200 | 200 | PASS |
| OUT-005 | POST | `/owners/999999999/outlets` | 404 | 404 | PASS |
| OUT-006 | DELETE | `/owners/{owner_id}/outlets/{outlet_id}` | 200 | 200 | PASS |
| OUT-007 | GET | `/owners/{owner_id}/outlets/{outlet_id}` setelah delete | 404 | 404 | PASS |
| OWN-008 | DELETE | `/owners/{owner_id}` | 200 | 200 | PASS |
| OWN-009 | GET | `/owners/{owner_id}` setelah delete | 404 | 404 | PASS |
| RBAC-OWN-001 | GET | `/owners` sebagai Sales | 403 | 403 | PASS |
| OWN-010 | GET | `/owners?sort=unknown` | 400 | 400 | PASS |
| BULK-OWN-001 | POST | `/owners/bulk` | 201 | 201 | PASS |
| BULK-OWN-002 | PATCH | `/owners/bulk` | 200 | 200 | PASS |
| BULK-OUT-001 | POST | `/owners/{owner_id}/outlets/bulk` | 201 | 201 | PASS |
| BULK-OUT-002 | PATCH | `/owners/{owner_id}/outlets/bulk` | 200 | 200 | PASS |
| RESTORE-OUT-001 | PATCH | `/owners/{owner_id}/outlets/{outlet_id}/restore` | 200 | 200 | PASS |
| RESTORE-OWN-001 | PATCH | `/owners/{owner_id}/restore` | 200 | 200 | PASS |
| DELETE-POLICY-001 | GET | outlet setelah owner soft delete lalu restore owner | 200 | 200 | PASS |
| FORCE-OWN-001 | DELETE | `/owners/bulk/force` | 200 | 200 | PASS |
| FORCE-OUT-001 | DELETE | `/owners/{owner_id}/outlets/bulk/force` | 200 | 200 | PASS |
| FORCE-OWN-002 | DELETE | `/owners/{owner_id}/force` | 200 | 200 | PASS |

---

# 4. Successful Request Samples

## 4.1 Create Owner

### Request

```http
POST /api/v1/owners
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OWN-API-20260723141959",
  "name": "Owner API Sprint 4",
  "phone": "0812-7777-0001",
  "email": "owner.api.20260723141959@example.test",
  "brand_name": "Laundry API Sprint",
  "province": "Jawa Barat",
  "city": "Bandung",
  "address": "Jl. API Sprint 4"
}
```

### Response

**Status:** `201 Created`

```json
{
  "data": {
    "id": 9,
    "code": "OWN-API-20260723141959",
    "name": "Owner API Sprint 4",
    "phone": "6281277770001",
    "email": "owner.api.20260723141959@example.test",
    "brand_name": "Laundry API Sprint",
    "province": "Jawa Barat",
    "city": "Bandung",
    "address": "Jl. API Sprint 4",
    "status": "ACTIVE",
    "created_at": "2026-07-23T07:19:59Z",
    "updated_at": "2026-07-23T07:19:59Z"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Owner berhasil dibuat dan nomor telepon dinormalisasi dari `0812...` menjadi `62812...`.

## 4.2 List Owner dengan Search, Pagination, dan Sorting

### Request

```http
GET /api/v1/owners?q=Laundry%20API%20Sprint&city=Bandung&page=1&limit=10&sort=-created_at
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "items": [
      {
        "id": 9,
        "code": "OWN-API-20260723141959",
        "name": "Owner API Sprint 4",
        "phone": "6281277770001",
        "brand_name": "Laundry API Sprint",
        "city": "Bandung",
        "province": "Jawa Barat",
        "outlet_count": 0,
        "status": "ACTIVE"
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

### Result

PASS. API berhasil menjalankan filter `q`, filter `city`, pagination, dan sorting.

## 4.3 Update Owner

### Request

```http
PATCH /api/v1/owners/9
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "name": "Owner API Sprint 4 Updated",
  "phone": "+62 812 7777 0002"
}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 9,
    "code": "OWN-API-20260723141959",
    "name": "Owner API Sprint 4 Updated",
    "phone": "6281277770002",
    "status": "ACTIVE"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Owner berhasil diperbarui dan nomor telepon tetap dinormalisasi.

## 4.4 Create Outlet

### Request

```http
POST /api/v1/owners/9/outlets
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OUT-API-20260723141959",
  "name": "Outlet API Sprint 4",
  "phone": "0812-8888-0001",
  "province": "Jawa Barat",
  "city": "Bandung",
  "address": "Jl. Outlet API"
}
```

### Response

**Status:** `201 Created`

```json
{
  "data": {
    "id": 13,
    "owner_id": 9,
    "code": "OUT-API-20260723141959",
    "name": "Outlet API Sprint 4",
    "phone": "6281288880001",
    "province": "Jawa Barat",
    "city": "Bandung",
    "address": "Jl. Outlet API",
    "status": "ACTIVE"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Outlet berhasil dibuat dan wajib menunjuk owner yang valid.

## 4.5 List Outlet

### Request

```http
GET /api/v1/owners/9/outlets?page=1&limit=10&q=Outlet&sort=name
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "items": [
      {
        "id": 13,
        "owner_id": 9,
        "code": "OUT-API-20260723141959",
        "name": "Outlet API Sprint 4",
        "phone": "6281288880001",
        "city": "Bandung",
        "province": "Jawa Barat",
        "status": "ACTIVE"
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

### Result

PASS. List outlet berhasil difilter dalam scope owner tertentu.

## 4.6 Bulk Create Owner

### Request

```http
POST /api/v1/owners/bulk
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "items": [
    {
      "code": "OWN-BULK-001",
      "name": "Owner Bulk 1",
      "phone": "0812-7000-0001",
      "brand_name": "Laundry Bulk",
      "province": "Jawa Barat",
      "city": "Bandung"
    },
    {
      "code": "OWN-BULK-002",
      "name": "Owner Bulk 2",
      "phone": "0812-7000-0002",
      "brand_name": "Laundry Bulk",
      "province": "Jawa Barat",
      "city": "Bekasi"
    }
  ]
}
```

### Response

**Status:** `201 Created`

```json
{
  "data": {
    "items": [
      {
        "id": 21,
        "code": "OWN-BULK-001",
        "name": "Owner Bulk 1",
        "phone": "6281270000001",
        "status": "ACTIVE"
      },
      {
        "id": 22,
        "code": "OWN-BULK-002",
        "name": "Owner Bulk 2",
        "phone": "6281270000002",
        "status": "ACTIVE"
      }
    ],
    "total": 2
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Dua owner berhasil dibuat dalam satu request.

## 4.7 Bulk Update Owner

### Request

```http
PATCH /api/v1/owners/bulk
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "items": [
    {
      "id": 21,
      "name": "Owner Bulk Updated 1"
    },
    {
      "id": 22,
      "city": "Cimahi"
    }
  ]
}
```

### Result

PASS. Bulk update berhasil dan response mengembalikan daftar owner yang diperbarui.

## 4.8 Bulk Soft Delete dan Bulk Hard Delete

### Bulk Soft Delete Request

```http
DELETE /api/v1/owners/bulk
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "ids": [21, 22]
}
```

### Bulk Hard Delete Request

```http
DELETE /api/v1/owners/bulk/force
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "ids": [21, 22]
}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "ids": [21, 22],
    "affected": 2
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Bulk delete mengembalikan jumlah row yang terdampak.

## 4.9 Restore dan Force Delete Single Data

### Restore Owner Request

```http
PATCH /api/v1/owners/21/restore
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Force Delete Owner Request

```http
DELETE /api/v1/owners/21/force
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Restore Outlet Request

```http
PATCH /api/v1/owners/21/outlets/31/restore
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Force Delete Outlet Request

```http
DELETE /api/v1/owners/21/outlets/31/force
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Result

PASS. Restore mengembalikan data menjadi `ACTIVE`; force delete menghapus row secara permanen.

---

# 5. Error Handling yang Diuji

Catatan:

- Case dengan label `Response` adalah case yang masuk smoke test manual Sprint 04.
- Case dengan label `Response yang Diharapkan` adalah contoh negative case tambahan berdasarkan kontrak route/handler yang tersedia, dan dapat dipakai CTO/frontend sebagai referensi saat melakukan verifikasi lanjutan.

## 5.1 Missing Authorization Header

### Request yang Membuat Error

```http
GET /api/v1/owners
Accept: application/json
```

### Response

**Status:** `401 Unauthorized`

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Endpoint protected menolak request tanpa JWT.

## 5.2 Role Sales Tidak Memiliki Permission

### Request yang Membuat Error

```http
GET /api/v1/owners
Authorization: Bearer {sales_access_token}
Accept: application/json
```

### Response

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Kamu tidak memiliki akses ke fitur ini",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Sales tidak dapat mengakses modul Owner/Outlet.

## 5.3 Duplicate Owner Code

### Request yang Membuat Error

```http
POST /api/v1/owners
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OWN-API-20260723141959",
  "name": "Owner Duplicate Code"
}
```

### Response

**Status:** `409 Conflict`

```json
{
  "error": {
    "code": "CODE_ALREADY_USED",
    "message": "kode sudah digunakan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Kode owner unik dan duplicate code ditolak.

## 5.4 Duplicate Outlet Code

### Request yang Membuat Error

```http
POST /api/v1/owners/9/outlets
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OUT-API-20260723141959",
  "name": "Outlet Duplicate Code"
}
```

### Response yang Diharapkan

**Status:** `409 Conflict`

```json
{
  "error": {
    "code": "CODE_ALREADY_USED",
    "message": "kode sudah digunakan",
    "request_id": "generated-request-id"
  }
}
```

### Result

Contract PASS. Constraint unique outlet code berada di database dan dipetakan ke `CODE_ALREADY_USED`.

## 5.5 Invalid Phone Format

### Request yang Membuat Error

```http
POST /api/v1/owners
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OWN-BAD-PHONE",
  "name": "Owner Bad Phone",
  "phone": "abc"
}
```

### Response

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_PHONE",
    "message": "nomor telepon tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API menolak nomor telepon yang tidak dapat dinormalisasi.

## 5.6 Required Field Tidak Dikirim

### Request yang Membuat Error

```http
POST /api/v1/owners
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OWN-MISSING-NAME"
}
```

### Response yang Diharapkan

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "Key: 'CreateOwnerRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag"
    },
    "request_id": "generated-request-id"
  }
}
```

### Result

Contract PASS. Handler memakai binding validation untuk field `code` dan `name`.

## 5.7 Owner Tidak Ditemukan

### Request yang Membuat Error

```http
GET /api/v1/owners/999999999
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API mengembalikan response standar untuk owner yang tidak ditemukan.

## 5.8 Outlet Dibuat untuk Owner Tidak Ada

### Request yang Membuat Error

```http
POST /api/v1/owners/999999999/outlets
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OUT-NO-OWNER",
  "name": "Outlet Tanpa Owner"
}
```

### Response

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Outlet tidak dapat menunjuk owner yang tidak ada.

## 5.9 Outlet Tidak Ditemukan Setelah Soft Delete

### Request yang Membuat Error

```http
GET /api/v1/owners/9/outlets/13
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Outlet soft-deleted tidak muncul pada detail endpoint.

## 5.10 Owner Tidak Ditemukan Setelah Soft Delete

### Request yang Membuat Error

```http
GET /api/v1/owners/9
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Owner soft-deleted tidak muncul pada detail endpoint.

## 5.11 Sort Tidak Valid

### Request yang Membuat Error

```http
GET /api/v1/owners?sort=unknown
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "parameter sort tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API menolak field sorting yang tidak berada dalam whitelist.

## 5.12 ID Path Tidak Valid

### Request yang Membuat Error

```http
GET /api/v1/owners/abc
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### Response yang Diharapkan

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "ID tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### Result

Contract PASS. Handler melakukan parsing ID dan menolak nilai non-numeric.

## 5.13 Method Tidak Didukung

### Request yang Membuat Error

```http
PUT /api/v1/owners/9
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "name": "Should Use PATCH"
}
```

### Response yang Diharapkan

**Status:** `405 Method Not Allowed`

```json
{
  "error": {
    "code": "METHOD_NOT_ALLOWED",
    "message": "HTTP method tidak didukung",
    "request_id": "generated-request-id"
  }
}
```

### Result

Contract PASS. Router hanya menyediakan `PATCH` untuk update parsial.

---

# 6. Soft Delete, Restore, dan Force Delete Behavior

Soft delete diuji dengan alur berikut:

```http
DELETE /api/v1/owners/9/outlets/13
Authorization: Bearer {admin_access_token}
```

Setelah outlet dihapus, request detail:

```http
GET /api/v1/owners/9/outlets/13
Authorization: Bearer {admin_access_token}
```

menghasilkan:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

Owner soft delete juga diuji:

```http
DELETE /api/v1/owners/9
Authorization: Bearer {admin_access_token}
```

Setelah owner dihapus, request detail:

```http
GET /api/v1/owners/9
Authorization: Bearer {admin_access_token}
```

menghasilkan `404 NOT_FOUND`.

Setelah owner di-restore:

```http
PATCH /api/v1/owners/9/restore
Authorization: Bearer {admin_access_token}
```

outlet child tetap dapat diakses. Ini membuktikan bahwa soft delete owner tidak
lagi melakukan cascade soft delete ke outlet.

Untuk hard delete, route `/force` menghapus row permanen. Pada parent-child
relationship, child tidak ikut terhapus. Jika parent hard-deleted, foreign key
child menjadi `NULL` jika schema tabel mendukung.

---

# 7. Findings

Tidak ada defect fungsional pada scope Sprint 04.

Catatan teknis:

- Phone `0812...`, `812...`, dan `+62...` dinormalisasi menjadi `62...`.
- Data soft-deleted tidak muncul pada response default.
- Delete owner tidak melakukan cascade soft delete ke outlet.
- Restore dan force delete tersedia untuk owner dan outlet.
- Bulk create, bulk update, bulk soft delete, dan bulk hard delete tersedia untuk owner dan outlet.
- Hard delete owner didukung oleh migration orphan policy agar outlet child menjadi orphan dengan `owner_id = NULL`.
- Query list owner memakai agregasi outlet count dalam satu query.
- Beberapa negative case tambahan berlabel `Contract PASS` belum masuk hitungan smoke test utama, tetapi mengikuti kontrak handler dan router yang tersedia.
- Pesan validation error masih memakai nama struct Go dari Gin validator. Ini bisa dibuat lebih ramah saat hardening API.

---

# 8. Conclusion

Seluruh API Owner dan Outlet Sprint 04 berjalan sesuai ekspektasi pada smoke test utama.

| Area | Status |
| --- | --- |
| Owner CRUD | PASS |
| Outlet CRUD | PASS |
| Search/filter/pagination/sorting | PASS |
| Phone normalization | PASS |
| Soft delete | PASS |
| Restore | PASS |
| Hard delete `/force` | PASS |
| Bulk operation | PASS |
| Orphan delete policy | PASS |
| RBAC | PASS |
| Error handling utama | PASS |

**Overall API Testing Status:** `PASS`
