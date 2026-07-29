# API Testing Report - Sprint 14f

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14f |
| Tanggal Testing | 29 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| Fokus Testing | Route `/all`, `/all-deleted`, create Admin/Supervisor, reset password Sales/Admin/Supervisor, dan error handler |

## 2. Tujuan Pengujian

Dokumen ini dibuat untuk membantu:

- frontend memahami request/response Sprint 14f;
- QA melakukan smoke test manual;
- CTO membaca hasil validasi API tanpa harus membuka Postman;
- tim backend dan frontend memakai format error yang sama sebagai acuan integrasi.

## 3. Verifikasi Teknis yang Dijalankan

### 3.1 Command

```powershell
go build ./...
go test ./internal/identity/...
go test ./internal/platform/httpserver/...
```

### 3.2 Hasil

| Command | Status | Catatan |
| --- | --- | --- |
| `go build ./...` | PASS | Seluruh package berhasil di-compile |
| `go test ./internal/identity/...` | PASS | Test modul identity lulus |
| `go test ./internal/platform/httpserver/...` | PASS | Test router/health/openapi lulus |

Catatan:

- pada Windows sempat muncul cleanup warning `unlinkat ... Access is denied` setelah package test `httpserver` selesai `ok`;
- hasil test package tetap dianggap lulus karena assertion test sudah selesai sukses.

## 4. Header dan Envelope

### 4.1 Header Auth

```http
Authorization: Bearer {access_token}
Accept: application/json
```

### 4.2 Header POST JSON

```http
Content-Type: application/json
Authorization: Bearer {access_token}
Accept: application/json
```

### 4.3 Format Success

```json
{
  "data": {},
  "meta": {
    "request_id": "0db0ce5df5ef46cf8d1f0fe9db5b9e45"
  }
}
```

### 4.4 Format Error

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "0db0ce5df5ef46cf8d1f0fe9db5b9e45"
  }
}
```

### 4.5 Format Error dengan `details`

Beberapa modul, terutama importing, dapat mengembalikan detail tambahan:

```json
{
  "error": {
    "code": "INVALID_BATCH_STATUS",
    "message": "importing: batch is not in a state that allows this action",
    "details": {
      "root_cause": "Frontend memanggil aksi import pada status batch yang belum sesuai alur backend.",
      "solution": "Poll GET /imports/{id} dan hanya aktifkan aksi yang sesuai dengan status batch saat ini.",
      "frontend_prevent": "Gunakan status batch sebagai source of truth utama."
    },
    "request_id": "0db0ce5df5ef46cf8d1f0fe9db5b9e45"
  }
}
```

## 5. Ringkasan Test Case

| No | Endpoint | Method | Fokus | Status |
| --- | --- | --- | --- | --- |
| 1 | `/owners/all` | GET | Full list owner tanpa pagination | Documented |
| 2 | `/owners/all-deleted` | GET | Full list owner + soft deleted | Documented |
| 3 | `/imports/all` | GET | Histori import tanpa pagination | Documented |
| 4 | `/supervisors` | GET | List supervisor | Documented |
| 5 | `/supervisors` | POST | Create supervisor | Documented |
| 6 | `/admins` | POST | Create admin | Documented |
| 7 | `/supervisors/{id}/reset-password` | POST | Reset password supervisor | Documented |
| 8 | `/admins/{id}/reset-password` | POST | Reset password admin | Documented |
| 9 | `/sales/{id}/reset-password` | POST | Reset password sales | Documented |
| 10 | Auth / validation / conflict / import error | Mixed | Error handling | Documented |

## 6. Detail Testing API

### 6.1 GET `/owners/all`

Request:

```http
GET /api/v1/owners/all?name=laundry&city=bandung&sort=name
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "items": [
      {
        "id": 14,
        "code": "OWN-00014",
        "name": "Budi Laundry",
        "phone": "6281234567890",
        "brand_name": "Bersih Kilat",
        "city": "Bandung",
        "province": "Jawa Barat"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 1,
      "total": 1
    }
  },
  "meta": {
    "request_id": "req-owners-all-001"
  }
}
```

Validasi:

- route tidak memakai paging limit biasa;
- response shape tetap kompatibel untuk frontend lama;
- query filter tetap bekerja.

### 6.2 GET `/owners/all-deleted`

Request:

```http
GET /api/v1/owners/all-deleted?sort=-updated_at
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "items": [
      {
        "id": 14,
        "code": "OWN-00014",
        "name": "Budi Laundry",
        "deleted_at": "2026-07-29T07:12:00Z"
      },
      {
        "id": 11,
        "code": "OWN-00011",
        "name": "Andi Laundry",
        "deleted_at": null
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 2,
      "total": 2
    }
  },
  "meta": {
    "request_id": "req-owners-deleted-001"
  }
}
```

Validasi:

- trash table frontend bisa memuat data terhapus;
- frontend dapat membedakan data aktif dan data terhapus.

### 6.3 GET `/imports/all`

Request:

```http
GET /api/v1/imports/all?status=VALIDATED&profile=OWNER_OUTLET
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "items": [
      {
        "id": 35,
        "profile": "OWNER_OUTLET",
        "status": "VALIDATED",
        "file_name": "owner-juli-2026.xlsx",
        "created_at": "2026-07-29T09:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 1,
      "total": 1
    }
  },
  "meta": {
    "request_id": "req-imports-all-001"
  }
}
```

Validasi:

- histori import dapat diambil penuh;
- cocok untuk modal riwayat import atau audit upload.

### 6.4 GET `/supervisors`

Request:

```http
GET /api/v1/supervisors?status=ACTIVE
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "items": [
      {
        "id": 3,
        "code": "SPV-001",
        "name": "Supervisor Area 1",
        "email": "spv1@piposmart.test",
        "phone": "628111111111",
        "role": "SUPERVISOR",
        "status": "ACTIVE",
        "must_change_password": false
      }
    ],
    "total": 1
  },
  "meta": {
    "request_id": "req-supervisors-list-001"
  }
}
```

### 6.5 POST `/supervisors`

Request:

```http
POST /api/v1/supervisors
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "SPV-BDG-002",
  "name": "Supervisor Bandung 2",
  "email": "spv.bdg2@piposmart.test",
  "phone": "081234567890",
  "password": "TempPass123"
}
```

Contoh response:

```json
{
  "data": {
    "user": {
      "id": 12,
      "code": "SPV-BDG-002",
      "name": "Supervisor Bandung 2",
      "email": "spv.bdg2@piposmart.test",
      "phone": "081234567890",
      "role": "SUPERVISOR",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "TempPass123"
  },
  "meta": {
    "request_id": "req-create-supervisor-001"
  }
}
```

Validasi:

- hanya Admin yang boleh akses;
- user baru aktif dan wajib ganti password pada login awal berikutnya.

### 6.6 POST `/admins`

Request:

```http
POST /api/v1/admins
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "ADM-OPS-002",
  "name": "Admin Operasional 2",
  "email": "admin.ops2@piposmart.test",
  "phone": "081200000002",
  "password": "TempAdmin123"
}
```

Contoh response:

```json
{
  "data": {
    "user": {
      "id": 13,
      "code": "ADM-OPS-002",
      "name": "Admin Operasional 2",
      "email": "admin.ops2@piposmart.test",
      "phone": "081200000002",
      "role": "ADMIN",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "TempAdmin123"
  },
  "meta": {
    "request_id": "req-create-admin-001"
  }
}
```

### 6.7 POST `/supervisors/{id}/reset-password`

Request:

```http
POST /api/v1/supervisors/12/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "new_password": "ResetPass123"
}
```

Contoh response:

```json
{
  "data": {
    "user": {
      "id": 12,
      "code": "SPV-BDG-002",
      "name": "Supervisor Bandung 2",
      "email": "spv.bdg2@piposmart.test",
      "role": "SUPERVISOR",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "ResetPass123"
  },
  "meta": {
    "request_id": "req-reset-supervisor-001"
  }
}
```

### 6.8 POST `/admins/{id}/reset-password`

Request:

```http
POST /api/v1/admins/13/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "new_password": "ResetAdmin123"
}
```

Contoh response:

```json
{
  "data": {
    "user": {
      "id": 13,
      "code": "ADM-OPS-002",
      "name": "Admin Operasional 2",
      "email": "admin.ops2@piposmart.test",
      "role": "ADMIN",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "ResetAdmin123"
  },
  "meta": {
    "request_id": "req-reset-admin-001"
  }
}
```

### 6.9 POST `/sales/{id}/reset-password`

Request:

```http
POST /api/v1/sales/25/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "new_password": "ResetSales123"
}
```

Contoh response:

```json
{
  "data": {
    "user": {
      "id": 25,
      "code": "SLS-025",
      "name": "Sales 25",
      "email": "sales25@piposmart.test",
      "role": "SALES",
      "status": "ACTIVE",
      "must_change_password": true
    },
    "temporary_password": "ResetSales123"
  },
  "meta": {
    "request_id": "req-reset-sales-001"
  }
}
```

## 7. Error Handler, Contoh Error, dan Solusinya

### 7.1 Token Tidak Dikirim

Request:

```http
GET /api/v1/owners/all
Accept: application/json
```

Response:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "req-auth-001"
  }
}
```

Solusi:

- kirim `Authorization: Bearer {access_token}`.

Pencegahan frontend:

- pakai auth interceptor global;
- redirect ke login bila token kosong.

### 7.2 Token Tidak Valid / Expired

Request:

```http
GET /api/v1/supervisors
Authorization: Bearer token-salah-atau-expired
Accept: application/json
```

Response:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses tidak valid",
    "request_id": "req-auth-002"
  }
}
```

Solusi:

- lakukan refresh token;
- bila gagal, login ulang.

### 7.3 Supervisor Mencoba Membuat Admin

Request:

```http
POST /api/v1/admins
Authorization: Bearer {supervisor_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "ADM-OPS-003",
  "name": "Admin Tidak Sah",
  "email": "admin.tidaksah@piposmart.test",
  "phone": "081200000003",
  "password": "TempAdmin123"
}
```

Response:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "req-authz-001"
  }
}
```

Solusi:

- gunakan akun Admin.

Pencegahan frontend:

- sembunyikan tombol create Admin/create Supervisor/reset Admin/reset Supervisor untuk non-Admin.

### 7.4 Payload Tidak Valid

Request:

```http
POST /api/v1/supervisors
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "SPV-BDG-003",
  "name": "",
  "email": "bukan-email-valid",
  "phone": "08123"
}
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "field wajib kosong atau format email tidak valid"
    },
    "request_id": "req-validation-001"
  }
}
```

Solusi:

- isi `name`;
- gunakan email valid.

Pencegahan frontend:

- validasi field wajib dan format email sebelum submit.

### 7.5 Password Terlalu Lemah

Request:

```http
POST /api/v1/admins
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "ADM-OPS-004",
  "name": "Admin Lemah Password",
  "email": "admin.weak@piposmart.test",
  "phone": "081200000004",
  "password": "123"
}
```

Response:

```json
{
  "error": {
    "code": "WEAK_PASSWORD",
    "message": "password minimal 8 karakter",
    "request_id": "req-validation-002"
  }
}
```

Solusi:

- gunakan password minimal 8 karakter.

### 7.6 ID Path Tidak Valid

Request:

```http
POST /api/v1/supervisors/abc/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "ID tidak valid",
    "request_id": "req-validation-003"
  }
}
```

Solusi:

- selalu kirim path ID berupa integer positif.

### 7.7 Role Salah / Data Tidak Ada

Request:

```http
POST /api/v1/supervisors/25/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "new_password": "ResetPass123"
}
```

Response:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan",
    "request_id": "req-notfound-001"
  }
}
```

Penyebab:

- ID tidak ada;
- atau ID tersebut bukan user role `SUPERVISOR`.

### 7.8 Email Duplikat

Request:

```http
POST /api/v1/supervisors
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "SPV-BDG-004",
  "name": "Supervisor Duplikat Email",
  "email": "spv.bdg2@piposmart.test",
  "phone": "081234567891",
  "password": "TempPass123"
}
```

Response:

```json
{
  "error": {
    "code": "EMAIL_ALREADY_USED",
    "message": "email sudah digunakan",
    "request_id": "req-conflict-001"
  }
}
```

Solusi:

- gunakan email unik.

### 7.9 Kode User Duplikat

Request:

```http
POST /api/v1/admins
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "ADM-OPS-002",
  "name": "Admin Duplikat Kode",
  "email": "admin.kode.dup@piposmart.test",
  "phone": "081200000099",
  "password": "TempAdmin123"
}
```

Response:

```json
{
  "error": {
    "code": "CODE_ALREADY_USED",
    "message": "kode sudah digunakan",
    "request_id": "req-conflict-002"
  }
}
```

Solusi:

- gunakan kode unik.

### 7.10 Route Tidak Ditemukan

Request:

```http
GET /api/v1/admin
Authorization: Bearer {access_token}
Accept: application/json
```

Response:

```json
{
  "error": {
    "code": "ROUTE_NOT_FOUND",
    "message": "Route tidak ditemukan",
    "request_id": "req-route-001"
  }
}
```

Solusi:

- gunakan endpoint dari OpenAPI, jangan hardcode route typo.

### 7.11 Import Batch Status Tidak Sesuai

Request:

```http
POST /api/v1/imports/35/commit
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Kondisi pemicu:

- frontend memanggil commit saat status batch belum `VALIDATED`.

Response:

```json
{
  "error": {
    "code": "INVALID_BATCH_STATUS",
    "message": "importing: batch is not in a state that allows this action",
    "details": {
      "root_cause": "Frontend memanggil aksi import pada status batch yang belum sesuai alur backend.",
      "solution": "Poll GET /imports/{id} dan hanya aktifkan aksi yang sesuai dengan status batch saat ini.",
      "frontend_prevent": "Gunakan status batch sebagai source of truth utama. Tombol commit hanya aktif saat status VALIDATED.",
      "hint": "poll GET /imports/{id} until status VALIDATED before first commit"
    },
    "request_id": "req-import-001"
  }
}
```

Solusi:

- ikuti state batch:
  - upload
  - validating
  - validated
  - committing
  - committed

## 8. Checklist Validasi Frontend

- semua request auth mengirim Bearer token;
- route `/all` tidak dipakai sembarangan untuk tabel yang seharusnya tetap pagination;
- trash table memakai `/all-deleted` atau `/trash` sesuai kebutuhan halaman;
- tombol create Admin/Supervisor hanya tampil untuk Admin;
- tombol reset password Admin/Supervisor hanya tampil untuk Admin;
- validasi form create user:
  - `name` wajib
  - `email` valid
  - `password` minimal 8 karakter bila diisi manual
- tangani `409 CONFLICT` untuk email/kode duplikat;
- tampilkan `request_id` ketika error agar mudah tracing;
- modul import harus membaca status batch dari backend sebelum enable action.

## 9. Kesimpulan

Sprint 14f sekarang sudah memiliki dokumentasi testing yang mencakup:

- contoh request sukses;
- contoh response sukses;
- contoh request yang sengaja dibuat error;
- contoh error response;
- solusi per error;
- saran pencegahan di frontend;
- bukti verifikasi build dan test package backend.
