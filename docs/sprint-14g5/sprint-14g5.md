Sprint: 14g5 - Importing, Executive Analytics, Advanced Comparison, dan Export Center
Periode: 29 Juli 2026
Status: AMBER

Sprint Goal:
- Menyediakan executive analytics lintas modul.
- Menyediakan comparison builder yang lebih fleksibel.
- Menyediakan kontrak backend analytics Sprint 14g5 yang bisa langsung dipakai frontend.

Committed Deliverables:
- 20 endpoint diagram Sprint 14g5
- Custom comparison engine berbasis baseline period dan compare-series sederhana
- Executive dashboard analytics
- Export job center

Completed:
- 20 diagram backend aktif di catalog analytics:
  - imports: `batches-per-profile`, `success-vs-failed`, `invalid-rows-distribution`, `validation-error-by-profile`, `duplicate-detection-rate`, `import-duration-trend`, `batch-status-funnel`, `uploader-activity`, `file-history-usage`
  - executive: `end-to-end-funnel`, `revenue-closing-active-subscription-board`, `monthly-operating-review-board`, `north-star-kpi-trend`, `data-quality-score-by-module`, `forecast-summary-board`
  - custom: `multi-series-trend`, `metric-comparison-board`, `region-comparison-board`, `comparison-impact-summary`
  - subscriptions: `cohort-retention`
- Update analytics registry/catalog untuk seluruh diagram Sprint 14g5
- Service dispatch backend untuk seluruh diagram Sprint 14g5
- Baseline verification:
  - `go build ./...` ✅
  - `go vet ./internal/analytics/...` ✅
  - `go test ./internal/analytics/...` logic test ✅, namun Windows cleanup menutup dengan `unlinkat ... Access is denied`

Not Completed / Carry Over:
- Item: Export job center analytics (`xlsx`, `pdf`, `png`)
- Penyebab: pondasi renderer chart + format export spesifik analytics belum tersedia dan jika dipaksakan berisiko menurunkan stabilitas sprint ini
- Estimasi ulang: Sprint analytics/export berikutnya

- Item: persisted usage log untuk viewer/download file import
- Penyebab: backend saat ini belum menyimpan event open/download file histori
- Estimasi ulang: perlu event/audit log tambahan pada modul importing

Demo Evidence:
- Endpoint/Swagger:
  - `GET /api/v1/analytics/catalog`
  - `GET /api/v1/analytics/catalog/executive`
  - `POST /api/v1/analytics/executive/north-star-kpi-trend/query`
  - `POST /api/v1/analytics/custom/region-comparison-board/query`
  - `POST /api/v1/analytics/subscriptions/cohort-retention/query`
- Skenario request contoh:

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": true,
    "mode": "previous_period"
  },
  "metrics": [
    "owner_count",
    "confirmed_closing_count",
    "topup_revenue"
  ]
}
```

Quality:
- Unit/integration test: analytics unit test lulus; Windows temporary binary cleanup masih memunculkan `Access is denied` setelah package selesai diuji
- Migration status: tidak ada migration baru pada Sprint 14g5
- Docker build: tidak diubah pada Sprint 14g5
- Defect terbuka:
  - export center belum aktif
  - `file-history-usage` masih placeholder contract

Impediments:
- Observability untuk viewer/download file import belum ada persisted event source
- Export analytics butuh fondasi renderer/file packaging tambahan

Risiko Baru:
- Risiko: frontend menganggap `file-history-usage` sudah berbasis data nyata
- Dampak: interpretasi chart bisa menyesatkan
- Mitigasi: response diberi `source_note` jelas bahwa data usage belum persisted
- Owner: Backend

- Risiko: data quality score dianggap angka audit final
- Dampak: user menyimpulkan score sebagai governance resmi
- Mitigasi: dokumentasi dan response menandai score sebagai heuristic operational score
- Owner: Backend + Product

Keputusan yang Dibutuhkan:
- Prioritas implementasi export analytics vs penguatan event log/import observability

Rencana Sprint Berikutnya:
- Menyelesaikan export analytics async
- Menambahkan event usage histori file import
- Menyempurnakan compare-series lintas dimension lain bila dibutuhkan frontend/analyst
