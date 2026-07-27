Sprint: 14b - Trash Listing, Wallet Semua Owner, dan Outlet Global
Periode: 27 Juli 2026
Status: GREEN

Sprint Goal:
- Frontend dapat mengambil data trash/unscoped untuk entitas soft delete utama.
- Modul wallet dapat menampilkan semua owner beserta saldo default 0.
- Frontend memiliki endpoint outlet global dan detail outlet yang siap dipakai.

Committed Deliverables:
- Route `trash` dan `unscoped` untuk owner/outlet.
- Route `trash` dan `unscoped` untuk catalog dan closing.
- Perubahan wallet list agar menampilkan semua owner.
- Route global outlet dan detail outlet global.
- Filter `outlet_id` pada subscription.
- Dokumentasi Sprint 14b.

Completed:
- Menambahkan list soft-delete scope untuk owner, nested outlet, catalog (packages/plans/promotions), dan closing.
- Menambahkan route global outlet: `/api/v1/outlets`, `/trash`, `/unscoped`, dan detail `/api/v1/outlets/{outlet_id}`.
- Menambahkan ringkasan owner, wallet owner, dan subscription summary pada response outlet global/detail.
- Mengubah wallet read model agar berbasis seluruh owner; owner tanpa top-up sekarang tetap muncul dengan saldo dan ledger `0.00`.
- Menambahkan query `outlet_id` pada `/api/v1/subscriptions` untuk kebutuhan riwayat langganan outlet.
- Memperbarui README ringkas route dan membuat dokumentasi `docs/sprint-14b/README.md`.

Not Completed / Carry Over:
- Manual smoke test HTTP belum dijalankan pada Sprint 14b ini; verifikasi saat ini berbasis build/test otomatis.

Demo Evidence:
- Endpoint/Swagger: route backend sudah terdaftar di handler modul terkait.
- Dokumen frontend handoff: `docs/sprint-14b/README.md`.
- Build/Test: `go build ./...` PASS, `go test ./...` PASS.

Quality:
- Unit/integration test: PASS.
- Migration status: tidak ada migration baru pada Sprint 14b.
- Docker build: tidak diubah pada Sprint 14b.
- Defect terbuka: tidak ada defect compile/test yang tersisa.

Impediments:
- Tidak ada blocker fungsional. Ada satu gangguan sementara `go test` pada temp-file Windows, namun rerun berikutnya PASS dan terkonfirmasi bukan masalah kode.

Risiko Baru:
- Risiko: endpoint global outlet menggunakan agregasi/subquery SQL untuk wallet dan subscription summary.
- Dampak: pada dataset sangat besar perlu dipantau performanya.
- Mitigasi: sudah dibatasi pagination, dan bisa dioptimasi lebih lanjut pada sprint hardening/performance.
- Owner: Backend Engineer.

Keputusan yang Dibutuhkan:
- Tidak ada.

Rencana Sprint Berikutnya:
- Lanjut ke sprint selanjutnya dengan dasar endpoint frontend yang lebih lengkap.
- Bila dibutuhkan, lanjutkan update OpenAPI/Swagger detail untuk route Sprint 14b dan tambahkan manual smoke test HTTP.
