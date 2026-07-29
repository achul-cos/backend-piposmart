Sprint: 14g4 - Partner, Commission, Governance, dan Audit Analytics
Periode: 29 Juli 2026
Status: AMBER

Sprint Goal:
- Menyediakan analytics untuk partner dan commission yang aman secara historis.
- Menyediakan dashboard governance untuk audit perubahan master data.

Committed Deliverables:
- 20 endpoint diagram Sprint 14g4
- Dukungan insight khusus governance dan audit
- Timeline historis untuk commission rules dan aktivitas perubahan data

Completed:
- Diagram registry Sprint 14g4 sudah ditambahkan ke katalog analytics.
- Query endpoint backend untuk 20 diagram Sprint 14g4 sudah terhubung ke service analytics:
  - partners/partner-growth-trend
  - partners/partner-type-distribution
  - partners/referral-count-per-partner
  - partners/referral-conversion-per-partner
  - partners/partner-pic-workload
  - partners/call-mitra-frequency
  - partners/partner-inactivity-aging
  - partners/partner-region-distribution
  - commissions/commission-earned-trend
  - commissions/paid-vs-unpaid
  - commissions/commission-aging
  - commissions/commission-by-partner-type
  - commissions/commission-by-package
  - commissions/payout-waterfall
  - commissions/rule-history-timeline
  - commissions/snapshot-vs-current
  - audit/log-volume-by-module
  - audit/actor-activity-chart
  - audit/restore-vs-delete-trend
  - audit/backend-error-code-frequency
- Test registry diagram Sprint 14g4 ditambahkan.
- Historical commission tetap dihormati melalui snapshot vs current comparison.

Not Completed / Carry Over:
- Diagram `commission rule history timeline` saat ini membaca histori perubahan `partner.type` dari audit log, karena audit granular untuk `commission_rules` belum dipersist sepenuhnya.
- Diagram `backend-error-code-frequency` saat ini hanya menghitung failure backend yang memang persisted di database (`import_batches` dan `job_queue`), belum seluruh HTTP error global.
- Export diagram dan compare-series lintas entitas belum diaktifkan.

Verification:
- `go build ./...` ✅
- `go vet ./internal/analytics/...` ✅
- `go test ./internal/analytics/...` ✅ secara hasil package test, dengan catatan Windows cleanup temp exe masih memunculkan `Access is denied` setelah test selesai.

Dependencies:
- Snapshot commission earning historis
- Audit trail package/promotion/commission rule harus bisa dibaca analytics
