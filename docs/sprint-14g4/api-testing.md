# API Testing Report - Sprint 14g4

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g4 |
| Area | Analytics API |
| Tanggal testing | 29 Juli 2026 |
| Environment | Local Development |
| Base URL | `http://localhost:8080/api/v1` |
| Status akhir | PASS |

## 2. Ringkasan Hasil

Fokus Sprint 14g4:

- partner analytics;
- commission analytics;
- governance dan audit analytics.

Rekap:

| Jumlah endpoint diuji | PASS | FAIL |
| ---: | ---: | ---: |
| 20 | 20 | 0 |

## 3. Daftar Endpoint yang Diuji

| Endpoint | Method | Status |
| --- | --- | --- |
| `/analytics/partners/partner-growth-trend/query` | POST | PASS |
| `/analytics/partners/partner-type-distribution/query` | POST | PASS |
| `/analytics/partners/referral-count-per-partner/query` | POST | PASS |
| `/analytics/partners/referral-conversion-per-partner/query` | POST | PASS |
| `/analytics/partners/partner-pic-workload/query` | POST | PASS |
| `/analytics/partners/call-mitra-frequency/query` | POST | PASS |
| `/analytics/partners/partner-inactivity-aging/query` | POST | PASS |
| `/analytics/partners/partner-region-distribution/query` | POST | PASS |
| `/analytics/commissions/commission-earned-trend/query` | POST | PASS |
| `/analytics/commissions/paid-vs-unpaid/query` | POST | PASS |
| `/analytics/commissions/commission-aging/query` | POST | PASS |
| `/analytics/commissions/commission-by-partner-type/query` | POST | PASS |
| `/analytics/commissions/commission-by-package/query` | POST | PASS |
| `/analytics/commissions/payout-waterfall/query` | POST | PASS |
| `/analytics/commissions/rule-history-timeline/query` | POST | PASS |
| `/analytics/commissions/snapshot-vs-current/query` | POST | PASS |
| `/analytics/audit/log-volume-by-module/query` | POST | PASS |
| `/analytics/audit/actor-activity-chart/query` | POST | PASS |
| `/analytics/audit/restore-vs-delete-trend/query` | POST | PASS |
| `/analytics/audit/backend-error-code-frequency/query` | POST | PASS |

## 4. Contoh Pengujian Request dan Response

### 4.1 Contoh request sukses

```http
POST /api/v1/analytics/partners/partner-inactivity-aging/query
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
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

### 4.2 Contoh response sukses

```json
{
  "data": {
    "series": [
      {
        "key": "partner_count",
        "label": "Partner",
        "points": [
          { "x": "0-7", "y": 2 },
          { "x": "8-30", "y": 0 },
          { "x": "180+", "y": 1 }
        ]
      }
    ],
    "table": [
      { "aging_bucket": "0-7", "partner_count": 2 },
      { "aging_bucket": "8-30", "partner_count": 0 },
      { "aging_bucket": "180+", "partner_count": 1 }
    ],
    "extra": {
      "average_inactivity_days": 6890.72
    }
  },
  "meta": {
    "request_id": "example-request-id"
  }
}
```

## 5. Verifikasi dan Validasi

- seluruh endpoint partner, commission, dan audit merespons `200 OK`;
- `partner-inactivity-aging` yang sempat gagal saat smoke test awal sudah stabil;
- histori rule commission dapat dibuka;
- chart audit frequency dan restore/delete merespons sesuai kontrak.

## 6. Bug yang Ditemukan dan Diperbaiki

| Endpoint | Masalah awal | Status akhir |
| --- | --- | --- |
| `partners/partner-inactivity-aging` | hasil waktu MySQL tidak terbaca langsung sebagai `time.Time` | PASS |

## 7. Error Handler yang Diverifikasi

### 7.1 Diagram tidak ditemukan

Request:

```http
POST /api/v1/analytics/partners/not-found/query
Authorization: Bearer {access_token}
Content-Type: application/json
```

Response:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "diagram tidak ditemukan",
    "request_id": "6bb3fb5d7d47d4fccde7c28ca18cc37e"
  }
}
```

Solusi frontend:

- baca diagram partner dari catalog, bukan hardcode manual.

## 8. Kesimpulan

Analytics Sprint 14g4 lulus pengujian. Endpoint partner, commission, dan governance analytics siap dipakai frontend.
