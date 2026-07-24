# API Testing Report - Sprint 12 Partner Commission

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 12 - Partner Commission |
| Tanggal Testing | 24 Juli 2026 |
| Environment | Local Development, **terisolasi dari Sprint 11c** |
| API Base URL | `http://localhost:8090/api/v1` |
| Database | `piposmart_sprint12` (terpisah dari `test_piposmart` milik Sprint 11c) |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via `curl` |
| Database Migration | `go run . migrate up` (`20260724001000_partner_commission.sql`) |
| Seeder | `go run . seed master` dan `go run . seed demo --preset=minimal --seed=20260724 --as-of=2026-07-24` |

Alasan isolasi: Sprint 11c (large seeder bug fixes) berjalan paralel pada port `8080` dan database `test_piposmart` (nilai default `.env`). Seluruh perintah di atas dijalankan dengan override environment variable (`DB_NAME=piposmart_sprint12`, `APP_PORT=8090`) tanpa mengubah `.env`, sehingga kedua sprint bisa berjalan bersamaan tanpa saling menimpa data.

## 2. Header Pengujian

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Akun demo yang digunakan:

| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` |
| Sales | `sales.001@demo.piposmart.id` | `Password123!` |

## 3. Scope Pengujian

- `PartnerType` CRUD dengan `commission_mode` (`PERCENTAGE`/`FIXED`) dan `commission_value` — sebelumnya kolom ini ada di skema sejak Sprint 1 tapi tidak pernah terhubung ke layer aplikasi.
- Sync komisi dari closing `CONFIRMED` yang terhubung ke referral partner (idempotent).
- Lifecycle commission: `PENDING → APPROVED → PAID`, atau `→ CANCELLED`.
- Role gating: ADMIN/SUPERVISOR untuk sync/approve/cancel, ADMIN saja untuk pay.
- Isolasi antar partner (commission ID milik partner lain tidak bisa diakses).
- Validasi `commission_value` sesuai `commission_mode`.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Partner Type | `/partner-types` (lihat commission fields) | GET | 200 OK | PASS |
| Partner Type | Create dengan `commission_mode=PERCENTAGE`, `value=150.00` (invalid) | POST | 400 INVALID_COMMISSION_VALUE | PASS |
| Partner Type | Create dengan `commission_mode=FIXED`, `value=-500.00` (invalid) | POST | 400 INVALID_COMMISSION_VALUE | PASS |
| Partner Type | Create dengan `commission_mode=BOGUS` | POST | 400 VALIDATION_ERROR | PASS |
| Partner | Create partner tipe `AGENT` (FIXED) | POST | 201 Created | PASS |
| Partner | Get partner by ID (nested `partner_type` commission fields terisi) | GET | 200 OK | PASS |
| Partner Referral | Create referral REF-001 → lead 3 | POST | 201 Created | PASS |
| Partner Commission | Sync commission (PERCENTAGE) | POST | 200 OK, `created:1` | PASS |
| Partner Commission | Sync ulang (idempotent) | POST | 200 OK, `created:0` | PASS |
| Partner Commission | Approve commission | PATCH | 200 OK, status APPROVED | PASS |
| Partner Commission | Pay commission | PATCH | 200 OK, status PAID | PASS |
| Partner Commission | Approve ulang setelah PAID | PATCH | 400 INVALID_STATUS | PASS |
| Partner Commission | Sync commission (FIXED, AGENT) | POST | 200 OK, flat amount | PASS |
| Partner Commission | Cancel commission (PENDING) | PATCH | 200 OK, status CANCELLED | PASS |
| Partner Commission | Cancel ulang setelah CANCELLED | PATCH | 400 INVALID_STATUS | PASS |
| Error Handling | Sales mencoba sync | POST | 403 FORBIDDEN | PASS |
| Error Handling | Sales mencoba pay | PATCH | 403 FORBIDDEN | PASS |
| Error Handling | Get commission via partner ID yang salah | GET | 404 NOT_FOUND | PASS |

## 5. Detail Skenario Pengujian API

### 5.1 List Partner Types dengan Commission Fields (`GET /api/v1/partner-types`)

Response `200 OK` (sebagian):

```json
{
  "data": {
    "items": [
      {
        "id": 3,
        "code": "AGENT",
        "name": "Agent Regional",
        "commission_mode": "FIXED",
        "commission_value": "150000.00",
        "description": "Agen pemasaran tingkat daerah/regional."
      },
      {
        "id": 4,
        "code": "REFERRAL_PARTNER",
        "name": "Referral Community",
        "commission_mode": "PERCENTAGE",
        "commission_value": "3.00",
        "description": "Mitra komunitas perujuk calon pelanggan."
      }
    ],
    "pagination": { "page": 1, "limit": 6, "total": 6 }
  }
}
```

### 5.2 Validasi Commission Value Invalid (`POST /api/v1/partner-types`)

Request (PERCENTAGE di luar rentang 0-100):

```json
{
  "code": "BAD_PCT",
  "name": "Bad Percent",
  "commission_mode": "PERCENTAGE",
  "commission_value": "150.00"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_COMMISSION_VALUE",
    "message": "partner: commission rate must be a decimal between 0 and 100"
  }
}
```

Request (FIXED negatif):

```json
{
  "code": "BAD_FIXED",
  "name": "Bad Fixed",
  "commission_mode": "FIXED",
  "commission_value": "-500.00"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_COMMISSION_VALUE",
    "message": "partner: nilai uang harus decimal valid"
  }
}
```

> **Bug ditemukan saat smoke test**: kedua kasus di atas awalnya mengembalikan `500 INTERNAL_ERROR` karena handler `CreatePartnerType`/`UpdatePartnerType` belum memetakan error validasi commission ke status HTTP yang tepat. Diperbaiki dengan menambah case eksplisit pada error mapping.

### 5.3 Create Partner Tipe AGENT (`POST /api/v1/partners`)

Request:

```json
{
  "partner_type_id": 3,
  "code": "AGT-001",
  "name": "Agen Regional Jakarta",
  "phone": "081234567999",
  "status": "ACTIVE"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 4,
    "partner_type": {
      "id": 3,
      "code": "AGENT",
      "name": "Agent Regional",
      "commission_mode": "FIXED",
      "commission_value": "150000.00"
    },
    "code": "AGT-001",
    "name": "Agen Regional Jakarta",
    "status": "ACTIVE"
  }
}
```

> **Bug ditemukan saat smoke test**: field `partner_type.commission_mode`/`commission_value` bersarang ini awalnya selalu kosong (`""`) karena service hanya menyalin `Code`/`Name` saat merekonstruksi objek `partner_type` di dalam response Partner. Diperbaiki dengan helper `attachPartnerType()`.

### 5.4 Create Referral (`POST /api/v1/partners/{partnerID}/referrals`)

Request (`partnerID=3`, REF-001, tipe `REFERRAL_PARTNER` PERCENTAGE 3.00%):

```json
{
  "lead_id": 3,
  "notes": "Sprint 12 smoke test referral"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 1,
    "partner_id": 3,
    "lead_id": 3,
    "referral_date": "2026-07-24T22:17:22.79Z",
    "note": "Sprint 12 smoke test referral"
  }
}
```

Lead 3 sudah memiliki closing `CONFIRMED` (`DEMO-CLS-000003-PRO_12_MONTHS`, `final_amount=3768703.00`) dari seed demo.

### 5.5 Sync Commission — PERCENTAGE (`POST /api/v1/partners/{partnerID}/commissions/sync`)

Request: `POST /api/v1/partners/3/commissions/sync` (tanpa body)

Response `200 OK`:

```json
{
  "data": {
    "created": 1,
    "items": [
      {
        "id": 1,
        "code": "COM-20260724-000003-957356",
        "partner_id": 3,
        "partner_code": "REF-001",
        "referral_id": 1,
        "closing_id": 3,
        "closing_code": "DEMO-CLS-000003-PRO_12_MONTHS",
        "commission_mode": "PERCENTAGE",
        "commission_value": "3.00",
        "base_amount": "3768703.00",
        "commission_amount": "113061.09",
        "currency": "IDR",
        "status": "PENDING"
      }
    ]
  }
}
```

Verifikasi kalkulasi: `3768703.00 × 3% = 113061.09` — tepat, tanpa floating point error (dihitung dalam cents-based integer arithmetic dengan rounding half-up).

Sync kedua kali (idempotency check) — Response `200 OK`:

```json
{ "data": { "created": 0, "items": [] } }
```

### 5.6 Approve → Pay Commission

`PATCH /api/v1/partners/3/commissions/1/approve` — Response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "status": "APPROVED",
    "approved_by": { "id": 1, "name": "Admin Demo 001" },
    "approved_at": "2026-07-24T22:18:23Z"
  }
}
```

`PATCH /api/v1/partners/3/commissions/1/pay` — Response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "status": "PAID",
    "approved_by": { "id": 1, "name": "Admin Demo 001" },
    "paid_by": { "id": 1, "name": "Admin Demo 001" },
    "paid_at": "2026-07-24T22:18:23Z"
  }
}
```

Percobaan approve ulang setelah `PAID` — Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_STATUS",
    "message": "partner: invalid commission status transition"
  }
}
```

### 5.7 Sync Commission — FIXED, Uncapped (`POST /api/v1/partners/4/commissions/sync`)

Partner `AGT-001` (tipe `AGENT`, FIXED `150000.00`) direferensikan ke lead 2, yang closing-nya (`DEMO-CLS-000002-BUSINESS_12_MONTHS`) memiliki `final_amount=1698702.00` — lebih besar dari nilai komisi flat.

Response `200 OK`:

```json
{
  "data": {
    "created": 1,
    "items": [
      {
        "id": 2,
        "partner_id": 4,
        "partner_code": "AGT-001",
        "commission_mode": "FIXED",
        "commission_value": "150000.00",
        "base_amount": "1698702.00",
        "commission_amount": "150000.00",
        "status": "PENDING"
      }
    ]
  }
}
```

Komisi tetap `150000.00` flat, tidak dipotong maupun dibatasi oleh `base_amount` — sesuai keputusan bisnis bahwa FIXED commission selalu dibayar penuh.

### 5.8 Cancel Commission

`PATCH /api/v1/partners/4/commissions/2/cancel`:

```json
{ "note": "Closing dibatalkan (smoke test)" }
```

Response `200 OK`: status berubah menjadi `CANCELLED`, `note` tersimpan.

Percobaan cancel kedua kali — Response `400 Bad Request`: `INVALID_STATUS` (commission yang sudah `CANCELLED` tidak bisa dibatalkan lagi).

### 5.9 Role Gating (Negative Case)

Login sebagai `sales.001@demo.piposmart.id`, lalu:

`POST /api/v1/partners/3/commissions/sync` — Response `403 Forbidden`:

```json
{ "error": { "code": "FORBIDDEN", "message": "not allowed to perform this action" } }
```

`PATCH /api/v1/partners/4/commissions/2/pay` — Response `403 Forbidden`: sama seperti di atas.

### 5.10 Cross-Partner Isolation (Negative Case)

Commission `id=2` milik partner `4` (AGT-001). Mengaksesnya lewat partner `3` (REF-001):

`GET /api/v1/partners/3/commissions/2` — Response `404 Not Found`:

```json
{ "error": { "code": "NOT_FOUND", "message": "commission not found for this partner" } }
```

## 6. Kesimpulan Pengujian

Seluruh pengujian unit dan smoke test API pada modul Partner Commission Sprint 12 **BERHASIL (100% PASS)** setelah dua bug ditemukan dan diperbaiki selama smoke test (nested partner_type commission fields kosong, dan validasi commission value mengembalikan 500 alih-alih 400). Modul commission siap digunakan; otomatisasi sync saat closing dikonfirmasi (tanpa perlu panggilan manual) dan laporan komisi massal menjadi kandidat Sprint 13.
