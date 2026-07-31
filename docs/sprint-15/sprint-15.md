Sprint: 15 - Import Transaksi, Mitra, dan Data Sales  
Periode: 30 Juli 2026  
Status: GREEN

Sprint Goal:
- Memvalidasi backend Sprint 15 untuk modul importing profil operasional utama.
- Memastikan alur upload → validate → preview rows → commit → audit hasil benar-benar berjalan.
- Menyusun dokumentasi testing API yang dapat dipakai frontend, QA, dan CTO.

Committed Deliverables:
- Validasi runtime profile:
  - `NEW_SUBSCRIBE`
  - `MONTHLY_ACTIVE`
  - `BONUS_MITRA`
  - `SALES_TARGET`
  - `SALES_CALL_CHAT`
- Validasi endpoint histori import:
  - `GET /api/v1/imports`
  - `GET /api/v1/imports/all`
  - `GET /api/v1/imports/{id}`
  - `GET /api/v1/imports/{id}/rows`
  - `GET /api/v1/imports/{id}/rows/all`
  - `GET /api/v1/imports/{id}/file`
  - `GET /api/v1/imports/{id}/file/download`
  - `POST /api/v1/imports/{id}/commit`
- Dokumentasi request/response/error handler Sprint 15.

Completed:
- Melakukan smoke test nyata untuk `MONTHLY_ACTIVE`.
- Melakukan smoke test nyata untuk `NEW_SUBSCRIBE`.
- Melakukan smoke test nyata untuk `BONUS_MITRA`.
- Melakukan smoke test nyata untuk `SALES_TARGET`.
- Melakukan smoke test nyata untuk `SALES_CALL_CHAT`.
- Memverifikasi side-effect hasil commit ke tabel tujuan di database.
- Menyusun dokumentasi:
  - `docs/sprint-15/README.md`
  - `docs/sprint-15/api-testing.md`
  - `docs/sprint-15/sprint-15.md`
- Menyelesaikan temuan runtime saat validasi:
  - deadlock progress update worker import;
  - error mapping import yang belum eksplisit;
  - collision `import_batches.code` saat file sama di-upload pada hari yang sama dengan konteks berbeda;
  - migration `partner_bonus_referral_snapshots` yang belum diterapkan di DB lokal.

Not Completed / Carry Over:
- Item: pembaruan OpenAPI rinci per field khusus Sprint 15.
- Penyebab: fokus turn ini diprioritaskan ke testing runtime, verifikasi side-effect, dan dokumentasi sprint.
- Estimasi ulang: 30-60 menit bila ingin ditulis sampai level tiap error code dan contoh payload lengkap di YAML.

Demo Evidence:
- Endpoint/flow yang terbukti berjalan:
  - upload `MONTHLY_ACTIVE` batch `7`
  - upload `SALES_TARGET` batch `8`
  - upload `SALES_CALL_CHAT` batch `9`
  - upload `BONUS_MITRA` batch `12`
  - upload `NEW_SUBSCRIBE` batch `13`
  - validation failed sample batch `16`
- Dokumentasi:
  - `docs/sprint-15/README.md`
  - `docs/sprint-15/api-testing.md`

Addendum (30 Juli 2026, setelah Sprint 15a): melanjutkan carry-over yang tersisa dari sprint ini —
- `POST /imports/{id}/rows/{row_id}/relink` — resolusi manual baris berstatus UNMATCHED (admin
  memasok owner/outlet/lead ID yang tidak bisa ditemukan otomatis saat commit), baris kembali ke
  VALID untuk di-commit ulang. Diverifikasi live (flip UNMATCHED->VALID, serta 409 saat dipanggil
  pada baris yang bukan UNMATCHED).
- `GET /imports/summary` — agregat jumlah batch per status, `needs_attention` untuk
  VALIDATION_FAILED/COMMIT_FAILED. Diverifikasi live.
- Unit test parser untuk 3 profil yang sebelumnya belum punya test eksplisit: `NEW_SUBSCRIBE`,
  `SALES_CALL_CHAT`, `SALES_TARGET` (`TestParseNewSubscribeRow_*`, `TestParseSalesCallChatRow_*`,
  `TestParseSalesTargetRow_*` di `excel_test.go`) — melengkapi `MONTHLY_ACTIVE`/`BONUS_MITRA` yang
  sudah punya test sejak sebelumnya.
- Diverifikasi ulang: kelima profil (`NEW_SUBSCRIBE`, `MONTHLY_ACTIVE`, `BONUS_MITRA`,
  `SALES_CALL_CHAT`, `SALES_TARGET`) re-upload fixture existing (`.cache/sprint15-fixtures/*.xlsx`)
  → seluruhnya idempotent-return batch COMMITTED yang sama seperti sebelumnya, mengonfirmasi tidak
  ada regresi dari perubahan skema commission (§5 Sprint 15a) terhadap `BONUS_MITRA` (profil ini
  menyimpan data sebagai snapshot historis di `partner_bonus_referral_snapshots`, tidak menyentuh
  `partner`/`commission_rules` sama sekali — jadi aman dari restrukturisasi package_id->plan_id).
- `go build/vet/test ./...` bersih di seluruh repo setelah penambahan ini.

Quality:
- Unit/integration test:
  - `go test ./internal/importing/...` PASS
  - `go test ./internal/platform/httpserver/...` PASS
  - `go test ./internal/platform/migration/...` PASS
- Migration status:
  - `20260730000200_bonus_mitra_snapshots.sql` berhasil dijalankan di DB lokal.
- Docker build:
  - tidak dilakukan perubahan Docker pada turn ini.
- Defect terbuka:
  - tidak ada defect blocker tersisa pada jalur smoke test Sprint 15 setelah perbaikan runtime di atas diterapkan.

Impediments:
- Terdapat worker lama yang masih aktif dan ikut mengambil antrean job dari database yang sama, sehingga hasil awal pengujian sempat tercampur antara binary lama dan binary baru.

Risiko Baru:
- Risiko: frontend mencoba commit batch tanpa polling status final validasi.
- Dampak: akan menerima `INVALID_BATCH_STATUS` dan mengira backend rusak.
- Mitigasi: dokumentasi Sprint 15 menegaskan bahwa `status` batch adalah source of truth utama UI.
- Owner: Frontend Engineer

- Risiko: environment lokal/staging belum menjalankan migration Sprint 15 terbaru.
- Dampak: profile seperti `BONUS_MITRA` bisa gagal commit walaupun upload dan validasi sukses.
- Mitigasi: pastikan `go run . migrate up` dijalankan sebelum UAT/import testing.
- Owner: Backend Engineer / DevOps

- Risiko: file profile multi-sheet dikirim tanpa `sheet_name` atau `target_sales_user_id`.
- Dampak: request gagal di validasi.
- Mitigasi: frontend wajib membuat field pemilihan sheet dan sales menjadi mandatory pada profile terkait.
- Owner: Frontend Engineer

Keputusan yang Dibutuhkan:
- Apakah setelah Sprint 15 ini OpenAPI juga perlu diperluas lagi sampai contoh payload per profile import di level schema agar lebih nyaman dipakai tim frontend.

Rencana Sprint Berikutnya:
- Jika diperlukan, lanjut ke update OpenAPI Sprint 15.
- Review ulang bersama frontend untuk UX form import multi-sheet.
- Lanjut ke backlog Sprint 16 atau carry-over lain setelah dokumen Sprint 15 disetujui.
