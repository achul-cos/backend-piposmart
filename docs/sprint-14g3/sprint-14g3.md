Sprint: 14g3 - Wallet, Topup, Subscription, dan Reconciliation Analytics
Periode: 29 Juli 2026
Status: AMBER

Sprint Goal:
- Menyediakan analytics finansial dan subscription yang aman secara definisi bisnis.
- Menegaskan pemisahan omset topup, closing sales, dan subscription health.

Committed Deliverables:
- 20 endpoint diagram Sprint 14g3
- Rule polarity khusus finance/reconciliation
- Insight generator untuk renewal, churn, dan issue queue

Completed:
- Diagram registry Sprint 14g3 sudah ditambahkan ke katalog analytics.
- Query endpoint backend untuk 20 diagram Sprint 14g3 sudah terhubung ke service analytics:
  - wallets/topup-revenue-trend
  - wallets/topup-transaction-count
  - wallets/owner-balance-distribution
  - wallets/topup-used-vs-unused
  - wallets/topup-to-subscribe-lag
  - wallets/zero-vs-nonzero-balance
  - subscriptions/active-subscription-trend
  - subscriptions/activation-vs-expiry-trend
  - subscriptions/renewal-rate
  - subscriptions/expiry-forecast
  - subscriptions/package-mix
  - subscriptions/tenure-mix
  - subscriptions/days-remaining-histogram
  - subscriptions/churn-bucket-trend
  - reconciliation/success-rate
  - reconciliation/issue-by-type
  - reconciliation/issue-aging
  - reconciliation/auto-vs-manual
  - reconciliation/hanging-transaction-trend
  - reconciliation/revenue-vs-closing-period-compare
- Test registry diagram Sprint 14g3 ditambahkan.
- Pemisahan topup revenue vs closing revenue snapshot dipertahankan pada analytics.

Not Completed / Carry Over:
- Formula renewal rate saat ini memakai pendekatan pragmatic: expired lalu renewed dalam 30 hari berikutnya.
- Export diagram dan compare-series lintas entitas belum diaktifkan.

Verification:
- `go build ./...` ✅
- `go vet ./internal/analytics/...` ✅
- `go test ./internal/analytics/...` ✅ secara hasil package test, dengan catatan Windows cleanup temp exe masih memunculkan `Access is denied` setelah test selesai.

Dependencies:
- Snapshot revenue topup berdasarkan `paid_at`
- Snapshot closing berdasarkan data historis closing
- Rule status outlet/subscription tetap mengikuti definisi Sprint 14c
