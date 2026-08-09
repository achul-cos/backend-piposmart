# Sprint 16g — Kode Baris, Akun Testing, Export DB-Driven, dan Guard Lead/Sales

## Ringkasan

Sprint 16g melanjutkan pekerjaan migrasi data admin (`Owner & Outlet` + `New & Subscribe` 2021–2026)
dan menyelesaikan dua kebutuhan bisnis yang sebelumnya belum tuntas:

1. **`Kode Baris` dipersist ke database** (`outlets.row_code`) lalu ditampilkan kembali di export admin.
2. **`Akun Testing` dipersist dan bisa diubah oleh admin** (`owners.is_testing_account`) agar owner milik
   karyawan/internal Piposmart tidak ikut masuk pipeline lead/sales/call/chat/closing.
3. **Export Excel owner/outlet diubah total jadi DB-driven**, tetap memakai format admin yang sama, tapi
   tidak lagi membaca file Excel sumber langsung dari disk.
4. **Guard visibilitas dan assignment diperketat**: testing account hilang dari list lead, tidak bisa
   di-assign ke sales, dan tidak terlihat lagi oleh supervisor/sales di modul owner.

## Dokumen Terkait

- [sprint-16g.md](./sprint-16g.md) — laporan perubahan lengkap.
- [api-testing.md](./api-testing.md) — bukti verifikasi API/smoke test yang dijalankan pada 9 Agustus 2026.
