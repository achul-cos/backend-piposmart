# API Testing Report — Sprint 16b

## Informasi Testing

| Item | Nilai |
| --- | --- |
| Sprint | 16b |
| Tanggal Testing | 3 Agustus 2026 |
| Environment | Local Development |
| Fokus | subscription upgrade, effective start date, prorata sisa hari, dan filter histori order upgrade |

## Summary

| Area | Status | Catatan |
| --- | --- | --- |
| Build backend | PASS | `go build .` berhasil |
| Test unit subscription | PASS | test prorata dan validasi tanggal lolos |
| Validasi OpenAPI | Pending verifikasi akhir | dilakukan setelah patch route/docs |
| Known environment note | WARN | Windows kadang gagal cleanup file `.test.exe` walau hasil test `ok` |

## Command Verifikasi

```bash
go test ./internal/subscription
go build .
npx -y @apidevtools/swagger-cli validate internal/platform/httpserver/openapi.yaml
```

## Skenario Utama

### 1. Membuat order subscription biasa dengan tanggal mulai manual

Request:

```http
POST /api/v1/owners/12/subscription-orders
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "plan_id": 3,
  "outlet_id": 21,
  "idempotency_key": "sub-order-owner-12-20260803-001",
  "external_reference": "ADMIN-SUB-20260803-001",
  "purchased_at": "2026-08-01T19:30:00Z",
  "subscription_start_date": "2026-08-01",
  "note": "Backfill pembelian owner di luar jam kerja admin."
}
```

Ekspektasi:

- order berhasil dibuat;
- `subscription_start_date` mengikuti tanggal bisnis yang diinput admin, bukan tanggal input hari ini.

### 2. Upgrade subscription aktif secara prorata

Asumsi data:

- subscription aktif Basic berakhir pada `2026-09-01`
- effective start upgrade = `2026-08-29`
- sisa hari = `3`
- harga plan target Pro 1 bulan = `300000`

Request:

```http
POST /api/v1/subscriptions/15/upgrades
Authorization: Bearer {access_token}
Content-Type: application/json
```

```json
{
  "plan_id": 7,
  "closing_id": 22,
  "idempotency_key": "sub-upgrade-15-20260803-001",
  "external_reference": "ADMIN-UPGRADE-20260803-001",
  "purchased_at": "2026-08-03T10:30:00Z",
  "effective_start_date": "2026-08-29",
  "note": "Upgrade dari Basic ke Pro untuk sisa 3 hari."
}
```

Ekspektasi:

- backend memotong subscription lama pada `2026-08-29`;
- backend membuat subscription baru sampai akhir masa aktif lama;
- amount upgrade dihitung prorata dari plan target;
- order upgrade tercatat dengan `order_type = UPGRADE`;
- jika `closing_id` dikirim, order langsung masuk flow reconciliation sales.

### 3. Filter histori order upgrade

Request:

```http
GET /api/v1/subscription-orders?order_type=UPGRADE&source_subscription_id=15
Authorization: Bearer {access_token}
```

Ekspektasi:

- hanya order upgrade dari subscription sumber tersebut yang tampil.

## Error Handler yang Diverifikasi

### A. Subscription sumber tidak aktif

Request:

```http
POST /api/v1/subscriptions/99/upgrades
```

```json
{
  "plan_id": 7,
  "idempotency_key": "sub-upgrade-99-001"
}
```

Response:

```json
{
  "error": {
    "code": "SUBSCRIPTION_NOT_ACTIVE",
    "message": "subscription tidak aktif untuk di-upgrade"
  }
}
```

Solusi frontend:

- tampilkan tombol upgrade hanya untuk subscription berstatus `ACTIVE`.

### B. Effective start date lebih besar dari tanggal pembelian

Request:

```json
{
  "plan_id": 7,
  "idempotency_key": "sub-upgrade-15-002",
  "purchased_at": "2026-08-03T10:30:00Z",
  "effective_start_date": "2026-08-04"
}
```

Response:

```json
{
  "error": {
    "code": "UPGRADE_NOT_ALLOWED",
    "message": "upgrade paket tidak valid"
  }
}
```

Solusi frontend:

- pastikan `effective_start_date` tidak melebihi tanggal bisnis `purchased_at`.

### C. Plan target bukan upgrade

Kasus:

- subscription saat ini ada di package level lebih tinggi atau sama
- admin mencoba memilih plan target yang levelnya tidak naik

Response:

```json
{
  "error": {
    "code": "UPGRADE_NOT_ALLOWED",
    "message": "upgrade paket tidak valid"
  }
}
```

Solusi frontend:

- filter plan target hanya untuk package dengan level di atas package subscription aktif saat ini.

### D. Closing sales tidak cocok dengan target upgrade

Kasus:

- `closing_id` dikirim
- owner berbeda, outlet berbeda, atau plan closing tidak sama dengan plan target upgrade

Response:

```json
{
  "error": {
    "code": "CLOSING_MISMATCH",
    "message": "closing tidak sesuai dengan order atau owner"
  }
}
```

Solusi frontend:

- hanya tawarkan daftar closing owner yang sama;
- bila memungkinkan, batasi closing yang plan-nya sama dengan plan target upgrade.

### E. Idempotency tidak dikirim

Request:

```json
{
  "plan_id": 7
}
```

Response:

```json
{
  "error": {
    "code": "IDEMPOTENCY_REQUIRED",
    "message": "idempotency_key atau external_reference wajib dikirim"
  }
}
```

Solusi frontend:

- selalu kirim salah satu:
  - `idempotency_key`
  - `external_reference`

## Kesimpulan

Sprint 16b berhasil menambahkan fondasi backend untuk upgrade paket subscription:

- aman secara histori;
- mendukung backfill tanggal bisnis;
- menghitung nilai upgrade berdasarkan sisa hari;
- tidak mengganggu flow subscription order biasa.
