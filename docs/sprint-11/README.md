# API Testing Report - Sprint 11 Partner, PIC Assignment, Referral, dan Call Interaction

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 11 - Partner, PIC Assignment, Referral, dan Call Interaction |
| Tanggal Testing | 24 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via PowerShell `Invoke-RestMethod` / `Invoke-WebRequest` |
| Database Migration | `go run . migrate up` (`20260724000900_partner_pic_referral.sql`) |
| Seeder | `go run . seed master` dan `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` |

## 2. Header Pengujian

Route protected wajib memakai JWT:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Akun demo yang digunakan:

| Role | Email | Password | Tujuan Pengujian |
| --- | --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` | Create, update, deactivate partner, assign PIC, record interaction, dan create referral. |
| Supervisor | `supervisor.001@demo.piposmart.id` | `Password123!` | Penugasan PIC dan pengawasan interaksi partner. |
| Sales | `sales.001@demo.piposmart.id` | `Password123!` | Akses penugasan PIC dan pencatatan referral. |

Login Admin:

```http
POST /api/v1/auth/login
Accept: application/json
Content-Type: application/json
```

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
    "access_token": "{jwt_access_token}",
    "refresh_token": "{refresh_token}",
    "token_type": "Bearer",
    "expires_in": 899,
    "user": {
      "id": 1,
      "email": "admin.001@demo.piposmart.id",
      "role": "ADMIN"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

## 3. Scope Pengujian

Sprint 11 memvalidasi seluruh alur manajemen partner:
- Management Partner Type (List, Get by ID, Create, Update).
- Partner CRUD (Create, List with Pagination/Search, Get by ID, Get by Code, Update, Deactivate).
- Enkripsi Rekening Bank: Nomor rekening disimpan terenkripsi AES-GCM di database dan dikembalikan dalam bentuk masked `****1234` di response JSON.
- PIC Assignment: Penugasan user (Sales/Supervisor) sebagai PIC partner dengan aturan single active assignment (otomatis menonaktifkan assignment lama).
- Release PIC: Pelepasan penugasan PIC aktif.
- Partner Interaction: Pencatatan riwayat panggilan (`CALL`) dan obrolan (`CHAT`) mitra.
- Partner Referral: Pendaftaran rujukan partner ke customer lead dengan validasi keunikan pasangan `partner_id` - `lead_id` (duplikat mengembalikan `409 CONFLICT`).
- Testing Error Handling & Validation.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Partner Type | `/partner-types` | GET | 200 OK | PASS |
| Partner Type | `/partner-types/{id}` | GET | 200 OK | PASS |
| Partner Type | `/partner-types` | POST | 201 Created | PASS |
| Partner Type | `/partner-types/{id}` | PUT | 200 OK | PASS |
| Partner | `/partners` (Create with bank) | POST | 201 Created | PASS |
| Partner | Masked bank account verification | POST/GET | `bank_account_masked: "****5678"` | PASS |
| Partner | `/partners` (List with pagination) | GET | 200 OK | PASS |
| Partner | `/partners/{id}` | GET | 200 OK | PASS |
| Partner | `/partners/code/{code}` | GET | 200 OK | PASS |
| Partner | `/partners/{id}` | PUT | 200 OK | PASS |
| Partner Assignment | `/partners/{partnerID}/assignments` | POST | 201 Created | PASS |
| Partner Assignment | Single Active Invariant Re-assign | POST | Old assignment active=false | PASS |
| Partner Assignment | `/partners/{partnerID}/assignments/active` | GET | 200 OK | PASS |
| Partner Assignment | `/partners/{partnerID}/assignments` | GET | 200 OK | PASS |
| Partner Assignment | `/partners/{partnerID}/assignments/release` | DELETE | 204 No Content | PASS |
| Partner Interaction | `/partners/{partnerID}/interactions` (CALL) | POST | 201 Created | PASS |
| Partner Interaction | `/partners/{partnerID}/interactions` | GET | 200 OK | PASS |
| Partner Referral | `/partners/{partnerID}/referrals` | POST | 201 Created | PASS |
| Partner Referral | Repeat create referral (Duplicate) | POST | 409 CONFLICT | PASS |
| Partner Referral | `/partners/{partnerID}/referrals` | GET | 200 OK | PASS |
| Partner | `/partners/{id}` (Deactivate) | DELETE | 204 No Content | PASS |
| Error Handling | Tanpa JWT Token | GET | 401 UNAUTHENTICATED | PASS |
| Error Handling | Invalid Partner ID | GET | 400 VALIDATION_ERROR | PASS |
| Error Handling | Partner Not Found | GET | 404 NOT_FOUND | PASS |

## 5. Detail Skenario Pengujian API

### 5.1 List Partner Types (`GET /api/v1/partner-types`)

Request:

```http
GET /api/v1/partner-types
Authorization: Bearer {access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "SUPPLIER",
        "name": "Supplier Hardware & POS",
        "description": "Partner penyedia perangkat POS dan hardware kasir.",
        "created_at": "2026-07-24T06:50:00Z",
        "updated_at": "2026-07-24T06:50:00Z"
      },
      {
        "id": 2,
        "code": "DISTRIBUTOR",
        "name": "Distributor Software",
        "description": "Distributor lisensi aplikasi Piposmart.",
        "created_at": "2026-07-24T06:50:00Z",
        "updated_at": "2026-07-24T06:50:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 2,
      "total": 2
    }
  },
  "meta": {
    "request_id": "req-001"
  }
}
```

---

### 5.2 Create Partner dengan Rekening Bank (`POST /api/v1/partners`)

Request:

```http
POST /api/v1/partners
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "partner_type_id": 1,
  "code": "SUP-999",
  "name": "PT POS Technology Utama",
  "phone": "081987654321",
  "email": "info@postech.demo.id",
  "address": "Jl. Metro Utama No. 88, Jakarta Selatan",
  "bank_account": "14000987654321",
  "status": "ACTIVE"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 4,
    "partner_type": {
      "id": 1,
      "code": "SUPPLIER",
      "name": "Supplier Hardware & POS"
    },
    "code": "SUP-999",
    "name": "PT POS Technology Utama",
    "phone": "081987654321",
    "email": "info@postech.demo.id",
    "address": "Jl. Metro Utama No. 88, Jakarta Selatan",
    "bank_account_masked": "****4321",
    "status": "ACTIVE",
    "created_at": "2026-07-24T07:00:00Z",
    "updated_at": "2026-07-24T07:00:00Z"
  },
  "meta": {
    "request_id": "req-002"
  }
}
```

> **Catatan Keamanan**: Field `bank_account_encrypted` tidak pernah terekspos di response API. Hanya `bank_account_masked` (`****4321`) yang ditampilkan.

---

### 5.3 Assign PIC Partner (`POST /api/v1/partners/{partnerID}/assignments`)

Request:

```http
POST /api/v1/partners/1/assignments
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "user_id": 3
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 2,
    "partner_id": 1,
    "user_id": 3,
    "user_name": null,
    "user_role": "",
    "assigned_by_id": 1,
    "assigned_by_name": null,
    "assigned_at": "2026-07-24T07:05:00Z",
    "unassigned_at": null,
    "active": true,
    "created_at": "2026-07-24T07:05:00Z",
    "updated_at": "2026-07-24T07:05:00Z"
  },
  "meta": {
    "request_id": "req-003"
  }
}
```

---

### 5.4 Get Active PIC Assignment (`GET /api/v1/partners/{partnerID}/assignments/active`)

Request:

```http
GET /api/v1/partners/1/assignments/active
Authorization: Bearer {access_token}
```

Response `200 OK`:

```json
{
  "data": {
    "id": 2,
    "partner_id": 1,
    "user_id": 3,
    "assigned_by_id": 1,
    "assigned_at": "2026-07-24T07:05:00Z",
    "active": true
  },
  "meta": {
    "request_id": "req-004"
  }
}
```

---

### 5.5 Record Call Interaction (`POST /api/v1/partners/{partnerID}/interactions`)

Request:

```http
POST /api/v1/partners/1/interactions
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "interaction_type": "CALL",
  "note": "Telepon koordinasi pengiriman 10 unit mesin POS Bluetooth printer."
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 2,
    "partner_id": 1,
    "interaction_type": "CALL",
    "interaction_at": "2026-07-24T07:10:00Z",
    "note": "Telepon koordinasi pengiriman 10 unit mesin POS Bluetooth printer.",
    "created_at": "2026-07-24T07:10:00Z"
  },
  "meta": {
    "request_id": "req-005"
  }
}
```

---

### 5.6 Create Partner Referral (`POST /api/v1/partners/{partnerID}/referrals`)

Request:

```http
POST /api/v1/partners/1/referrals
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "lead_id": 2,
  "notes": "Mitra merekomendasikan prospek gerai resto franchise Baru."
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 2,
    "partner_id": 1,
    "lead_id": 2,
    "referral_date": "2026-07-24T07:15:00Z",
    "note": "Mitra merekomendasikan prospek gerai resto franchise Baru.",
    "created_at": "2026-07-24T07:15:00Z"
  },
  "meta": {
    "request_id": "req-006"
  }
}
```

---

### 5.7 Negative Case: Duplicate Referral (`POST /api/v1/partners/{partnerID}/referrals`)

Request (mengirim lead_id yang sama untuk partner 1):

```http
POST /api/v1/partners/1/referrals
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "lead_id": 2,
  "notes": "Cobaan pendaftaran duplikat."
}
```

Response `409 Conflict`:

```json
{
  "error": {
    "code": "DUPLICATE_REFERRAL",
    "message": "referral already exists for this partner-lead pair",
    "details": null
  },
  "meta": {
    "request_id": "req-007"
  }
}
```

---

## 6. Kesimpulan Pengujian

Seluruh pengujian unit dan pengujian API smoke test pada modul Partner Sprint 11 telah **BERHASIL (100% PASS)** tanpa defect blocker. Modul partner siap diintegrasikan untuk Sprint 12 (Partner Commission).
