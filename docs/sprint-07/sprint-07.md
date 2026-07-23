# Sprint 7 - Package, Plan, Promotion, dan Benefit

## Sprint

Sprint 7

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk vertical slice catalog package, plan, promotion,
benefit, eligibility, dan rekomendasi promo. Status belum `GREEN` karena data
harga/promo resmi tetap perlu validasi stakeholder sebelum digunakan produksi.

## Sprint Goal

Admin/Supervisor dapat mengelola paket, harga, tenor, promo, benefit, dan
eligibility promo. Sistem dapat menampilkan promo eligible per plan serta
merekomendasikan promo gratis lebih dulu.

## Completed

- [x] Package catalog API.
- [x] Subscription plan API.
- [x] Promotion API.
- [x] Promotion benefit API.
- [x] Promotion-plan eligibility API.
- [x] Eligible promotion API per plan.
- [x] Rule `duration_days = tenure_months x 30`.
- [x] Money diproses sebagai decimal string, bukan `float64`.
- [x] Promo `FREE` diprioritaskan sebagai `recommended_promotion`.
- [x] Promo `PAID` tetap muncul sebagai opsi, tetapi tidak dipilih otomatis.
- [x] Filter effective date melalui `as_of`.
- [x] Soft delete, restore, force delete, dan bulk delete untuk package, plan,
  dan promotion.
- [x] OpenAPI diperbarui ke `0.7.0-sprint-7`.

## Demo Evidence

Smoke test API lokal:

- `POST /api/v1/auth/login` sebagai Admin: `200`.
- `POST /api/v1/auth/login` sebagai Sales: `200`.
- `GET /api/v1/catalog/packages?limit=5&sort=level_order`: `200`.
- `POST /api/v1/catalog/plans` tenor 12 bulan: `201`, `duration_days = 360`.
- `POST /api/v1/catalog/promotions` promo `FREE`: `201`.
- `PUT /api/v1/catalog/promotions/{promotion_id}/eligible-plans`: `200`.
- `GET /api/v1/catalog/plans/{plan_id}/eligible-promotions?as_of=2026-07-23`:
  `200`, `recommended_promotion.charge_type = FREE`.
- Sales membuat package: `403 FORBIDDEN`.
- Harga decimal invalid `12.345`: `400 INVALID_DECIMAL`.

Seeder evidence:

- `go run . seed master`: PASS.
- `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01`:
  PASS.

## Quality

- `go test ./...`: PASS.
- `go vet ./...`: PASS.
- `go build .`: PASS.
- `git diff --check`: PASS, hanya warning CRLF Windows.

## Impediments

- Sandbox normal sempat menolak read/exec/edit file existing dengan ACL Windows.
  Validasi sementara dijalankan memakai command escalated.
- Data harga/promo resmi masih perlu approval stakeholder.

## Risiko Baru

- Risiko: Harga/promo di master seed masih asumsi awal.
- Dampak: Jika stakeholder mengubah harga/promo, seed perlu disesuaikan.
- Mitigasi: Perlakukan harga, promo, dan benefit sebagai master data yang dapat
  diubah; transaksi Sprint 8 wajib memakai snapshot.
- Owner: Backend Engineer.

## Rencana Sprint Berikutnya

- Sprint 8: Closing dan laporan penjualan.
- Menggunakan package/plan/promo catalog sebagai sumber pilihan closing.
- Menyimpan snapshot package, tenor, harga, dan promo pada transaksi closing.
