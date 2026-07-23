# API Testing Report - Sprint 03

## Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 03 - Authentication, RBAC, dan Sales Management |
| Tanggal Testing | 23 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| OpenAPI URL | `http://localhost:8080/openapi.yaml` |
| Swagger UI | `http://localhost:8080/swagger/index.html` |
| Testing Tool | Manual smoke test via PowerShell `Invoke-RestMethod` / `Invoke-WebRequest` |
| Database | MySQL local, database `piposmart` |
| Migration Version | `20260723000200` |
| Seeder | `seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` |

## Testing Summary

| Module | Total Case | Passed | Failed | Status |
| --- | ---: | ---: | ---: | --- |
| Core & Documentation | 3 | 3 | 0 | PASS |
| Authentication | 6 | 6 | 0 | PASS |
| Sales Management | 10 | 10 | 0 | PASS |
| RBAC | 4 | 4 | 0 | PASS |
| Error Handling | 1 | 1 | 0 | PASS |
| **Total** | **24** | **24** | **0** | **PASS** |

**Success Rate:** `100%`

## JWT Usage Guide

Endpoint yang membutuhkan autentikasi wajib mengirim header berikut:

```http
Authorization: Bearer {access_token}
Content-Type: application/json
```

Access token didapat dari endpoint:

```http
POST /api/v1/auth/login
```

Contoh flow frontend:

1. Frontend melakukan login memakai email dan password.
2. Backend mengembalikan `access_token`, `refresh_token`, `token_type`, `expires_in`, dan data user.
3. Frontend menyimpan access token sesuai strategi auth frontend.
4. Setiap request protected mengirim header `Authorization: Bearer {access_token}`.
5. Saat access token expired, frontend memakai `refresh_token` ke endpoint `/auth/refresh`.
6. Refresh token bersifat rotating. Refresh token lama tidak valid lagi setelah dipakai.

Contoh header:

```http
GET /api/v1/auth/me
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Test Data

| Role | Email | Password | Catatan |
| --- | --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` | Dari demo seeder |
| Supervisor | `supervisor.001@demo.piposmart.id` | `Password123!` | Dari demo seeder |
| Sales | `sales.001@demo.piposmart.id` | `Password123!` | Dari demo seeder |
| Sales Created by Admin | `sales.api.20260723133902@demo.piposmart.id` | Generated temporary password | Dibuat saat API testing |
| Sales Created by Supervisor | `sales.supervisor.api.20260723133902@demo.piposmart.id` | Generated temporary password | Dibuat saat API testing |

## Test Case Matrix

| ID | Module | Method | Endpoint | Expected | Actual | Result |
| --- | --- | --- | --- | ---: | ---: | --- |
| CORE-001 | Core | GET | `/health/live` | 200 | 200 | PASS |
| CORE-002 | Core | GET | `/health/ready` | 200 | 200 | PASS |
| CORE-003 | Core | GET | `/openapi.yaml` | 200 | 200 | PASS |
| AUTH-001 | Authentication | POST | `/auth/login` | 200 | 200 | PASS |
| AUTH-002 | Authentication | POST | `/auth/login` dengan password salah | 401 | 401 | PASS |
| AUTH-003 | Authentication | GET | `/auth/me` tanpa token | 401 | 401 | PASS |
| AUTH-004 | Authentication | GET | `/auth/me` dengan token Admin | 200 | 200 | PASS |
| AUTH-005 | Authentication | POST | `/auth/refresh` dengan refresh token valid | 200 | 200 | PASS |
| AUTH-006 | Authentication | POST | `/auth/refresh` memakai refresh token lama | 401 | 401 | PASS |
| SALES-001 | Sales Management | GET | `/sales` sebagai Admin | 200 | 200 | PASS |
| SALES-002 | Sales Management | POST | `/sales` sebagai Admin | 201 | 201 | PASS |
| SALES-003 | Sales Management | POST | `/sales` duplicate email | 409 | 409 | PASS |
| SALES-004 | Sales Management | POST | `/sales` payload invalid | 400 | 400 | PASS |
| SALES-005 | Sales Management | GET | `/sales/{id}` | 200 | 200 | PASS |
| SALES-006 | Sales Management | PATCH | `/sales/{id}` | 200 | 200 | PASS |
| SALES-007 | Sales Management | POST | `/sales/{id}/deactivate` | 200 | 200 | PASS |
| SALES-008 | Sales Management | POST | `/sales/{id}/activate` | 200 | 200 | PASS |
| SALES-009 | Sales Management | POST | `/sales/{id}/reset-password` | 200 | 200 | PASS |
| SALES-010 | Sales Management | GET | `/sales/999999999` | 404 | 404 | PASS |
| RBAC-001 | RBAC | POST | `/auth/login` sebagai Supervisor | 200 | 200 | PASS |
| RBAC-002 | RBAC | POST | `/sales` sebagai Supervisor | 201 | 201 | PASS |
| RBAC-003 | RBAC | POST | `/auth/login` sebagai Sales | 200 | 200 | PASS |
| RBAC-004 | RBAC | GET | `/sales` sebagai Sales | 403 | 403 | PASS |
| ERR-001 | Error Handling | GET | `/route-not-found` | 404 | 404 | PASS |

---

# 1. Core & Documentation

## CORE-001 - GET `/health/live`

### Request

```http
GET /health/live
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "service": "crm-piposmart-backend",
    "status": "alive"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API process hidup dan dapat menerima request HTTP.

## CORE-002 - GET `/health/ready`

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "mysql": "available",
    "status": "ready"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API berhasil melakukan ping ke MySQL.

## CORE-003 - GET `/openapi.yaml`

### Response

**Status:** `200 OK`

OpenAPI YAML berhasil diakses dan digunakan oleh Swagger UI.

### Result

PASS. Dokumentasi API aktif tersedia melalui HTTP.

---

# 2. Authentication

## AUTH-001 - POST `/auth/login`

### Request

```json
{
  "email": "admin.001@demo.piposmart.id",
  "password": "Password123!"
}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "masked-refresh-token",
    "token_type": "Bearer",
    "expires_in": 899,
    "refresh_token_expires_at": "2026-07-30T06:39:03Z",
    "user": {
      "id": 1,
      "code": "ADM-001",
      "name": "Admin Demo 001",
      "email": "admin.001@demo.piposmart.id",
      "role": "ADMIN",
      "status": "ACTIVE",
      "must_change_password": true,
      "permissions": [
        "catalog.manage",
        "leads.assign",
        "leads.work",
        "owners.manage",
        "reports.read_all",
        "users.manage_all",
        "users.manage_sales",
        "users.read"
      ]
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Login valid menghasilkan access token, refresh token, role, dan permission user.

## AUTH-002 - POST `/auth/login` Password Salah

### Request

```json
{
  "email": "admin.001@demo.piposmart.id",
  "password": "wrong-password"
}
```

### Response

**Status:** `401 Unauthorized`

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "email atau password tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API tidak membocorkan apakah email atau password yang salah.

## AUTH-003 - GET `/auth/me` Tanpa Token

### Request

```http
GET /api/v1/auth/me
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

PASS. Endpoint protected menolak request tanpa Bearer token.

## AUTH-004 - GET `/auth/me` Dengan Token Admin

### Request

```http
GET /api/v1/auth/me
Authorization: Bearer {admin_access_token}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 1,
    "code": "ADM-001",
    "name": "Admin Demo 001",
    "email": "admin.001@demo.piposmart.id",
    "role": "ADMIN",
    "status": "ACTIVE",
    "permissions": [
      "users.manage_all",
      "users.manage_sales",
      "users.read"
    ]
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Access token valid dapat membaca profil user login.

## AUTH-005 - POST `/auth/refresh`

### Request

```json
{
  "refresh_token": "{valid_refresh_token}"
}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "new-masked-refresh-token",
    "token_type": "Bearer",
    "expires_in": 899,
    "user": {
      "email": "admin.001@demo.piposmart.id",
      "role": "ADMIN"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Refresh token valid menghasilkan token baru.

## AUTH-006 - POST `/auth/refresh` Memakai Refresh Token Lama

### Response

**Status:** `401 Unauthorized`

```json
{
  "error": {
    "code": "INVALID_TOKEN",
    "message": "token tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Refresh token lama ditolak setelah rotation. Ini mencegah reuse refresh token.

---

# 3. Sales Management

## SALES-001 - GET `/sales`

### Request

```http
GET /api/v1/sales
Authorization: Bearer {admin_access_token}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "items": [
      {
        "id": 11,
        "name": "Sales API Testing",
        "email": "sales.api.20260723133902@demo.piposmart.id",
        "role": "SALES",
        "status": "ACTIVE"
      }
    ],
    "total": 5
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Admin dapat melihat daftar Sales.

## SALES-002 - POST `/sales`

### Request

```http
POST /api/v1/sales
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "name": "Sales API Testing",
  "email": "sales.api.20260723133902@demo.piposmart.id",
  "phone": "6281299900011"
}
```

### Response

**Status:** `201 Created`

```json
{
  "data": {
    "user": {
      "id": 11,
      "name": "Sales API Testing",
      "email": "sales.api.20260723133902@demo.piposmart.id",
      "phone": "6281299900011",
      "role": "SALES",
      "status": "ACTIVE",
      "must_change_password": true,
      "permissions": [
        "leads.work",
        "reports.read_own"
      ]
    },
    "temporary_password": "masked-temporary-password"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Admin dapat membuat Sales. API hanya membuat role `SALES`, bukan Admin/Supervisor.

## SALES-003 - POST `/sales` Duplicate Email

### Response

**Status:** `409 Conflict`

```json
{
  "error": {
    "code": "EMAIL_ALREADY_USED",
    "message": "email sudah digunakan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API menolak email Sales yang sudah dipakai.

## SALES-004 - POST `/sales` Payload Invalid

### Request

```json
{
  "name": "Sales Payload Invalid"
}
```

### Response

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "Key: 'CreateSalesRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
    },
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API mengembalikan validation error saat field wajib tidak dikirim.

## SALES-005 - GET `/sales/{id}`

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 11,
    "name": "Sales API Testing",
    "email": "sales.api.20260723133902@demo.piposmart.id",
    "role": "SALES",
    "status": "ACTIVE"
  }
}
```

### Result

PASS. Detail Sales dapat dibaca oleh Admin.

## SALES-006 - PATCH `/sales/{id}`

### Request

```json
{
  "name": "Sales API Testing Updated"
}
```

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 11,
    "name": "Sales API Testing Updated",
    "email": "sales.api.20260723133902@demo.piposmart.id",
    "role": "SALES",
    "status": "ACTIVE"
  }
}
```

### Result

PASS. Data Sales berhasil diperbarui.

## SALES-007 - POST `/sales/{id}/deactivate`

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 11,
    "status": "INACTIVE"
  }
}
```

### Result

PASS. Sales dapat dinonaktifkan dan session user terkait direvoke.

## SALES-008 - POST `/sales/{id}/activate`

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "id": 11,
    "status": "ACTIVE"
  }
}
```

### Result

PASS. Sales dapat diaktifkan kembali.

## SALES-009 - POST `/sales/{id}/reset-password`

### Response

**Status:** `200 OK`

```json
{
  "data": {
    "user": {
      "id": 11,
      "email": "sales.api.20260723133902@demo.piposmart.id",
      "must_change_password": true
    },
    "temporary_password": "masked-temporary-password"
  }
}
```

### Result

PASS. Password Sales berhasil direset dan session lama direvoke.

## SALES-010 - GET `/sales/999999999`

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

PASS. API mengembalikan response standar untuk data Sales yang tidak ditemukan.

---

# 4. RBAC

## RBAC-001 - Login Supervisor

### Response

**Status:** `200 OK`

Supervisor berhasil login dan mendapatkan permission `users.manage_sales`.

### Result

PASS.

## RBAC-002 - Supervisor Membuat Sales

### Request

```json
{
  "name": "Sales Supervisor API",
  "email": "sales.supervisor.api.20260723133902@demo.piposmart.id",
  "phone": "6281299900099"
}
```

### Response

**Status:** `201 Created`

```json
{
  "data": {
    "user": {
      "email": "sales.supervisor.api.20260723133902@demo.piposmart.id",
      "role": "SALES",
      "status": "ACTIVE"
    },
    "temporary_password": "masked-temporary-password"
  }
}
```

### Result

PASS. Supervisor dapat membuat Sales, sesuai keputusan Sprint 3.

## RBAC-003 - Login Sales

### Response

**Status:** `200 OK`

Sales berhasil login dengan permission terbatas:

```json
[
  "leads.work",
  "reports.read_own"
]
```

### Result

PASS.

## RBAC-004 - Sales Mengakses `/sales`

### Request

```http
GET /api/v1/sales
Authorization: Bearer {sales_access_token}
```

### Response

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. Sales tidak dapat membaca data user global.

---

# 5. Error Handling

## ERR-001 - Route Tidak Ditemukan

### Request

```http
GET /api/v1/route-not-found
```

### Response

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "ROUTE_NOT_FOUND",
    "message": "Route tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### Result

PASS. API mengembalikan error envelope standar untuk route yang tidak tersedia.

---

# 6. Findings

## Defect

Tidak ada defect fungsional yang ditemukan pada scope Sprint 03.

## Technical Notes

- Error response sudah memakai envelope standar `error.code`, `error.message`, dan `error.request_id`.
- Protected endpoint menolak request tanpa token dengan status `401`.
- Endpoint yang tidak boleh diakses oleh role tertentu mengembalikan status `403`.
- Duplicate email Sales mengembalikan status `409`.
- Payload invalid mengembalikan status `400` dengan detail validasi.
- Refresh token lama tidak dapat digunakan kembali setelah rotation.
- Temporary password muncul hanya pada response create/reset Sales. Frontend perlu memperlakukannya sebagai one-time secret dan tidak menyimpannya di log.

## Data Created During Testing

Pengujian membuat data Sales tambahan pada database lokal:

```text
sales.api.20260723133902@demo.piposmart.id
sales.supervisor.api.20260723133902@demo.piposmart.id
sales.duplicate.report@demo.piposmart.id
```

Data tersebut hanya berada di local development database.

---

# 7. Conclusion

Berdasarkan manual smoke test Sprint 03, seluruh endpoint yang diuji berjalan sesuai ekspektasi.

| Area | Status |
| --- | --- |
| Core health check | PASS |
| OpenAPI availability | PASS |
| Authentication | PASS |
| Refresh token rotation | PASS |
| Sales Management | PASS |
| RBAC Admin/Supervisor/Sales | PASS |
| Error Handling | PASS |

**Overall API Testing Status:** `PASS`

API Sprint 03 layak digunakan untuk demo internal dan integrasi awal frontend dengan catatan integration test DB otomatis tetap perlu ditambahkan pada sprint berikutnya.
