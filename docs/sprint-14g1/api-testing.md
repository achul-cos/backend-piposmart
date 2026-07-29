# API Testing Report - Sprint 14g1

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g1 |
| Area | Analytics API |
| Tanggal testing | 29 Juli 2026 |
| Environment | Local Development |
| Base URL | `http://localhost:8080/api/v1` |
| Status akhir | PASS |

## 2. Ringkasan Hasil

Fokus Sprint 14g1:

- owner analytics;
- outlet analytics;
- lead analytics;
- interaction analytics;
- training analytics;
- peta Indonesia owner dan outlet.

Rekap:

| Jumlah endpoint diuji | PASS | FAIL |
| ---: | ---: | ---: |
| 20 | 20 | 0 |

## 3. Header dan Format Request

### 3.1 Header

```http
Content-Type: application/json
Authorization: Bearer {access_token}
Accept: application/json
```

### 3.2 Payload baseline

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": false
  },
  "filters": {},
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

## 4. Daftar Endpoint yang Diuji

| Endpoint | Method | Status |
| --- | --- | --- |
| `/analytics/owners/growth-trend/query` | POST | PASS |
| `/analytics/owners/ownership-distribution/query` | POST | PASS |
| `/analytics/owners/province-distribution/query` | POST | PASS |
| `/analytics/owners/city-top10/query` | POST | PASS |
| `/analytics/owners/soft-delete-trend/query` | POST | PASS |
| `/analytics/outlets/growth-trend/query` | POST | PASS |
| `/analytics/outlets/outlet-per-owner-histogram/query` | POST | PASS |
| `/analytics/outlets/subscription-status-recap/query` | POST | PASS |
| `/analytics/outlets/not-subscribe-trend/query` | POST | PASS |
| `/analytics/owners/indonesia-distribution-map/query` | POST | PASS |
| `/analytics/outlets/indonesia-distribution-map/query` | POST | PASS |
| `/analytics/leads/funnel/query` | POST | PASS |
| `/analytics/leads/aging-by-stage/query` | POST | PASS |
| `/analytics/leads/assignment-distribution/query` | POST | PASS |
| `/analytics/leads/ownership-transfer-sankey/query` | POST | PASS |
| `/analytics/interactions/volume-trend/query` | POST | PASS |
| `/analytics/interactions/remark-distribution/query` | POST | PASS |
| `/analytics/interactions/follow-up-compliance/query` | POST | PASS |
| `/analytics/interactions/first-response-lag/query` | POST | PASS |
| `/analytics/trainings/scheduled-vs-completed/query` | POST | PASS |

## 5. Contoh Pengujian Request dan Response

### 5.1 Contoh request sukses

```http
POST /api/v1/analytics/owners/growth-trend/query
Authorization: Bearer {admin_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": false
  },
  "filters": {},
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

### 5.2 Contoh response sukses

```json
{
  "data": {
    "series": [
      {
        "key": "owner_count",
        "label": "Owner Baru",
        "points": [
          { "x": "2026-01", "y": 3 },
          { "x": "2026-02", "y": 4 },
          { "x": "2026-03", "y": 4 }
        ]
      }
    ],
    "table": [
      { "period": "2026-01", "owner_count": 3 },
      { "period": "2026-02", "owner_count": 4 },
      { "period": "2026-03", "owner_count": 4 }
    ],
    "insight": {
      "summary": "Tren Pertumbuhan Owner pada periode terpilih bernilai 47.00."
    }
  },
  "meta": {
    "request_id": "example-request-id"
  }
}
```

## 6. Verifikasi dan Validasi

- seluruh endpoint owner, outlet, lead, interaction, dan training merespons `200 OK`;
- peta Indonesia owner dan outlet berhasil di-query;
- endpoint ownership transfer sankey berhasil merespons;
- validasi role berhasil:
  - Admin melihat total owner baru `47`;
  - Sales melihat total owner baru `14`;
  - artinya visibility analytics tetap sesuai scope role.

## 7. Error Handler yang Diverifikasi

### 7.1 Missing token

Request:

```http
GET /api/v1/analytics/catalog
```

Response:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "58c7a302d05a9b15daea677fb2213c46"
  }
}
```

Solusi frontend:

- selalu kirim JWT pada request analytics.

### 7.2 Time filter tidak valid

Request salah:

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-07",
    "month_to": "2026-01",
    "granularity": "month"
  },
  "comparison": {
    "enabled": false
  }
}
```

Response:

```json
{
  "error": {
    "code": "ANALYTICS_QUERY_ERROR",
    "message": "time filter tidak valid: month_from harus lebih kecil dari atau sama dengan month_to",
    "request_id": "9625908bb74782fbf6c9277faba2cf8b"
  }
}
```

Solusi frontend:

- validasi `from <= to` sebelum submit.

## 8. Kesimpulan

Analytics Sprint 14g1 sudah lulus pengujian. Endpoint customer ops dan geo analytics stabil dan siap dikonsumsi frontend.
