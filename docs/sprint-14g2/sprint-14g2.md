Sprint: 14g2 - Sales, Catalog, Closing, Target, dan KPI Analytics
Periode: 29 Juli 2026
Status: AMBER

Sprint Goal:
- Menyediakan analytics yang paling dibutuhkan Supervisor dan Sales.
- Membuat endpoint diagram spesifik untuk catalog, closing, target, dan KPI.

Committed Deliverables:
- 20 endpoint diagram Sprint 14g2
- Comparison engine v2 untuk sales vs sales dan package vs package
- Metadata insight untuk semua diagram Sprint 14g2

Completed:
- Diagram registry Sprint 14g2 sudah terdaftar di katalog analytics.
- Query endpoint backend untuk 20 diagram Sprint 14g2 sudah terhubung ke service analytics:
  - trainings/training-to-closing-conversion
  - catalog/package-popularity
  - catalog/tenure-popularity
  - catalog/package-tenure-heatmap
  - catalog/promo-adoption-rate
  - catalog/additional-charge-adoption
  - catalog/price-history-timeline
  - catalog/promotion-history-timeline
  - closings/closing-trend
  - closings/closing-by-sales
  - closings/closing-by-supervisor
  - closings/closing-by-package
  - closings/closing-by-tenure
  - closings/status-distribution
  - closings/average-ticket-size-trend
  - closings/closing-amount-waterfall
  - targets/target-vs-actual
  - targets/target-burnup
  - kpi/leaderboard
  - kpi/activity-vs-closing-scatter
- Metric historis closing memakai snapshot omzet tanpa unique code.
- Timeline histori catalog membaca audit log perubahan plan dan promotion.
- Test registry diagram Sprint 14g2 ditambahkan.

Not Completed / Carry Over:
- Comparison engine v2 untuk compare series lintas sales/package belum diaktifkan.
- Export endpoint diagram Sprint 14g2 belum dibuat.

Verification:
- `go build ./...` ✅
- `go vet ./internal/analytics/...` ✅
- `go test ./internal/analytics/...` ✅ secara hasil package test, dengan catatan Windows cleanup temp exe masih memunculkan `Access is denied` setelah test selesai.

Dependencies:
- Snapshot closing omzet harus tetap memakai nilai historis
- Histori package/promo harus dapat dibaca analytics

Risiko:
- Risiko: frontend ingin satu chart memuat terlalu banyak series.
- Mitigasi: backend membatasi daftar `metrics` dan `compare_series` yang valid per diagram.
