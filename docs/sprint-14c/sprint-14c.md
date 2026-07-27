Sprint: 14c - Rekap Status Langganan Outlet Bulanan
Periode: 27 Juli 2026
Status: GREEN

Sprint Goal:
- Menyediakan endpoint backend yang dapat merekap status langganan seluruh outlet per bulan.

Committed Deliverables:
- Endpoint rekap bulanan outlet subscription status.
- Aturan klasifikasi status per akhir bulan.
- Dukungan filter status bulanan.
- Dokumentasi Sprint 14c.

Completed:
- Menambahkan route baru `GET /api/v1/outlets/subscription-statuses`.
- Menambahkan rule status bulanan: `NOT_SUBSCRIBE`, `EXPIRED`, `JATUH_TEMPO`, `BERLANGGANAN`, `NEW`.
- Menambahkan perilaku khusus paket 1 bulan -> label `BERLANGGANAN 1 BULAN`.
- Menambahkan unit test untuk rule klasifikasi status langganan outlet.
- Memperbarui README ringkas route.
- Membuat dokumentasi `docs/sprint-14c/README.md` dan laporan sprint ini.

Not Completed / Carry Over:
- Belum menambahkan endpoint ini ke OpenAPI/Swagger lama pada sprint ini.

Demo Evidence:
- Endpoint: `GET /api/v1/outlets/subscription-statuses`
- Dokumentasi: `docs/sprint-14c/README.md`
- Test: `go build ./...` PASS, `go test ./...` PASS

Quality:
- Unit/integration test: PASS.
- Migration status: tidak ada migration baru.
- Defect terbuka: tidak ada defect compile/test yang tersisa.

Impediments:
- Terdapat gangguan sementara cleanup file temp pada `go test` di Windows, tetapi rerun berikutnya PASS dan terkonfirmasi bukan masalah kode aplikasi.

Risiko Baru:
- Risiko: logika status bulanan bersifat business-rule heavy dan bisa berubah bila stakeholder mengubah definisi rentang hari/status.
- Dampak: endpoint perlu penyesuaian rule dan kemungkinan update unit test.
- Mitigasi: rule sudah diisolasi pada file baru Sprint 14c dan dilindungi unit test agar perubahan berikutnya lebih aman.
- Owner: Backend Engineer.

Keputusan yang Dibutuhkan:
- Tidak ada blocker. Jika nanti stakeholder ingin kode status berbeda dari label saat ini, perubahan cukup dilakukan di layer rule/response endpoint ini.

Rencana Sprint Berikutnya:
- Jika diperlukan, lanjutkan sinkronisasi OpenAPI/Swagger untuk endpoint Sprint 14c.
- Jika diperlukan, tambahkan manual smoke test HTTP dan/atau export report untuk rekap langganan outlet.
