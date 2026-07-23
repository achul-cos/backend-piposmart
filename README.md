# Backend CRM Piposmart

Backend internal CRM PT Piposmart Digital Indonesia. Fondasi baru memakai Go,
Gin, GORM, MySQL 8, Goose, dan OpenAPI.

Fondasi saat ini menyediakan:

- entrypoint root `main.go`, sehingga command lokal cukup `go run . ...`;
- konfigurasi tervalidasi dari environment dan `.env`;
- API dan worker dengan graceful shutdown;
- structured logging, request ID, panic recovery, serta CORS eksplisit;
- liveness dan readiness check;
- migration command berbasis Goose;
- baseline schema, factory, dan seeder awal;
- Dockerfile multi-stage dan Docker Compose;
- pipeline test, vet, build, dan container build.

## Prasyarat

- Go sesuai versi pada `go.mod`;
- MySQL 8 untuk menjalankan tanpa Docker; atau
- Docker Engine dan Docker Compose v2 untuk environment container.

## Menjalankan dengan Docker

Salin konfigurasi contoh:

```powershell
Copy-Item .env.example .env
```

Ganti minimal `DB_PASSWORD` dan `MYSQL_ROOT_PASSWORD` untuk kebutuhan lokal.
Kemudian:

```powershell
docker compose up --build
```

Service yang dijalankan:

- `mysql`: database MySQL 8;
- `migrate`: menjalankan Goose sekali sebelum API;
- `api`: HTTP API di `http://localhost:8080`;
- `worker`: fondasi background worker.

Endpoint demo:

```text
GET http://localhost:8080/
GET http://localhost:8080/health/live
GET http://localhost:8080/health/ready
GET http://localhost:8080/api/v1/status
GET http://localhost:8080/swagger/index.html
```

Menghentikan environment:

```powershell
docker compose down
```

Data MySQL berada pada named volume. Gunakan `docker compose down --volumes`
hanya ketika memang ingin menghapus seluruh data development.

## Menjalankan secara Lokal

Salin `.env.example` menjadi `.env`, kemudian ubah:

```dotenv
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=crm_piposmart
DB_USER=crm_user
DB_PASSWORD=your-local-password
MIGRATION_DIR=./migrations
UPLOAD_DIR=./storage/uploads
EXPORT_DIR=./storage/exports
```

Jalankan migration dan API:

```powershell
go run . migrate up
go run . seed master
go run . bootstrap-admin
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
go run . api
```

Worker dijalankan pada terminal lain:

```powershell
go run . worker
```

Command yang tersedia:

```text
crm api
crm worker
crm migrate up
crm migrate down
crm migrate status
crm migrate version
crm seed master
crm seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
crm bootstrap-admin
crm version
crm help
```

`bootstrap-admin` membuat akun Admin awal berdasarkan variable
`BOOTSTRAP_ADMIN_*` di `.env`. Seeder demo juga membuat akun berikut:

```text
admin.001@demo.piposmart.id / Password123!
supervisor.001@demo.piposmart.id / Password123!
sales.001@demo.piposmart.id / Password123!
sales.002@demo.piposmart.id / Password123!
```

Contoh login dan memakai access token:

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/auth/login `
  -ContentType 'application/json' `
  -Body '{"email":"admin.001@demo.piposmart.id","password":"Password123!"}'

$token = $login.data.access_token
Invoke-RestMethod `
  -Method Get `
  -Uri http://localhost:8080/api/v1/auth/me `
  -Headers @{ Authorization = "Bearer $token" }
```

Contoh membuat Sales baru:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/sales `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType 'application/json' `
  -Body '{"name":"Sales Baru","email":"sales.baru@demo.piposmart.id","phone":"6281212345678"}'
```

## Dokumentasi Route API

Base URL lokal:

```text
http://localhost:8080
```

Base path API bisnis:

```text
/api/v1
```

Semua response memakai envelope standar.

Response sukses:

```json
{
  "data": {},
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Response error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": null,
    "request_id": "generated-request-id"
  }
}
```

Header untuk route publik:

```http
Accept: application/json
Content-Type: application/json
```

Header untuk route protected:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

### System, Health, dan Dokumentasi API

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| Service Info | GET | `/` | Melihat informasi service backend. |
| Health Live | GET | `/health/live` | Mengecek proses API hidup. |
| Health Ready | GET | `/health/ready` | Mengecek API siap dan database terhubung. |
| API Status | GET | `/api/v1/status` | Mengecek status API versi 1. |
| OpenAPI YAML | GET | `/openapi.yaml` | Mengambil kontrak OpenAPI mentah. |
| Swagger UI | GET | `/swagger/index.html` | Membuka UI dokumentasi API interaktif. |

Contoh request:

```http
GET /health/ready
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "status": "ready"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error:

```http
GET /health/ready
Accept: application/json
```

Jika database tidak dapat diakses:

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Dependency belum siap",
    "request_id": "generated-request-id"
  }
}
```

### Authentication

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| Login | POST | `/api/v1/auth/login` | Login user dan mendapatkan access token serta refresh token. |
| Refresh Token | POST | `/api/v1/auth/refresh` | Menukar refresh token lama dengan token baru. |
| Logout | POST | `/api/v1/auth/logout` | Revoke refresh token. |
| Me | GET | `/api/v1/auth/me` | Melihat profil user yang sedang login. |
| Change Password | POST | `/api/v1/auth/change-password` | Mengubah password user yang sedang login. |

#### POST `/api/v1/auth/login`

Request:

```http
POST /api/v1/auth/login
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "email": "admin.001@demo.piposmart.id",
  "password": "Password123!"
}
```

Response `200 OK`:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "refresh-token-value",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token_expires_at": "2026-08-22T08:00:00Z",
    "user": {
      "id": 1,
      "name": "Admin Demo 001",
      "email": "admin.001@demo.piposmart.id",
      "role": "ADMIN",
      "status": "ACTIVE"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error request:

```json
{
  "email": "admin.001@demo.piposmart.id",
  "password": "password-salah"
}
```

Response `401 Unauthorized`:

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "email atau password tidak valid",
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/auth/refresh`

Request:

```http
POST /api/v1/auth/refresh
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "refresh_token": "refresh-token-value"
}
```

Response `200 OK`: format sama seperti login, berisi access token dan refresh token baru.

Contoh error request:

```json
{
  "refresh_token": "token-tidak-valid"
}
```

Response `401 Unauthorized`:

```json
{
  "error": {
    "code": "INVALID_TOKEN",
    "message": "token tidak valid",
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/auth/logout`

Request:

```http
POST /api/v1/auth/logout
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "refresh_token": "refresh-token-value"
}
```

Response `200 OK`:

```json
{
  "data": {
    "status": "logged_out"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error tanpa header:

```http
POST /api/v1/auth/logout
Accept: application/json
```

Response `401 Unauthorized`:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

#### GET `/api/v1/auth/me`

Request:

```http
GET /api/v1/auth/me
Authorization: Bearer {access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "code": "ADM-001",
    "name": "Admin Demo 001",
    "email": "admin.001@demo.piposmart.id",
    "role": "ADMIN",
    "status": "ACTIVE",
    "permissions": ["owners.manage", "users.manage_sales"]
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/auth/change-password`

Request:

```http
POST /api/v1/auth/change-password
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "current_password": "Password123!",
  "new_password": "PasswordBaru123!"
}
```

Response `200 OK`:

```json
{
  "data": {
    "status": "password_changed"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error body:

```json
{
  "current_password": "Password123!",
  "new_password": "short"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "Field validation for 'NewPassword' failed on the 'min' tag"
    },
    "request_id": "generated-request-id"
  }
}
```

### Sales Management

Route Sales membutuhkan JWT Admin atau Supervisor. Sales tidak boleh mengelola
akun Sales lain.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Sales | GET | `/api/v1/sales` | Melihat daftar akun Sales. |
| Create Sales | POST | `/api/v1/sales` | Membuat akun Sales baru. |
| Detail Sales | GET | `/api/v1/sales/{id}` | Melihat detail akun Sales. |
| Update Sales | PATCH | `/api/v1/sales/{id}` | Mengubah profil akun Sales. |
| Activate Sales | POST | `/api/v1/sales/{id}/activate` | Mengaktifkan akun Sales. |
| Deactivate Sales | POST | `/api/v1/sales/{id}/deactivate` | Menonaktifkan akun Sales dan revoke session aktif. |
| Reset Sales Password | POST | `/api/v1/sales/{id}/reset-password` | Reset password akun Sales. |

#### GET `/api/v1/sales`

Query params:

| Param | Tipe | Contoh | Fungsi |
| --- | --- | --- | --- |
| `status` | string | `ACTIVE` atau `INACTIVE` | Filter status Sales. |

Request:

```http
GET /api/v1/sales?status=ACTIVE
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 3,
        "code": "SLS-001",
        "name": "Sales Demo 001",
        "email": "sales.001@demo.piposmart.id",
        "role": "SALES",
        "status": "ACTIVE"
      }
    ],
    "total": 1
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/sales`

Request:

```http
POST /api/v1/sales
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "code": "SLS-NEW",
  "name": "Sales Baru",
  "email": "sales.baru@demo.piposmart.id",
  "phone": "6281212345678",
  "password": "Password123!"
}
```

Response `201 Created`:

```json
{
  "data": {
    "user": {
      "id": 10,
      "code": "SLS-NEW",
      "name": "Sales Baru",
      "email": "sales.baru@demo.piposmart.id",
      "role": "SALES",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "Password123!"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Jika `password` dikosongkan, sistem membuat temporary password otomatis.

Contoh error duplicate email:

```json
{
  "name": "Sales Duplikat",
  "email": "sales.001@demo.piposmart.id"
}
```

Response `409 Conflict`:

```json
{
  "error": {
    "code": "EMAIL_ALREADY_USED",
    "message": "email sudah digunakan",
    "request_id": "generated-request-id"
  }
}
```

#### GET `/api/v1/sales/{id}`

Path params:

| Param | Tipe | Fungsi |
| --- | --- | --- |
| `id` | integer | ID user Sales. |

Request:

```http
GET /api/v1/sales/3
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object user Sales.

Contoh error:

```http
GET /api/v1/sales/999999
Authorization: Bearer {admin_or_supervisor_access_token}
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

#### PATCH `/api/v1/sales/{id}`

Request:

```http
PATCH /api/v1/sales/3
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "name": "Sales Demo Updated",
  "phone": "6281299998888"
}
```

Response `200 OK`: mengembalikan object user Sales terbaru.

#### POST `/api/v1/sales/{id}/activate`

Request:

```http
POST /api/v1/sales/3/activate
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object user Sales dengan `status = ACTIVE`.

#### POST `/api/v1/sales/{id}/deactivate`

Request:

```http
POST /api/v1/sales/3/deactivate
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object user Sales dengan `status = INACTIVE`.

#### POST `/api/v1/sales/{id}/reset-password`

Request:

```http
POST /api/v1/sales/3/reset-password
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body opsional:

```json
{
  "new_password": "PasswordBaru123!"
}
```

Response `200 OK`:

```json
{
  "data": {
    "user": {
      "id": 3,
      "name": "Sales Demo 001",
      "email": "sales.001@demo.piposmart.id",
      "role": "SALES",
      "must_change_password": true
    },
    "temporary_password": "PasswordBaru123!"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error akses Sales:

```http
GET /api/v1/sales
Authorization: Bearer {sales_access_token}
Accept: application/json
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

### Owner Management

Route Owner membutuhkan JWT Admin atau Supervisor dengan permission
`owners.manage`. Sales belum mengakses route ini secara langsung; akses Sales
ke customer miliknya akan masuk melalui modul Lead/Assignment.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Owner | GET | `/api/v1/owners` | Melihat daftar owner aktif. |
| Create Owner | POST | `/api/v1/owners` | Membuat owner baru. |
| Bulk Create Owner | POST | `/api/v1/owners/bulk` | Membuat banyak owner dalam satu request. |
| Bulk Update Owner | PATCH | `/api/v1/owners/bulk` | Mengubah banyak owner dalam satu request. |
| Bulk Soft Delete Owner | DELETE | `/api/v1/owners/bulk` | Soft delete banyak owner. |
| Bulk Hard Delete Owner | DELETE | `/api/v1/owners/bulk/force` | Menghapus permanen banyak owner. |
| Detail Owner | GET | `/api/v1/owners/{owner_id}` | Melihat detail owner aktif. |
| Update Owner | PATCH | `/api/v1/owners/{owner_id}` | Mengubah data owner. |
| Soft Delete Owner | DELETE | `/api/v1/owners/{owner_id}` | Mengisi `deleted_at` owner. |
| Restore Owner | PATCH | `/api/v1/owners/{owner_id}/restore` | Memulihkan owner soft-deleted. |
| Hard Delete Owner | DELETE | `/api/v1/owners/{owner_id}/force` | Menghapus owner permanen. |

#### GET `/api/v1/owners`

Query params:

| Param | Tipe | Contoh | Fungsi |
| --- | --- | --- | --- |
| `q` | string | `Laundry Cerah` | Search global pada kode, nama, telepon, brand, kota, provinsi. |
| `code` | string | `OWN-00001` | Filter kode owner. |
| `name` | string | `Budi` | Filter nama owner. |
| `phone` | string | `0812` | Filter nomor telepon. |
| `brand_name` | string | `Laundry Cerah` | Filter brand laundry. |
| `province` | string | `Jawa Barat` | Filter provinsi. |
| `city` | string | `Bandung` | Filter kota. |
| `page` | integer | `1` | Nomor halaman, default `1`. |
| `limit` | integer | `10` | Jumlah data per halaman, default `10`, maksimum `100`. |
| `sort` | string | `-created_at` | Sorting. Field: `created_at`, `updated_at`, `code`, `name`, `brand_name`, `city`, `province`. Prefix `-` untuk descending. |

Request:

```http
GET /api/v1/owners?q=Laundry&page=1&limit=10&sort=-created_at
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "OWN-00001",
        "name": "Owner Laundry 001",
        "phone": "6281320000001",
        "brand_name": "Laundry Cerah 001",
        "province": "Jawa Barat",
        "city": "Bandung",
        "status": "ACTIVE",
        "outlet_count": 2
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

Contoh error sort:

```http
GET /api/v1/owners?sort=unknown
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "parameter sort tidak valid",
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/owners`

Request:

```http
POST /api/v1/owners
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "code": "OWN-API-001",
  "name": "Budi Santoso",
  "phone": "0812-3456-7890",
  "email": "budi@example.test",
  "brand_name": "Laundry Cerah",
  "province": "Jawa Barat",
  "city": "Bandung",
  "address": "Jl. Contoh No. 1"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 20,
    "code": "OWN-API-001",
    "name": "Budi Santoso",
    "phone": "6281234567890",
    "email": "budi@example.test",
    "brand_name": "Laundry Cerah",
    "province": "Jawa Barat",
    "city": "Bandung",
    "address": "Jl. Contoh No. 1",
    "status": "ACTIVE"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error invalid phone:

```json
{
  "code": "OWN-BAD-PHONE",
  "name": "Owner Bad Phone",
  "phone": "abc"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_PHONE",
    "message": "nomor telepon tidak valid",
    "request_id": "generated-request-id"
  }
}
```

#### GET `/api/v1/owners/{owner_id}`

Path params:

| Param | Tipe | Fungsi |
| --- | --- | --- |
| `owner_id` | integer | ID owner. |

Request:

```http
GET /api/v1/owners/20
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object owner.

Contoh error:

```http
GET /api/v1/owners/999999
Authorization: Bearer {admin_or_supervisor_access_token}
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

#### PATCH `/api/v1/owners/{owner_id}`

Request:

```http
PATCH /api/v1/owners/20
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "name": "Budi Santoso Updated",
  "phone": "+62 812 9999 8888",
  "city": "Bekasi"
}
```

Response `200 OK`: mengembalikan object owner terbaru.

#### DELETE `/api/v1/owners/{owner_id}`

Request:

```http
DELETE /api/v1/owners/20
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "status": "deleted"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Soft delete owner tidak menghapus outlet child.

#### PATCH `/api/v1/owners/{owner_id}/restore`

Request:

```http
PATCH /api/v1/owners/20/restore
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object owner dengan `status = ACTIVE`.

#### DELETE `/api/v1/owners/{owner_id}/force`

Request:

```http
DELETE /api/v1/owners/20/force
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "status": "force_deleted"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Hard delete owner menghapus row owner permanen. Outlet child tidak ikut terhapus;
`owner_id` pada outlet menjadi `null`.

#### POST `/api/v1/owners/bulk`

Request:

```http
POST /api/v1/owners/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

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

Response `201 Created`:

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
      }
    ],
    "total": 2
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### PATCH `/api/v1/owners/bulk`

Request:

```http
PATCH /api/v1/owners/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

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

Response `200 OK`: mengembalikan daftar owner yang berhasil diperbarui.

#### DELETE `/api/v1/owners/bulk`

Request:

```http
DELETE /api/v1/owners/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "ids": [21, 22]
}
```

Response `200 OK`:

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

#### DELETE `/api/v1/owners/bulk/force`

Request dan response sama seperti bulk soft delete, tetapi row owner dihapus permanen.

Contoh error bulk kosong:

```json
{
  "ids": []
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "Field validation for 'IDs' failed on the 'min' tag"
    },
    "request_id": "generated-request-id"
  }
}
```

### Outlet Management

Route Outlet berada di bawah Owner. Owner harus aktif untuk create/list/detail
outlet melalui nested route.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Outlet | GET | `/api/v1/owners/{owner_id}/outlets` | Melihat daftar outlet aktif milik owner. |
| Create Outlet | POST | `/api/v1/owners/{owner_id}/outlets` | Membuat outlet untuk owner. |
| Bulk Create Outlet | POST | `/api/v1/owners/{owner_id}/outlets/bulk` | Membuat banyak outlet. |
| Bulk Update Outlet | PATCH | `/api/v1/owners/{owner_id}/outlets/bulk` | Mengubah banyak outlet. |
| Bulk Soft Delete Outlet | DELETE | `/api/v1/owners/{owner_id}/outlets/bulk` | Soft delete banyak outlet. |
| Bulk Hard Delete Outlet | DELETE | `/api/v1/owners/{owner_id}/outlets/bulk/force` | Menghapus permanen banyak outlet. |
| Detail Outlet | GET | `/api/v1/owners/{owner_id}/outlets/{outlet_id}` | Melihat detail outlet. |
| Update Outlet | PATCH | `/api/v1/owners/{owner_id}/outlets/{outlet_id}` | Mengubah outlet. |
| Soft Delete Outlet | DELETE | `/api/v1/owners/{owner_id}/outlets/{outlet_id}` | Mengisi `deleted_at` outlet. |
| Restore Outlet | PATCH | `/api/v1/owners/{owner_id}/outlets/{outlet_id}/restore` | Memulihkan outlet soft-deleted. |
| Hard Delete Outlet | DELETE | `/api/v1/owners/{owner_id}/outlets/{outlet_id}/force` | Menghapus outlet permanen. |

#### GET `/api/v1/owners/{owner_id}/outlets`

Query params:

| Param | Tipe | Contoh | Fungsi |
| --- | --- | --- | --- |
| `q` | string | `Outlet Utama` | Search global pada kode, nama, telepon, kota, provinsi. |
| `code` | string | `OUT-00001` | Filter kode outlet. |
| `name` | string | `Outlet Utama` | Filter nama outlet. |
| `phone` | string | `0812` | Filter nomor telepon outlet. |
| `province` | string | `Jawa Barat` | Filter provinsi. |
| `city` | string | `Bandung` | Filter kota. |
| `page` | integer | `1` | Nomor halaman. |
| `limit` | integer | `10` | Jumlah data per halaman, maksimum `100`. |
| `sort` | string | `name` | Field: `created_at`, `updated_at`, `code`, `name`, `city`, `province`. |

Request:

```http
GET /api/v1/owners/20/outlets?q=Outlet&page=1&limit=10&sort=name
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 31,
        "owner_id": 20,
        "code": "OUT-API-001",
        "name": "Laundry Cerah Outlet 1",
        "phone": "6281234567891",
        "province": "Jawa Barat",
        "city": "Bandung",
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

#### POST `/api/v1/owners/{owner_id}/outlets`

Request:

```http
POST /api/v1/owners/20/outlets
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "code": "OUT-API-001",
  "name": "Laundry Cerah Outlet 1",
  "phone": "0812-3456-7891",
  "province": "Jawa Barat",
  "city": "Bandung",
  "address": "Jl. Outlet No. 1"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 31,
    "owner_id": 20,
    "code": "OUT-API-001",
    "name": "Laundry Cerah Outlet 1",
    "phone": "6281234567891",
    "province": "Jawa Barat",
    "city": "Bandung",
    "address": "Jl. Outlet No. 1",
    "status": "ACTIVE"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error owner tidak ada:

```http
POST /api/v1/owners/999999/outlets
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "OUT-NO-OWNER",
  "name": "Outlet Tanpa Owner"
}
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

#### GET `/api/v1/owners/{owner_id}/outlets/{outlet_id}`

Request:

```http
GET /api/v1/owners/20/outlets/31
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object outlet.

#### PATCH `/api/v1/owners/{owner_id}/outlets/{outlet_id}`

Request:

```http
PATCH /api/v1/owners/20/outlets/31
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "name": "Outlet Updated",
  "city": "Bekasi"
}
```

Response `200 OK`: mengembalikan object outlet terbaru.

#### DELETE `/api/v1/owners/{owner_id}/outlets/{outlet_id}`

Request:

```http
DELETE /api/v1/owners/20/outlets/31
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "status": "deleted"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### PATCH `/api/v1/owners/{owner_id}/outlets/{outlet_id}/restore`

Request:

```http
PATCH /api/v1/owners/20/outlets/31/restore
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: mengembalikan object outlet dengan `status = ACTIVE`.

#### DELETE `/api/v1/owners/{owner_id}/outlets/{outlet_id}/force`

Request:

```http
DELETE /api/v1/owners/20/outlets/31/force
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "status": "force_deleted"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/owners/{owner_id}/outlets/bulk`

Request:

```http
POST /api/v1/owners/20/outlets/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "items": [
    {
      "code": "OUT-BULK-001",
      "name": "Outlet Bulk 1",
      "phone": "0812-7100-0001",
      "province": "Jawa Barat",
      "city": "Bandung"
    },
    {
      "code": "OUT-BULK-002",
      "name": "Outlet Bulk 2",
      "phone": "0812-7100-0002",
      "province": "Jawa Barat",
      "city": "Cimahi"
    }
  ]
}
```

Response `201 Created`:

```json
{
  "data": {
    "items": [
      {
        "id": 41,
        "owner_id": 20,
        "code": "OUT-BULK-001",
        "name": "Outlet Bulk 1",
        "phone": "6281271000001",
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

#### PATCH `/api/v1/owners/{owner_id}/outlets/bulk`

Request:

```http
PATCH /api/v1/owners/20/outlets/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "items": [
    {
      "id": 41,
      "name": "Outlet Bulk Updated 1"
    },
    {
      "id": 42,
      "city": "Bekasi"
    }
  ]
}
```

Response `200 OK`: mengembalikan daftar outlet yang berhasil diperbarui.

#### DELETE `/api/v1/owners/{owner_id}/outlets/bulk`

Request:

```http
DELETE /api/v1/owners/20/outlets/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "ids": [41, 42]
}
```

Response `200 OK`:

```json
{
  "data": {
    "ids": [41, 42],
    "affected": 2
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### DELETE `/api/v1/owners/{owner_id}/outlets/bulk/force`

Request dan response sama seperti bulk soft delete, tetapi row outlet dihapus permanen.

### Lead Management dan Ownership

Mulai Sprint 5, visibility owner/customer dikendalikan oleh `customer_leads`.
Aturannya:

- Admin melihat semua owner/lead dan dapat assign ke Supervisor.
- Supervisor melihat lead miliknya dan lead milik Sales di bawahnya.
- Sales hanya melihat lead miliknya.
- Jika Sales menandai invalid, lead kembali ke Supervisor dengan `score = 0`.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Lead | GET | `/api/v1/leads` | Melihat list lead sesuai visibility actor. |
| Create Lead | POST | `/api/v1/leads` | Membuat lead dari owner yang sudah ada. |
| Detail Lead | GET | `/api/v1/leads/{lead_id}` | Melihat detail lead. |
| Assignment History | GET | `/api/v1/leads/{lead_id}/assignment-history` | Melihat riwayat perpindahan ownership. |
| Assign Supervisor | POST | `/api/v1/leads/{lead_id}/assign-supervisor` | Admin membagikan lead ke Supervisor. |
| Assign Sales | POST | `/api/v1/leads/{lead_id}/assign-sales` | Supervisor membagikan/redistribusi lead ke Sales. |
| Release Lead | POST | `/api/v1/leads/{lead_id}/release` | Mengembalikan lead ke Admin. |
| Mark Invalid | POST | `/api/v1/leads/{lead_id}/mark-invalid` | Sales menandai lead invalid dan mengembalikan ke Supervisor. |
| Bulk Assign Supervisor | POST | `/api/v1/leads/bulk/assign-supervisor` | Bulk assign lead ke Supervisor. |
| Bulk Assign Sales | POST | `/api/v1/leads/bulk/assign-sales` | Bulk assign lead ke Sales. |
| Bulk Release | POST | `/api/v1/leads/bulk/release` | Bulk release lead ke Admin. |

#### GET `/api/v1/leads`

Query params:

| Param | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Laundry` | Search lead/owner. |
| `ownership` | `SALES` | Filter current owner role: `ADMIN`, `SUPERVISOR`, `SALES`. |
| `stage` | `POSSIBLE` | Filter stage. |
| `status` | `OPEN` | Filter status. |
| `score` | `0` | Filter score 0-3. |
| `supervisor_id` | `2` | Filter Supervisor. |
| `sales_id` | `3` | Filter Sales. |
| `follow_up_from` | `2026-07-01` | Filter jadwal follow-up awal. |
| `follow_up_to` | `2026-07-31` | Filter jadwal follow-up akhir. |
| `page` | `1` | Nomor halaman. |
| `limit` | `10` | Jumlah data per halaman. |

Request:

```http
GET /api/v1/leads?q=Laundry&ownership=SALES&page=1&limit=10
Authorization: Bearer {access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 10,
        "code": "LEAD-000010",
        "owner": {
          "available": true,
          "id": 20,
          "code": "OWN-00020",
          "name": "Owner Laundry"
        },
        "current_owner_role": "SALES",
        "current_owner": {
          "id": 3,
          "name": "Sales Demo 001",
          "role": "SALES"
        },
        "supervisor": {
          "id": 2,
          "name": "Supervisor Demo 001",
          "role": "SUPERVISOR"
        },
        "stage": "POSSIBLE",
        "status": "OPEN",
        "current_score": 1
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

#### POST `/api/v1/leads/{lead_id}/assign-supervisor`

Request:

```http
POST /api/v1/leads/10/assign-supervisor
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "supervisor_id": 2,
  "reason": "Distribusi dari Admin ke Supervisor"
}
```

Response `200 OK`: lead berpindah ke `current_owner_role = SUPERVISOR`.

#### POST `/api/v1/leads/{lead_id}/assign-sales`

Request:

```http
POST /api/v1/leads/10/assign-sales
Authorization: Bearer {supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "sales_id": 3,
  "reason": "Distribusi dari Supervisor ke Sales"
}
```

Response `200 OK`: lead berpindah ke `current_owner_role = SALES`.

#### POST `/api/v1/leads/{lead_id}/mark-invalid`

Request:

```http
POST /api/v1/leads/10/mark-invalid
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

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
    "stage": "INVALID",
    "status": "INVALID",
    "current_score": 0
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### GET `/api/v1/leads/{lead_id}/assignment-history`

Request:

```http
GET /api/v1/leads/10/assignment-history
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`: daftar riwayat perpindahan ownership lead.

Contoh error Sales lain membuka lead bukan miliknya:

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

### Customer Activity, Follow-up, dan Training

Mulai Sprint 6, Sales dapat mencatat aktivitas call/chat customer, remark score,
jadwal follow-up, dan training/demo aplikasi. Visibility tetap mengikuti
ownership lead:

- Admin melihat seluruh activity.
- Supervisor melihat activity miliknya dan activity Sales di bawahnya.
- Sales hanya melihat activity lead miliknya.
- Interaksi bersifat append-only; perubahan skor/stage disimpan pada stage
  history.

Aturan remark:

- `0` → lead menjadi `INVALID`, ownership kembali ke Supervisor, Sales kehilangan
  visibility.
- `1` → lead menjadi `POSSIBLE`.
- `1` tidak menurunkan lead yang sudah `POTENTIAL`.
- `2` → lead menjadi `POTENTIAL`.
- `3` → lead menjadi `CLOSING` sementara; integrasi laporan closing akan
  dilanjutkan pada Sprint 8.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Interaction | GET | `/api/v1/customer-interactions` | List call/chat sesuai visibility actor. |
| List Follow-up | GET | `/api/v1/follow-ups` | List activity yang memiliki jadwal follow-up. |
| List Lead Interaction | GET | `/api/v1/leads/{lead_id}/interactions` | List call/chat pada satu lead. |
| Create Interaction | POST | `/api/v1/leads/{lead_id}/interactions` | Mencatat call/chat, remark, dan follow-up. |
| Stage History | GET | `/api/v1/leads/{lead_id}/stage-history` | Melihat riwayat perubahan stage/score lead. |
| List Training | GET | `/api/v1/trainings` | List jadwal/laporan training. |
| List Lead Training | GET | `/api/v1/leads/{lead_id}/trainings` | List training pada satu lead. |
| Schedule Training | POST | `/api/v1/leads/{lead_id}/trainings` | Menjadwalkan training online/offline. |
| Reschedule Training | POST | `/api/v1/trainings/{training_id}/reschedule` | Mengubah jadwal training. |
| Complete Training | POST | `/api/v1/trainings/{training_id}/complete` | Menyelesaikan training. |
| Cancel Training | POST | `/api/v1/trainings/{training_id}/cancel` | Membatalkan training. |

#### POST `/api/v1/leads/{lead_id}/interactions`

Request:

```http
POST /api/v1/leads/24/interactions
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "type": "CALL",
  "interaction_at": "2026-07-23T10:00:00+07:00",
  "remark_score": 2,
  "note": "Customer tertarik demo",
  "customer_response": "Minta dijadwalkan training online",
  "follow_up_at": "2026-07-28T10:00:00+07:00",
  "follow_up_note": "Follow-up jadwal training"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 13,
    "lead_id": 24,
    "type": "CALL",
    "remark_score": 2,
    "remark_code": "POTENTIAL",
    "stage_before": "POSSIBLE",
    "stage_after": "POTENTIAL",
    "follow_up_at": "2026-07-28T03:00:00Z"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh remark invalid:

```json
{
  "type": "CALL",
  "remark_score": 0,
  "note": "Customer membatalkan dan tidak potensial"
}
```

Response `201 Created`: interaksi tetap tercatat, lalu lead berpindah ke
Supervisor dengan stage `INVALID`.

#### GET `/api/v1/follow-ups`

Query params:

| Param | Contoh | Fungsi |
| --- | --- | --- |
| `follow_up_from` | `2026-07-28` | Filter jadwal follow-up awal. |
| `follow_up_to` | `2026-07-30` | Filter jadwal follow-up akhir. |
| `sales_id` | `3` | Filter Sales. |
| `page` | `1` | Nomor halaman. |
| `limit` | `10` | Jumlah data per halaman. |

Request:

```http
GET /api/v1/follow-ups?follow_up_from=2026-07-28&follow_up_to=2026-07-30
Authorization: Bearer {sales_access_token}
Accept: application/json
```

#### POST `/api/v1/leads/{lead_id}/trainings`

Request:

```http
POST /api/v1/leads/24/trainings
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "training_type": "ONLINE",
  "scheduled_at": "2026-07-30T13:00:00+07:00",
  "meeting_url": "https://meet.example.test/sprint-06",
  "trainer_name": "Sales Demo 001",
  "participant_name": "Owner Laundry",
  "note": "Demo aplikasi Piposmart"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 4,
    "lead_id": 24,
    "training_type": "ONLINE",
    "status": "SCHEDULED",
    "scheduled_at": "2026-07-30T06:00:00Z"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/trainings/{training_id}/complete`

```http
POST /api/v1/trainings/4/complete
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "completed_at": "2026-07-31T15:30:00+07:00",
  "result_note": "Owner memahami penggunaan kasir dan outlet"
}
```

Contoh error score tidak valid:

```http
POST /api/v1/leads/24/interactions
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "type": "CALL",
  "remark_score": 9
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SCORE",
    "message": "score remark harus 0 sampai 3",
    "request_id": "generated-request-id"
  }
}
```

### Catalog Package, Plan, Promotion, dan Benefit

Mulai Sprint 7, catalog digunakan untuk mengelola paket langganan, tenor/plan,
harga, promo, benefit promo, dan eligibility promo.

Aturan utama:

- Admin dan Supervisor dapat membuat/mengubah catalog.
- Sales hanya dapat membaca catalog.
- `duration_days` plan selalu dihitung dari `tenure_months x 30`.
- Nilai uang dikirim sebagai decimal string, contoh `1698600.00`, bukan float.
- Promo `FREE` diprioritaskan sebagai `recommended_promotion`.
- Promo `PAID` tetap tampil sebagai opsi, tetapi tidak otomatis dipilih.
- Filter `as_of=YYYY-MM-DD` digunakan untuk effective date.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Package | GET | `/api/v1/catalog/packages` | List paket. |
| Create Package | POST | `/api/v1/catalog/packages` | Membuat paket. |
| Detail Package | GET | `/api/v1/catalog/packages/{package_id}` | Detail paket. |
| Update Package | PATCH | `/api/v1/catalog/packages/{package_id}` | Update paket. |
| Delete/Restore/Force Package | DELETE/PATCH/DELETE | `/api/v1/catalog/packages/{package_id}` | Soft delete, restore, hard delete paket. |
| List Plan | GET | `/api/v1/catalog/plans` | List plan/tenor. |
| Create Plan | POST | `/api/v1/catalog/plans` | Membuat plan. |
| Eligible Promotions | GET | `/api/v1/catalog/plans/{plan_id}/eligible-promotions` | Promo eligible per plan. |
| List Promotion | GET | `/api/v1/catalog/promotions` | List promo. |
| Create Promotion | POST | `/api/v1/catalog/promotions` | Membuat promo. |
| Promotion Benefits | GET/POST | `/api/v1/catalog/promotions/{promotion_id}/benefits` | List/tambah benefit. |
| Set Eligible Plans | PUT | `/api/v1/catalog/promotions/{promotion_id}/eligible-plans` | Set plan yang eligible untuk promo. |

Contoh create plan:

```http
POST /api/v1/catalog/plans
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "package_id": 2,
  "code": "BUSINESS_12_MONTHS_TEST",
  "name": "Business 12 Bulan Test",
  "tenure_months": 12,
  "price": "1698600.00",
  "currency": "IDR",
  "effective_from": "2026-07-01"
}
```

Response penting: `duration_days` akan bernilai `360`.

Contoh eligible promo:

```http
GET /api/v1/catalog/plans/2/eligible-promotions?as_of=2026-07-23
Authorization: Bearer {access_token}
Accept: application/json
```

Response memiliki `recommended_promotion` hanya dari promo `FREE`.

Contoh error harga decimal tidak valid:

```json
{
  "package_id": 2,
  "code": "BAD_PRICE",
  "name": "Bad Price",
  "tenure_months": 12,
  "price": "12.345",
  "effective_from": "2026-07-01"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_DECIMAL",
    "message": "nilai uang harus decimal valid",
    "request_id": "generated-request-id"
  }
}
```
### Error Umum API

| Kondisi | Status | Code |
| --- | ---: | --- |
| Tidak mengirim JWT | 401 | `UNAUTHENTICATED` |
| JWT tidak valid | 401 | `INVALID_TOKEN` |
| Role/permission tidak cukup | 403 | `FORBIDDEN` |
| Resource tidak ditemukan | 404 | `NOT_FOUND` |
| Duplicate code | 409 | `CODE_ALREADY_USED` |
| Duplicate email | 409 | `EMAIL_ALREADY_USED` |
| Nomor telepon tidak valid | 400 | `INVALID_PHONE` |
| Sort tidak valid | 400 | `INVALID_SORT` |
| Method tidak tersedia | 405 | `METHOD_NOT_ALLOWED` |

Contoh method tidak tersedia:

```http
PUT /api/v1/owners/20
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Response `405 Method Not Allowed`:

```json
{
  "error": {
    "code": "METHOD_NOT_ALLOWED",
    "message": "HTTP method tidak didukung",
    "request_id": "generated-request-id"
  }
}
```

## Konfigurasi

`.env.example` adalah kontrak konfigurasi. File `.env` tidak boleh dikomit.
Environment yang diberikan oleh OS atau container selalu menang terhadap nilai
dari `.env`.

Production harus:

- menggunakan secret manager untuk database dan key autentikasi;
- memakai origin CORS eksplisit;
- memakai database user non-root;
- menjalankan migration sebagai release job, bukan saat setiap replica API
  dimulai.

## Verifikasi

Jalankan perintah berikut dari root backend:

```powershell
cd C:\piposmart\backend_crm_piposmart
```

### `go test`

`go test` menjalankan seluruh unit test dan integration test ringan pada semua
package Go. Gunakan ini setiap selesai mengubah logic, migration helper,
factory, seeder, handler, service, atau repository.

```powershell
go test ./...
```

Jika muncul error permission pada Go build cache di Windows, gunakan cache lokal
di workspace:

```powershell
$env:GOCACHE='C:\piposmart\backend_crm_piposmart\.cache\go-build'
go test ./...
```

### `go vet`

`go vet` mengecek potensi bug statis yang sering tidak tertangkap compiler,
misalnya format string salah, struct tag mencurigakan, atau penggunaan API yang
rawan keliru. Jalankan sebelum commit atau sebelum laporan Sprint.

```powershell
go vet ./...
```

### `go build`

`go build` memastikan aplikasi bisa dikompilasi menjadi binary. Karena
entrypoint ada di root `main.go`, command build cukup:

```powershell
go build .
```

Di Windows, command ini menghasilkan file binary seperti
`backend_crm_piposmart.exe`. File `.exe` sudah di-ignore oleh Git, jadi tidak
perlu dikomit.

Untuk menjalankan binary hasil build:

```powershell
.\backend_crm_piposmart.exe help
.\backend_crm_piposmart.exe migrate status
```

### Quality gate harian

Sebelum lanjut Sprint berikutnya atau sebelum commit besar, jalankan paket
lengkap ini:

```powershell
$env:GOCACHE='C:\piposmart\backend_crm_piposmart\.cache\go-build'
go test ./...
go vet ./...
go build .
git diff --check
```

Build container:

```powershell
docker build -t crm-piposmart-backend:local .
```

## Struktur Fondasi

```text
main.go                        Entry point API, worker, migration, dan seeder
internal/app/                  Lifecycle executable
internal/platform/config/      Konfigurasi dan validasi environment
internal/platform/database/    Koneksi dan pool MySQL
internal/platform/factory/     Factory data dummy deterministik
internal/platform/httpserver/  Router, middleware, health, OpenAPI
internal/platform/httpx/       Response envelope API
internal/platform/logging/     Structured logging
internal/platform/migration/   Goose runner
internal/platform/seeder/      Master dan demo seeder
migrations/                    SQL migration
```
