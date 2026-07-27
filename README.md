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
- baseline schema, factory, dan seeder (preset `minimal` dan `large`);
- background job queue berbasis MySQL (tanpa Redis) yang diproses worker — dipakai KPI recompute (Sprint 13), siap dipakai job import (Sprint 14);
- modul bisnis: identity/RBAC, owner/outlet, lead & assignment, customer activity & training, catalog paket/promo, closing, wallet, subscription & reconciliation, partner/PIC/referral/komisi/payout, sales target/KPI/ranking;
- Dockerfile multi-stage dan Docker Compose;
- pipeline test, vet, build, dan container build.

Status progres mengikuti roadmap `BACKEND_PLAN_SPRINT.md` (18 Sprint) — Sprint 1 s.d. 13 selesai dan
terdokumentasi di `docs/sprint-01/` s.d. `docs/sprint-13/`. Sprint 11b/11c menambah preset seeder
`large` (data skala besar untuk load test) tanpa endpoint API baru; Sprint 12 diperluas hari yang
sama dengan addendum TIER commission, effective-dated/package-scoped commission rule, dan
Payout/PayoutItem batching (`docs/sprint-12/ADDENDUM_02_commission_rules_payouts.md`).

> 📖 **Baru di project ini atau bingung dengan istilah teknis** (idempotent, row lock, snapshot,
> effective dating, job queue, RBAC, dll)? Lihat **[`docs/GLOSARIUM.md`](docs/GLOSARIUM.md)** —
> glosarium lengkap tiap istilah dengan penjelasan detail + analogi bahasa sederhana.

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

Worker dijalankan pada terminal lain. Sejak Sprint 13, worker bukan lagi sekadar heartbeat —
worker memproses job dari `job_queue` (MySQL murni, tanpa Redis), dipakai KPI recompute dan siap
dipakai job import Excel di Sprint 14:

```powershell
go run . worker
```

Preset seeder demo yang tersedia:

| Preset | Skala | Kegunaan |
| --- | --- | --- |
| `minimal` (default) | Beberapa baris per modul | Demo cepat dan smoke test harian. |
| `large` | Ribuan baris (Sprint 11b/11c) | Load test, uji index/query, dan simulasi data produksi sebelum UAT. |

Kedua preset otomatis menyertakan skenario Sales Target/KPI/Ranking (Sprint 13) — 3 Sales dengan
klasifikasi `ACHIEVED`/`NEAR_ACHIEVED`/`NOT_ACHIEVED` yang deterministik, langsung terlihat lewat
`GET /api/v1/kpi/ranking` tanpa perlu menjalankan worker (recompute dipanggil langsung oleh
seeder untuk skenario demo).

```powershell
go run . seed demo --preset=large --seed=20260723 --as-of=2026-07-01
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
crm seed demo --preset=large --seed=20260723 --as-of=2026-07-01
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
| List Owner | GET | /api/v1/owners | Melihat daftar owner aktif. |
| List Owner Trash | GET | /api/v1/owners/trash | Melihat daftar owner yang soft delete. |
| List Owner Unscoped | GET | /api/v1/owners/unscoped | Melihat seluruh owner aktif + terhapus. |
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
| List Outlet | GET | /api/v1/owners/{owner_id}/outlets | Melihat daftar outlet aktif milik owner. |
| List Outlet Trash (per Owner) | GET | /api/v1/owners/{owner_id}/outlets/trash | Melihat outlet soft delete milik owner. |
| List Outlet Unscoped (per Owner) | GET | /api/v1/owners/{owner_id}/outlets/unscoped | Melihat semua outlet milik owner, termasuk soft delete. |
| List Global Outlet | GET | /api/v1/outlets | Melihat semua outlet aktif lintas owner sesuai visibility actor. |
| List Global Outlet Trash | GET | /api/v1/outlets/trash | Melihat semua outlet soft delete lintas owner. |
| List Global Outlet Unscoped | GET | /api/v1/outlets/unscoped | Melihat semua outlet aktif + soft delete lintas owner. |
| Detail Global Outlet | GET | /api/v1/outlets/{outlet_id} | Detail outlet global berisi info owner, wallet owner, dan ringkasan subscription. |
| Outlet Subscription Status Recap | GET | /api/v1/outlets/subscription-statuses | Rekap status langganan outlet per bulan, termasuk outlet yang belum pernah subscribe. |
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
| List Package | GET | /api/v1/catalog/packages | List paket aktif. |
| List Package Trash | GET | /api/v1/catalog/packages/trash | List paket soft delete. |
| List Package Unscoped | GET | /api/v1/catalog/packages/unscoped | List paket aktif + terhapus. |
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
### Closing dan Laporan Penjualan

Mulai Sprint 8, closing penjualan dipisahkan dari activity biasa. Sales membuat
closing dari lead miliknya melalui endpoint khusus. Endpoint ini menyimpan:

- snapshot package, plan/tenor, harga, dan promo;
- perhitungan `base_price`, `discount_amount`, `additional_charge`,
  `unique_transfer_code`, dan `final_amount`;
- status awal `PENDING_RECONCILIATION`;
- interaction remark score `3` dan stage history `CLOSING` dalam satu transaksi.

Aturan utama:

- Sales hanya dapat membuat closing untuk lead yang sedang menjadi miliknya.
- Remark score `3` tidak boleh dibuat langsung melalui endpoint interaction.
- Jika create closing gagal, remark `3` juga batal karena berada dalam satu
  database transaction.
- Promo berbayar tidak dipilih otomatis; frontend harus mengirim
  `promotion_id` jika customer memilih promo tersebut.
- Snapshot closing tidak berubah walaupun master package/plan/promo diedit.
- Pending closing belum dianggap confirmed KPI/revenue closing.

Header wajib:

```http
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Closing | GET | /api/v1/closings | List closing aktif sesuai role actor. |
| List Closing Trash | GET | /api/v1/closings/trash | List closing soft delete sesuai role actor. |
| List Closing Unscoped | GET | /api/v1/closings/unscoped | List closing aktif + terhapus sesuai role actor. |
| Detail Closing | GET | `/api/v1/closings/{closing_id}` | Detail closing dan snapshot transaksi. |
| Create Lead Closing | POST | `/api/v1/leads/{lead_id}/closings` | Sales membuat closing untuk lead miliknya. |
| Confirm Closing | POST | `/api/v1/closings/{closing_id}/confirm` | Admin/Supervisor confirm closing pending. |
| Reject Closing | POST | `/api/v1/closings/{closing_id}/reject` | Admin/Supervisor reject closing pending. |
| Soft Delete Closing | DELETE | `/api/v1/closings/{closing_id}` | Soft delete closing. |
| Restore Closing | PATCH | `/api/v1/closings/{closing_id}/restore` | Restore closing soft-deleted. |
| Force Delete Closing | DELETE | `/api/v1/closings/{closing_id}/force` | Hard delete closing permanen. |
| Bulk Soft Delete | DELETE | `/api/v1/closings/bulk` | Bulk soft delete closing. |
| Bulk Restore | PATCH | `/api/v1/closings/bulk/restore` | Bulk restore closing. |
| Bulk Force Delete | DELETE | `/api/v1/closings/bulk/force` | Bulk hard delete closing. |

Query list closing:

| Parameter | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Laundry Cerah` | Search kode closing, kode lead, atau nama owner. |
| `status` | `PENDING_RECONCILIATION` | Filter status closing. |
| `sales_id` | `3` | Filter Sales. |
| `supervisor_id` | `2` | Filter Supervisor. |
| `lead_id` | `10` | Filter lead. |
| `owner_id` | `15` | Filter owner. |
| `plan_id` | `8` | Filter plan. |
| `closed_from` | `2026-07-01` | Batas awal tanggal closing. |
| `closed_to` | `2026-07-31` | Batas akhir tanggal closing. |
| `page` | `1` | Halaman list. |
| `limit` | `10` | Maksimal 100. |
| `sort` | `-closed_at` | Sort: `closed_at`, `created_at`, `updated_at`, `status`, `final_amount`, `code`. |

#### POST `/api/v1/leads/{lead_id}/closings`

```http
POST /api/v1/leads/10/closings
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
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

Response `201 Created`:

```json
{
  "data": {
    "id": 1,
    "code": "CLS-20260723-000010-123456",
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
      "code": "BUSINESS_12_MONTHS",
      "tenure_months": 12,
      "duration_days": 360,
      "price": "1698600.00"
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

#### POST `/api/v1/closings/{closing_id}/confirm`

```http
POST /api/v1/closings/1/confirm
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
```

```json
{
  "note": "Sudah cocok dengan laporan pembayaran manual"
}
```

Response `200 OK` mengembalikan status `CONFIRMED`.

#### Contoh error closing

Sales membuat closing untuk lead milik Sales lain:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

Discount membuat final amount negatif:

```json
{
  "plan_id": 8,
  "discount_amount": "999999999.00"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "FINAL_AMOUNT_NEGATIVE",
    "message": "final amount tidak boleh negatif",
    "request_id": "generated-request-id"
  }
}
```

Promo tidak eligible untuk plan:

```json
{
  "error": {
    "code": "INVALID_PROMOTION",
    "message": "promo tidak eligible untuk plan yang dipilih",
    "request_id": "generated-request-id"
  }
}
```
### Payment, Top-up, dan Wallet Ledger

Mulai Sprint 9, wallet owner dipakai untuk mencatat saldo internal aplikasi Piposmart. Top-up dianggap revenue perusahaan berdasarkan `paid_at`, sedangkan debit/penggunaan saldo dicatat di ledger agar tidak terjadi double counting dengan closing.

Aturan penting:

- Admin dapat mencatat top-up, debit, adjustment, dan refund.
- Supervisor dan Sales hanya dapat membaca wallet/payment/ledger sesuai cakupan owner yang dapat mereka akses.
- Top-up membuat `wallet_payments` dan ledger `CREDIT` dalam satu transaksi database.
- Debit, adjustment debit, dan refund tidak boleh membuat saldo negatif.
- `idempotency_key` atau `external_reference` wajib dikirim agar transaksi yang sama tidak diproses dua kali.
- Nilai uang dikirim sebagai string decimal, contoh `"1000000.00"`, bukan JSON number/float.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Wallet | GET | `/api/v1/wallets` | Melihat wallet seluruh owner sesuai role user, termasuk owner yang belum pernah top-up (saldo `0.00`). |
| Detail Owner Wallet | GET | `/api/v1/owners/{owner_id}/wallet` | Melihat saldo wallet satu owner. |
| List Payment | GET | `/api/v1/wallet-payments` | Rekap payment/top-up owner berdasarkan `paid_at`. |
| Detail Payment | GET | `/api/v1/wallet-payments/{payment_id}` | Detail top-up/payment. |
| List Ledger | GET | `/api/v1/wallet-transactions` | Melihat semua transaksi ledger sesuai role. |
| List Owner Ledger | GET | `/api/v1/owners/{owner_id}/wallet/transactions` | Melihat ledger untuk satu owner. |
| Create Top-up | POST | `/api/v1/owners/{owner_id}/wallet/topups` | Admin mencatat top-up owner. |
| Create Debit | POST | `/api/v1/owners/{owner_id}/wallet/debits` | Admin mencatat saldo terpakai/manual debit. |
| Create Adjustment | POST | `/api/v1/owners/{owner_id}/wallet/adjustments` | Admin mencatat koreksi saldo credit/debit. |
| Create Refund | POST | `/api/v1/owners/{owner_id}/wallet/refunds` | Admin mencatat saldo keluar karena refund. |

Query list wallet:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Laundry Cerah` | Search account code, kode owner, atau nama owner. |
| `owner_id` | `1` | Filter wallet milik owner tertentu. |
| `status` | `ACTIVE` | Filter status wallet. |
| `sort` | `-balance` | Sort by `created_at`, `updated_at`, `balance`, atau `code`. Prefix `-` berarti descending. |
| `page` | `1` | Halaman data. |
| `limit` | `10` | Jumlah data per halaman, maksimal 100. |

Query list payment/top-up:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `MIDTRANS` | Search kode payment, external reference, atau nama owner. |
| `owner_id` | `1` | Filter payment owner tertentu. |
| `payment_type` | `TOPUP` | Filter jenis payment. Saat ini tersedia `TOPUP`. |
| `channel` | `MIDTRANS` | Filter channel payment. |
| `paid_from` | `2026-07-01` | Batas awal tanggal top-up/revenue. |
| `paid_to` | `2026-07-31` | Batas akhir tanggal top-up/revenue. |
| `sort` | `-paid_at` | Sort by `paid_at`, `created_at`, `amount`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

Query list ledger:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `DEMO-TOPUP` | Search kode ledger, source reference, external reference, atau nama owner. |
| `owner_id` | `1` | Filter ledger owner tertentu. |
| `direction` | `CREDIT` | Filter arah ledger: `CREDIT` atau `DEBIT`. |
| `type` | `CREDIT` | Filter tipe transaksi: `CREDIT`, `DEBIT`, `ADJUSTMENT`, `REFUND`. |
| `occurred_from` | `2026-07-01` | Batas awal tanggal ledger. |
| `occurred_to` | `2026-07-31` | Batas akhir tanggal ledger. |
| `sort` | `-occurred_at` | Sort by `occurred_at`, `created_at`, `amount`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

#### POST `/api/v1/owners/{owner_id}/wallet/topups`

Contoh request:

```http
POST /api/v1/owners/1/wallet/topups
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "amount": "1000000.00",
  "payment_channel": "MIDTRANS",
  "external_reference": "MIDTRANS-TRX-20260723-001",
  "idempotency_key": "topup-owner-1-20260723-001",
  "paid_at": "2026-07-23T10:30:00Z",
  "note": "Top-up dari aplikasi Piposmart"
}
```

Contoh response `201 Created`:

```json
{
  "data": {
    "payment": {
      "id": 1,
      "code": "PAY-20260723103000-000001-000000",
      "payment_type": "TOPUP",
      "payment_channel": "MIDTRANS",
      "amount": "1000000.00",
      "currency": "IDR",
      "status": "PAID"
    },
    "transaction": {
      "id": 1,
      "transaction_type": "CREDIT",
      "direction": "CREDIT",
      "amount": "1000000.00",
      "balance_before": "0.00",
      "balance_after": "1000000.00"
    },
    "wallet": {
      "id": 1,
      "account_code": "WALLET-OWNER-000001",
      "balance": "1000000.00",
      "ledger_balance": "1000000.00",
      "currency": "IDR",
      "status": "ACTIVE"
    },
    "idempotent": false
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Jika request yang sama dikirim ulang dengan `idempotency_key` yang sama, API mengembalikan data existing dengan `idempotent: true` dan saldo tidak bertambah dua kali.

#### POST `/api/v1/owners/{owner_id}/wallet/debits`

Contoh request:

```http
POST /api/v1/owners/1/wallet/debits
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "amount": "250000.00",
  "source_reference": "MANUAL-USAGE-001",
  "external_reference": "CRM-DEBIT-20260723-001",
  "idempotency_key": "debit-owner-1-20260723-001",
  "occurred_at": "2026-07-23T11:00:00Z",
  "note": "Saldo digunakan untuk pembelian paket, sebelum modul order Sprint 10"
}
```

Contoh response `201 Created`:

```json
{
  "data": {
    "id": 2,
    "transaction_type": "DEBIT",
    "direction": "DEBIT",
    "amount": "250000.00",
    "balance_before": "1000000.00",
    "balance_after": "750000.00",
    "source_type": "MANUAL_DEBIT"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/owners/{owner_id}/wallet/adjustments`

Adjustment membutuhkan `direction` karena bisa menambah atau mengurangi saldo.

```json
{
  "amount": "50000.00",
  "direction": "CREDIT",
  "source_reference": "ADJ-ADMIN-001",
  "idempotency_key": "adjustment-owner-1-001",
  "note": "Koreksi saldo manual"
}
```

#### Contoh error wallet

Top-up tanpa `idempotency_key` dan tanpa `external_reference`:

```http
POST /api/v1/owners/1/wallet/topups
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "100000.00",
  "payment_channel": "MANUAL"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "IDEMPOTENCY_REQUIRED",
    "message": "idempotency_key atau external_reference wajib dikirim",
    "details": null,
    "request_id": "generated-request-id"
  }
}
```

Debit melebihi saldo:

```json
{
  "amount": "999999999.00",
  "idempotency_key": "debit-over-balance-owner-1"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "saldo wallet tidak mencukupi",
    "details": null,
    "request_id": "generated-request-id"
  }
}
```

Sales mencoba membuat adjustment:

```http
POST /api/v1/owners/1/wallet/adjustments
Authorization: Bearer {sales_access_token}
Content-Type: application/json
```

Response `403 Forbidden`:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "details": null,
    "request_id": "generated-request-id"
  }
}
```

List payment revenue top-up bulan Juli:

```http
GET /api/v1/wallet-payments?paid_from=2026-07-01&paid_to=2026-07-31&sort=-paid_at&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
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

### Subscription, Order, dan Reconciliation

Mulai Sprint 10, pembelian paket subscription dicatat sebagai `subscription_order` yang mendebit saldo wallet owner, membuat subscription aktif, dan dapat direkonsiliasi dengan closing Sales. Revenue perusahaan tetap diambil dari top-up/payment berdasarkan `paid_at`; order subscription tidak boleh dijumlahkan ulang sebagai revenue agar tidak terjadi double counting.

Aturan penting:

- Create subscription order hanya boleh dilakukan Admin.
- Manual reconciliation boleh dilakukan Admin dan Supervisor sesuai cakupan data.
- `idempotency_key` atau `external_reference` wajib pada create order.
- Wallet debit, order, subscription, dan subscription period dibuat dalam satu database transaction.
- Subscription memakai fixed duration: `tenure_months x 30 hari`, ditambah benefit free duration jika promo snapshot memiliki benefit tersebut.
- Order dengan `closing_id` akan dicoba auto reconciliation.
- Order tanpa closing menjadi `PAID` dan masuk `reconciliation_issues` sebagai `HANGING_ORDER`.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Subscription Order | GET | `/api/v1/subscription-orders` | Melihat order pembelian paket sesuai role/ownership. |
| Detail Subscription Order | GET | `/api/v1/subscription-orders/{order_id}` | Melihat detail order, snapshot paket/plan/promo, wallet transaction, dan status reconciliation. |
| Create Owner Subscription Order | POST | `/api/v1/owners/{owner_id}/subscription-orders` | Admin membuat pembelian paket dari saldo wallet owner. |
| Manual Reconcile Order | POST | `/api/v1/subscription-orders/{order_id}/reconcile` | Admin/Supervisor confirm atau reject order terhadap closing. |
| List Subscription | GET | `/api/v1/subscriptions` | Melihat subscription owner aktif/expired/cancelled. |
| Detail Subscription | GET | `/api/v1/subscriptions/{subscription_id}` | Melihat detail subscription owner. |
| List Reconciliation | GET | `/api/v1/reconciliations` | Melihat histori auto/manual reconciliation order dengan closing. |
| List Reconciliation Issue | GET | `/api/v1/reconciliation-issues` | Melihat hanging transaction dan issue reconciliation yang perlu review. |

Query list subscription order:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Owner Laundry` | Search kode order, kode owner, nama owner, atau external reference. |
| `owner_id` | `4` | Filter berdasarkan owner. |
| `closing_id` | `9` | Filter berdasarkan closing. |
| `sales_id` | `3` | Filter berdasarkan Sales. |
| `supervisor_id` | `2` | Filter berdasarkan Supervisor. |
| `plan_id` | `1` | Filter berdasarkan plan. |
| `status` | `RECONCILED` | Filter status order: `PAID`, `RECONCILED`, `REJECTED`, `CANCELED`. |
| `purchased_from` | `2026-07-01` | Batas awal tanggal pembelian. |
| `purchased_to` | `2026-07-31` | Batas akhir tanggal pembelian. |
| `sort` | `-purchased_at` | Sort by `purchased_at`, `created_at`, `updated_at`, `final_amount`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

Query list subscription:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `OWN-00004` | Search kode subscription, kode owner, atau nama owner. |
| `owner_id` | `4` | Filter berdasarkan owner. |
| `order_id` | `3` | Filter berdasarkan order. |
| `plan_id` | `1` | Filter berdasarkan plan. |
| `status` | `ACTIVE` | Filter status: `ACTIVE`, `EXPIRED`, `CANCELED`. |
| `active_from` | `2026-07-01` | Batas awal tanggal aktif. |
| `active_to` | `2026-07-31` | Batas akhir tanggal aktif. |
| `sort` | `-active_from` | Sort by `active_from`, `active_until`, `created_at`, `updated_at`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

Query list reconciliation issue:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `HANGING` | Search kode issue, kode order, atau nama owner. |
| `order_id` | `2` | Filter issue berdasarkan order. |
| `owner_id` | `4` | Filter issue berdasarkan owner. |
| `issue_type` | `HANGING_ORDER` | Filter tipe issue: `HANGING_ORDER`, `CLOSING_MISMATCH`, `MANUAL_REVIEW`. |
| `status` | `OPEN` | Filter status issue: `OPEN`, `RESOLVED`. |
| `sort` | `-detected_at` | Sort by `detected_at`, `created_at`, `updated_at`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

#### GET `/api/v1/subscription-orders`

Contoh request:

```http
GET /api/v1/subscription-orders?status=RECONCILED&purchased_from=2026-07-01&purchased_to=2026-07-31&sort=-purchased_at&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Contoh response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "DEMO-ORD-000003-RDER-APRIL-TOPUP-JULY-PURCHASE-OWNER-003",
        "owner": {
          "id": 3,
          "code": "OWN-00003",
          "name": "Owner Laundry 003"
        },
        "closing": {
          "id": 9,
          "code": "DEMO-CLS-000003-PRO_12_MONTHS"
        },
        "plan": {
          "id": 13,
          "code": "PRO_12_MONTHS",
          "name": "Pro 12 Bulan"
        },
        "duration_days": 360,
        "final_amount": "3768703.00",
        "status": "RECONCILED",
        "purchased_at": "2026-07-10T13:00:00Z",
        "subscription_start_date": "2026-07-10"
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

#### POST `/api/v1/owners/{owner_id}/subscription-orders`

Contoh request order tanpa closing:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 1,
  "idempotency_key": "subscription-order-owner-4-20260724-001",
  "external_reference": "ADMIN-DASHBOARD-SUB-20260724-001",
  "purchased_at": "2026-07-24T03:00:00Z",
  "subscription_start_date": "2026-07-24",
  "note": "Pembelian Basic 1 bulan dari saldo wallet"
}
```

Contoh response `201 Created`:

```json
{
  "data": {
    "order": {
      "id": 3,
      "owner": {
        "id": 4,
        "code": "OWN-00004",
        "name": "Owner Laundry 004"
      },
      "wallet_transaction_id": 13,
      "plan": {
        "id": 1,
        "code": "BASIC_01_MONTHS"
      },
      "duration_days": 30,
      "final_amount": "99000.00",
      "status": "PAID"
    },
    "subscription": {
      "id": 3,
      "status": "ACTIVE",
      "active_from": "2026-07-24",
      "active_until": "2026-08-23",
      "total_duration_days": 30
    },
    "period": {
      "period_index": 1,
      "start_date": "2026-07-24",
      "end_date": "2026-08-23",
      "duration_days": 30
    },
    "issue": {
      "issue_type": "HANGING_ORDER",
      "status": "OPEN"
    },
    "idempotent": false
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Jika `closing_id` dikirim dan cocok, order akan dibuat memakai snapshot closing dan reconciliation dapat langsung `CONFIRMED`.

#### POST `/api/v1/subscription-orders/{order_id}/reconcile`

Contoh request manual confirm:

```http
POST /api/v1/subscription-orders/4/reconcile
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "action": "CONFIRM",
  "closing_id": 7,
  "note": "Cocok dengan closing Sales Juli"
}
```

Contoh response `200 OK`:

```json
{
  "data": {
    "order": {
      "id": 4,
      "status": "RECONCILED"
    },
    "reconciliation": {
      "status": "CONFIRMED",
      "match_type": "MANUAL",
      "amount_difference": "-101.00"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### Contoh error subscription order

Create order tanpa `idempotency_key` dan tanpa `external_reference`:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "plan_id": 1,
  "purchased_at": "2026-07-20T10:00:00Z",
  "subscription_start_date": "2026-07-20"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "IDEMPOTENCY_REQUIRED",
    "message": "idempotency_key atau external_reference wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

Saldo wallet tidak cukup:

```json
{
  "plan_id": 13,
  "idempotency_key": "subscription-order-owner-4-pro-12",
  "purchased_at": "2026-07-24T04:00:00Z",
  "subscription_start_date": "2026-07-24"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "saldo wallet tidak mencukupi",
    "request_id": "generated-request-id"
  }
}
```

Sales mencoba membuat order:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {sales_access_token}
Content-Type: application/json
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

Reconcile order yang sudah `RECONCILED`:

```json
{
  "action": "CONFIRM",
  "closing_id": 9,
  "note": "negative test already reconciled"
}
```

Response `409 Conflict`:

```json
{
  "error": {
    "code": "ORDER_ALREADY_RECONCILED",
    "message": "order sudah direkonsiliasi",
    "request_id": "generated-request-id"
  }
}
```

Dokumentasi testing lengkap Sprint 10 tersedia di `docs/sprint-10/README.md`.

### Partner, PIC, Referral, dan Call Mitra

Mulai Sprint 11, tim Sales dapat mengelola mitra (partner) — supplier hardware/POS, distributor software, agent regional, atau komunitas referral — beserta PIC yang bertanggung jawab, riwayat call/chat ke mitra, dan customer lead yang direferensikan mitra tersebut.

Aturan penting:

- Satu mitra hanya boleh memiliki **satu PIC aktif** dalam satu waktu (di-enforce di database lewat generated column + `UNIQUE KEY`, bukan hanya validasi aplikasi — lihat `migrations/20260724001100_partner_assignment_concurrency_guard.sql`). Assign PIC baru otomatis melepas PIC lama secara atomik dalam satu transaction dengan row lock, aman terhadap assignment bersamaan.
- Sales hanya melihat mitra yang PIC-nya adalah dirinya sendiri; Admin/Supervisor melihat semua.
- Nomor rekening mitra dienkripsi di database (`bank_account_encrypted`); response API hanya menampilkan `bank_account_masked` (contoh: `****1234`), tidak pernah nomor lengkap.
- Referral tidak dapat didaftarkan dua kali untuk pasangan mitra-lead yang sama (`UNIQUE KEY` pada `partner_id, lead_id`).

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Partner Type | GET | `/api/v1/partner-types` | Melihat daftar tipe mitra (SUPPLIER, DISTRIBUTOR, AGENT, REFERRAL_PARTNER, REFERRAL_REGULAR). |
| Create Partner Type | POST | `/api/v1/partner-types` | Membuat tipe mitra baru dengan rate komisi flat (`commission_mode`/`commission_value`). |
| Detail Partner Type | GET | `/api/v1/partner-types/{id}` | Melihat detail tipe mitra. |
| Update Partner Type | PUT | `/api/v1/partner-types/{id}` | Mengubah rate komisi flat tipe mitra. |
| List Partner | GET | `/api/v1/partners` | Melihat daftar mitra (Sales hanya melihat mitra kelolaannya). |
| Create Partner | POST | `/api/v1/partners` | Mendaftarkan mitra baru. |
| Detail Partner | GET | `/api/v1/partners/{partnerID}` | Melihat detail mitra. |
| Detail Partner by Code | GET | `/api/v1/partners/code/{code}` | Mencari mitra berdasarkan kode. |
| Update Partner | PUT | `/api/v1/partners/{partnerID}` | Mengubah data mitra (termasuk ganti nomor rekening). |
| Deactivate Partner | DELETE | `/api/v1/partners/{partnerID}` | Menonaktifkan mitra. |
| Active Assignment | GET | `/api/v1/partners/{partnerID}/assignments/active` | Melihat PIC aktif mitra saat ini. |
| List Assignment | GET | `/api/v1/partners/{partnerID}/assignments` | Melihat riwayat PIC mitra. |
| Assign PIC | POST | `/api/v1/partners/{partnerID}/assignments` | Menetapkan PIC baru (otomatis melepas PIC lama). |
| Release PIC | DELETE | `/api/v1/partners/{partnerID}/assignments/release` | Melepas PIC aktif tanpa menggantinya. |
| List Interaction | GET | `/api/v1/partners/{partnerID}/interactions` | Melihat riwayat call/chat ke mitra. |
| Record Interaction | POST | `/api/v1/partners/{partnerID}/interactions` | Mencatat call/chat ke mitra. |
| List Referral | GET | `/api/v1/partners/{partnerID}/referrals` | Melihat customer lead yang direferensikan mitra. |
| Create Referral | POST | `/api/v1/partners/{partnerID}/referrals` | Mendaftarkan referral mitra ke lead. |

#### POST `/api/v1/partners`

Request:

```http
POST /api/v1/partners
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "partner_type_id": 3,
  "code": "PTR-AGENT-001",
  "name": "Agen Regional Bandung",
  "phone": "6281200001111",
  "bank_account": "1234567890123456"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 5,
    "partner_type": {
      "id": 3,
      "code": "AGENT",
      "name": "Agent Regional",
      "commission_mode": "PERCENTAGE",
      "commission_value": "5.00"
    },
    "code": "PTR-AGENT-001",
    "name": "Agen Regional Bandung",
    "phone": "6281200001111",
    "bank_account_masked": "****3456",
    "status": "ACTIVE"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Nomor rekening lengkap yang dikirim di request **tidak pernah** dikembalikan lagi di response manapun — hanya 4 digit terakhir.

#### POST `/api/v1/partners/{partnerID}/assignments`

Request:

```http
POST /api/v1/partners/5/assignments
Authorization: Bearer {supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "user_id": 3
}
```

Response `201 Created`: assignment baru menjadi `active: true`; assignment PIC sebelumnya (jika ada) otomatis mendapat `unassigned_at` terisi dan `active: false` dalam transaction yang sama.

Contoh error assign bersamaan (dua request assign PIC berbeda ke mitra yang sama, dikirim nyaris bersamaan):

```json
{
  "error": {
    "code": "INVALID_ASSIGNMENT",
    "message": "partner: invalid partner assignment (only one active PIC allowed)",
    "request_id": "generated-request-id"
  }
}
```

Row lock pada `AssignPIC` menyerialkan request yang bersamaan — hasil akhirnya selalu tepat satu PIC aktif, tidak pernah dua.

#### POST `/api/v1/partners/{partnerID}/referrals`

Request:

```http
POST /api/v1/partners/5/referrals
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "lead_id": 24,
  "notes": "Referral dari agen regional Bandung"
}
```

Contoh error referral duplikat (lead yang sama direferensikan mitra yang sama dua kali):

```json
{
  "error": {
    "code": "DUPLICATE_REFERRAL",
    "message": "partner: referral already exists for this partner-lead pair",
    "request_id": "generated-request-id"
  }
}
```

Dokumentasi testing lengkap Sprint 11 tersedia di `docs/sprint-11/README.md`.

### Partner Commission dan Payout

Mulai Sprint 12 (diperluas hari yang sama dengan addendum TIER/effective-date/Payout), komisi mitra dihitung otomatis dari closing `CONFIRMED` yang terhubung ke referral mitra, dengan lifecycle persetujuan bertingkat sebelum dibayarkan — baik satu per satu maupun dibatch dalam satu payout.

Aturan penting:

- Rate komisi punya 3 sumber, dicek berurutan saat sync: **commission rule** yang effective-dated & opsional package-scoped (`commission_rules`, mode `PERCENTAGE`/`FIXED`/`TIER`) → kalau tidak ada yang cocok, **fallback ke rate flat** `partner_types.commission_mode`/`commission_value` (perilaku dasar sejak Sprint 12, tidak pernah berubah).
- Mode **TIER**: rate ditentukan dari bracket volume closing `CONFIRMED` kumulatif mitra **per bulan kalender** — closing ke-1 s/d ke-3 bisa punya rate berbeda dari closing ke-4 dan seterusnya, tergantung konfigurasi tier.
- Rule package-specific selalu mengalahkan rule type-wide; kalau ada beberapa rule berlaku, `effective_from` paling baru yang menang.
- Lifecycle commission: `PENDING → APPROVED → PAID`, atau `→ CANCELLED` (dari PENDING/APPROVED). Sync bersifat idempotent — closing yang sudah punya commission tidak diproses ulang.
- Role gating: sync/approve/cancel/kelola commission rule = ADMIN atau SUPERVISOR; pay (baik individual maupun payout) = **ADMIN saja**.
- Payout membatch beberapa commission `APPROVED` milik satu mitra menjadi satu pencairan. Commission yang sudah masuk payout aktif **tidak bisa** dibayar individual (double-pay guard, dijamin database lewat `UNIQUE KEY` pada generated column, bukan hanya validasi aplikasi).
- Cancel payout melepas commission kembali ke `APPROVED` (soft-release, `released_at` diisi) — riwayat payout yang dibatalkan tetap tersimpan untuk audit, tidak dihapus.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| Create Commission Rule | POST | `/api/v1/partner-types/{id}/commission-rules` | Membuat rate komisi effective-dated (PERCENTAGE/FIXED/TIER), opsional package-scoped. |
| List Commission Rule | GET | `/api/v1/partner-types/{id}/commission-rules` | Melihat daftar commission rule tipe mitra. |
| Detail Commission Rule | GET | `/api/v1/partner-types/{id}/commission-rules/{ruleID}` | Melihat detail rule termasuk tier (jika mode TIER). |
| Deactivate Commission Rule | PATCH | `/api/v1/partner-types/{id}/commission-rules/{ruleID}/deactivate` | Menonaktifkan rule (supersede lewat rule baru, bukan edit di tempat). |
| Sync Commission | POST | `/api/v1/partners/{partnerID}/commissions/sync` | Membuat commission `PENDING` dari closing `CONFIRMED` yang belum diproses. |
| List Commission | GET | `/api/v1/partners/{partnerID}/commissions` | Melihat daftar commission mitra. |
| Detail Commission | GET | `/api/v1/partners/{partnerID}/commissions/{commissionID}` | Melihat detail satu commission. |
| Approve Commission | PATCH | `/api/v1/partners/{partnerID}/commissions/{commissionID}/approve` | Menyetujui commission (PENDING → APPROVED). |
| Pay Commission | PATCH | `/api/v1/partners/{partnerID}/commissions/{commissionID}/pay` | Membayar satu commission secara individual (APPROVED → PAID), ADMIN saja. |
| Cancel Commission | PATCH | `/api/v1/partners/{partnerID}/commissions/{commissionID}/cancel` | Membatalkan commission. |
| Create Payout | POST | `/api/v1/partners/{partnerID}/payouts` | Membatch seluruh commission `APPROVED` mitra menjadi satu payout `PENDING`. |
| List Payout | GET | `/api/v1/partners/{partnerID}/payouts` | Melihat daftar payout mitra. |
| Detail Payout | GET | `/api/v1/partners/{partnerID}/payouts/{payoutID}` | Melihat detail payout beserta item commission-nya. |
| Pay Payout | PATCH | `/api/v1/partners/{partnerID}/payouts/{payoutID}/pay` | Membayar payout, mengubah seluruh commission di dalamnya jadi `PAID`, ADMIN saja. |
| Cancel Payout | PATCH | `/api/v1/partners/{partnerID}/payouts/{payoutID}/cancel` | Membatalkan payout, melepas commission kembali ke `APPROVED`. |

#### POST `/api/v1/partner-types/{id}/commission-rules`

Contoh rule TIER — closing ke-1 s/d ke-3 dalam sebulan dapat 2%, closing ke-4 dan seterusnya dapat 5%:

```http
POST /api/v1/partner-types/3/commission-rules
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "mode": "TIER",
  "effective_from": "2026-01-01T00:00:00Z",
  "tiers": [
    { "tier_order": 1, "min_closings": 1, "max_closings": 3, "mode": "PERCENTAGE", "value": "2.00" },
    { "tier_order": 2, "min_closings": 4, "mode": "PERCENTAGE", "value": "5.00" }
  ]
}
```

Response `201 Created`: object rule dengan `tiers` berisi kedua bracket. Tier terakhir wajib `max_closings` kosong (open-ended).

#### POST `/api/v1/partners/{partnerID}/commissions/sync`

Request:

```http
POST /api/v1/partners/5/commissions/sync
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "created": 1,
    "items": [
      {
        "id": 9,
        "code": "COM-20260725-000009-747811",
        "partner_id": 5,
        "closing_id": 12,
        "commission_mode": "PERCENTAGE",
        "commission_value": "5.00",
        "commission_rule_id": 2,
        "tier_ordinal": null,
        "base_amount": "3000000.00",
        "commission_amount": "150000.00",
        "currency": "IDR",
        "status": "PENDING"
      }
    ]
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

`commission_rule_id` dan `tier_ordinal` (khusus mode TIER) menunjukkan rule/bracket mana yang dipakai — `null` berarti memakai fallback rate flat `partner_types`.

#### POST `/api/v1/partners/{partnerID}/payouts`

Request:

```http
POST /api/v1/partners/5/payouts
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `201 Created`:

```json
{
  "data": {
    "id": 3,
    "code": "PAYOUT-20260725-000005-866534",
    "partner_id": 5,
    "total_amount": "150000.00",
    "currency": "IDR",
    "status": "PENDING",
    "items": [
      { "id": 6, "commission_id": 9, "commission_code": "COM-20260725-000009-747811", "amount": "150000.00" }
    ]
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error mencoba membayar commission yang sudah masuk payout aktif:

```json
{
  "error": {
    "code": "COMMISSION_IN_PAYOUT",
    "message": "partner: commission is already reserved in an active payout",
    "request_id": "generated-request-id"
  }
}
```

Dokumentasi testing lengkap Sprint 12 (termasuk addendum TIER/effective-date/Payout) tersedia di `docs/sprint-12/README.md` dan `docs/sprint-12/ADDENDUM_02_commission_rules_payouts.md`.

### Sales Target, KPI, dan Ranking

Mulai Sprint 13, performa Sales dihitung otomatis dari aktivitas CRM (call/chat, training, closing `CONFIRMED`) terhadap target bulanan, diproses **asinkron lewat background worker** (`job_queue`, MySQL murni tanpa Redis — worker sebelumnya cuma heartbeat stub, sekarang benar-benar memproses job).

Aturan penting:

- Target dan KPI Definition memakai `metric_codes` yang sudah ada sejak Sprint 2 (`CALL_CUSTOMER_COUNT`, `TRAINING_COUNT`, `CONFIRMED_CLOSING_COUNT`, `CONFIRMED_CLOSING_AMOUNT`, `PARTNER_CALL_COUNT` — metric terakhir belum didukung recompute, lihat catatan di bawah).
- Bulk-set target (`POST /sales-targets/bulk`) **tidak pernah menimpa** target yang sudah ada (baik hasil bulk maupun override sebelumnya) — hanya mengisi Sales yang belum punya target di periode itu. Override (`PUT /sales-targets/{salesID}`) **selalu menang**.
- Total weight seluruh KPI definition **aktif** dalam satu periode wajib tepat 100% — divalidasi saat recompute dijalankan, bukan saat definisi dibuat satu-satu.
- Recompute **idempoten**: dijalankan ulang untuk periode yang sama kapan pun menghasilkan hasil identik (delete-then-insert dalam satu transaction).
- Klasifikasi per Sales per periode: `ACHIEVED` (total score ≥ threshold_achieved), `NEAR_ACHIEVED` (≥ threshold_near), atau `NOT_ACHIEVED`.
- Sales hanya melihat KPI miliknya sendiri (termasuk posisi rank-nya) lewat `/kpi/results`; ranking lengkap satu periode (`/kpi/ranking`) hanya untuk Admin/Supervisor.
- `PARTNER_CALL_COUNT` sengaja **belum didukung** recompute (atribusi ke Sales tidak langsung, perlu join time-scoped ke assignment PIC mitra) — kalau ada KPI definition yang mengaktifkan metric ini, recompute akan gagal eksplisit dengan `UNSUPPORTED_METRIC`, bukan diam-diam dihitung nol.

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| Bulk Set Target | POST | `/api/v1/sales-targets/bulk` | Set target default untuk seluruh Sales aktif yang belum punya target di periode & metric ini. |
| Override Target | PUT | `/api/v1/sales-targets/{salesID}` | Menimpa target satu Sales untuk periode & metric tertentu. |
| List Target | GET | `/api/v1/sales-targets` | Melihat daftar target (Sales hanya melihat miliknya). |
| Create KPI Definition | POST | `/api/v1/kpi-definitions` | Mendefinisikan bobot & threshold satu metric untuk satu periode. |
| List KPI Definition | GET | `/api/v1/kpi-definitions` | Melihat daftar KPI definition. |
| Detail KPI Definition | GET | `/api/v1/kpi-definitions/{id}` | Melihat detail satu KPI definition. |
| Deactivate KPI Definition | PATCH | `/api/v1/kpi-definitions/{id}/deactivate` | Menonaktifkan definition (supersede lewat definition baru). |
| Trigger Recompute | POST | `/api/v1/kpi/recompute` | Meng-enqueue job recompute KPI untuk satu periode (diproses worker). |
| Job Status | GET | `/api/v1/kpi/jobs/{id}` | Mengecek status job recompute (`PENDING`/`PROCESSING`/`COMPLETED`/`FAILED`). |
| List KPI Result | GET | `/api/v1/kpi/results` | Melihat hasil KPI (Sales hanya melihat miliknya). |
| List Ranking | GET | `/api/v1/kpi/ranking` | Melihat ranking lengkap satu periode. Admin/Supervisor saja. |

#### POST `/api/v1/sales-targets/bulk`

Request:

```http
POST /api/v1/sales-targets/bulk
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "period_year": 2026,
  "period_month": 8,
  "metric_code": "CONFIRMED_CLOSING_COUNT",
  "target_value": "10.00"
}
```

Response `200 OK`:

```json
{
  "data": {
    "metric_code": "CONFIRMED_CLOSING_COUNT",
    "period_year": 2026,
    "period_month": 8,
    "target_value": "10.00",
    "eligible_sales": 5,
    "created": 5
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### POST `/api/v1/kpi/recompute`

Request:

```http
POST /api/v1/kpi/recompute
Authorization: Bearer {admin_or_supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "period_year": 2026,
  "period_month": 7
}
```

Response `202 Accepted`:

```json
{
  "data": {
    "id": 2,
    "job_type": "KPI_RECOMPUTE",
    "status": "PENDING",
    "attempts": 0,
    "max_attempts": 5
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Cek status lewat `GET /api/v1/kpi/jobs/2` setelah beberapa detik (sesuai `WORKER_POLL_INTERVAL`):

```json
{
  "data": {
    "id": 2,
    "job_type": "KPI_RECOMPUTE",
    "status": "COMPLETED",
    "attempts": 1,
    "max_attempts": 5,
    "completed_at": "2026-07-25T09:45:01Z"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error total weight definition aktif belum 100% (job gagal setelah dicoba ulang sebanyak `max_attempts`, bukan langsung ditolak saat trigger karena recompute berjalan async):

```json
{
  "data": {
    "id": 1,
    "status": "FAILED",
    "attempts": 5,
    "max_attempts": 5,
    "last_error": "kpi: active KPI definitions for this period must have weights summing to exactly 100"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

#### GET `/api/v1/kpi/ranking`

Request:

```http
GET /api/v1/kpi/ranking?period_year=2026&period_month=7
Authorization: Bearer {admin_or_supervisor_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      { "sales_id": 5, "sales_name": "Sales Demo 091", "total_score": "100.00", "classification": "ACHIEVED", "rank_position": 1 },
      { "sales_id": 6, "sales_name": "Sales Demo 092", "total_score": "80.00", "classification": "NEAR_ACHIEVED", "rank_position": 2 },
      { "sales_id": 7, "sales_name": "Sales Demo 093", "total_score": "20.00", "classification": "NOT_ACHIEVED", "rank_position": 3 }
    ]
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Contoh error Sales mencoba mengakses ranking penuh:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "kpi: forbidden",
    "request_id": "generated-request-id"
  }
}
```

Sales tetap bisa melihat posisi rank-nya sendiri lewat `GET /api/v1/kpi/results` (hanya mengembalikan baris miliknya).

Dokumentasi testing lengkap Sprint 13 tersedia di `docs/sprint-13/README.md`.

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
internal/app/                  Lifecycle executable (RunAPI, RunWorker, RunBootstrapAdmin)
internal/platform/config/      Konfigurasi dan validasi environment
internal/platform/database/    Koneksi dan pool MySQL
internal/platform/factory/     Factory data dummy deterministik
internal/platform/httpserver/  Router, middleware, health, OpenAPI
internal/platform/httpx/       Response envelope API
internal/platform/jobqueue/    Job queue generik berbasis MySQL (klaim, retry, stale reclaim)
internal/platform/logging/     Structured logging
internal/platform/migration/   Goose runner
internal/platform/seeder/      Master dan demo seeder (preset minimal & large)
migrations/                    SQL migration
```

## Struktur Modul Bisnis

Setiap modul mengikuti pola file yang sama: `types.go` (struct & response), `errors.go`,
`money.go` (bila ada validasi desimal/persen), `repository.go`, `service.go`, `handler.go`.

```text
internal/identity/     Auth, RBAC, Sales management (Sprint 3)
internal/customer/     Owner & Outlet (Sprint 4)
internal/lead/         Customer lead & assignment (Sprint 5)
internal/activity/     Interaction, remark, follow-up, training (Sprint 6)
internal/catalog/      Package, plan, promotion, benefit (Sprint 7)
internal/closing/      Sales closing & laporan penjualan (Sprint 8)
internal/wallet/       Payment, top-up, wallet ledger (Sprint 9)
internal/subscription/ Subscription order & reconciliation (Sprint 10)
internal/partner/      Partner, PIC, referral, commission, payout (Sprint 11, 12)
internal/target/       Sales target — bulk & override (Sprint 13)
internal/kpi/          KPI definition, recompute, ranking (Sprint 13)
```


