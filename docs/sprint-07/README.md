# API Testing Report - Sprint 07

## Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 07 - Package, Plan, Promotion, dan Benefit |
| Tanggal Testing | 23 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| Testing Tool | Manual smoke test via PowerShell/Postman-compatible request |

## Testing Summary

| Module | Case | Actual Result | Status |
| --- | --- | --- | --- |
| Authentication | Login Admin dummy | `200 OK` | PASS |
| Authentication | Login Sales dummy | `200 OK` | PASS |
| Package | List package master seed | `200 OK` | PASS |
| Plan | Create plan tenor 12 bulan | `201 Created`, `duration_days = 360` | PASS |
| Plan | Harga invalid `12.345` | `400 INVALID_DECIMAL` | PASS |
| Promotion | Create promo FREE | `201 Created` | PASS |
| Eligibility | Set eligible plans | `200 OK` | PASS |
| Recommendation | Eligible promotions | `200 OK`, `recommended_promotion.charge_type = FREE` | PASS |
| RBAC | Sales create package | `403 FORBIDDEN` | PASS |

## Quality Gate

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build .` | PASS |
| `go run . seed master` | PASS |
| `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` | PASS |

## Header

```http
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

## Endpoint Utama

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Package | GET | `/catalog/packages` | List paket. |
| Create Package | POST | `/catalog/packages` | Membuat paket. |
| Detail Package | GET | `/catalog/packages/{package_id}` | Detail paket. |
| Update Package | PATCH | `/catalog/packages/{package_id}` | Update paket. |
| Soft Delete Package | DELETE | `/catalog/packages/{package_id}` | Soft delete paket. |
| Restore Package | PATCH | `/catalog/packages/{package_id}/restore` | Restore paket. |
| Force Delete Package | DELETE | `/catalog/packages/{package_id}/force` | Hard delete paket. |
| List Plan | GET | `/catalog/plans` | List plan/tenor. |
| Create Plan | POST | `/catalog/plans` | Membuat plan. |
| Eligible Promotions | GET | `/catalog/plans/{plan_id}/eligible-promotions` | Promo eligible per plan. |
| List Promotion | GET | `/catalog/promotions` | List promo. |
| Create Promotion | POST | `/catalog/promotions` | Membuat promo. |
| Promotion Benefits | GET/POST | `/catalog/promotions/{promotion_id}/benefits` | List/tambah benefit. |
| Set Eligible Plans | PUT | `/catalog/promotions/{promotion_id}/eligible-plans` | Set plan eligible. |

## Contoh Request

### Create Plan

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

Expected response `201 Created`:

```json
{
  "data": {
    "code": "BUSINESS_12_MONTHS_TEST",
    "tenure_months": 12,
    "duration_days": 360,
    "price": "1698600.00",
    "currency": "IDR"
  }
}
```

### Create Promotion FREE

```json
{
  "code": "FREE_1_MONTH_BUSINESS_12_TEST",
  "name": "Business 12 + 1 Bulan Test",
  "promotion_type": "FREE_DURATION",
  "charge_type": "FREE",
  "additional_charge": "0.00",
  "priority": 10,
  "effective_from": "2026-07-01"
}
```

### Set Eligible Plans

```http
PUT /api/v1/catalog/promotions/{promotion_id}/eligible-plans
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "plan_ids": [2]
}
```

### Eligible Promotions

```http
GET /api/v1/catalog/plans/2/eligible-promotions?as_of=2026-07-23
Authorization: Bearer {access_token}
Accept: application/json
```

Expected response:

```json
{
  "data": {
    "recommended_promotion": {
      "charge_type": "FREE"
    },
    "items": [
      {
        "charge_type": "FREE"
      },
      {
        "charge_type": "PAID"
      }
    ]
  }
}
```

## Contoh Error

### Harga Decimal Tidak Valid

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

### Sales Membuat Catalog

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

Sprint 07 catalog API menyediakan pengelolaan paket, plan, promo, benefit, dan
eligibility. Perhitungan durasi memakai 30 hari per bulan dan rekomendasi promo
memprioritaskan promo gratis.
