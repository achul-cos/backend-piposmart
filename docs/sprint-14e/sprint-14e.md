Sprint: 14e - Seeder Large Scale & Dataset Refresh  
Periode: 28 Juli 2026  
Status: GREEN

Sprint Goal:
- Menambahkan flag `--scale` pada `seed demo --preset=large`.
- Memperkaya large seeder agar sesuai modul backend yang sudah berkembang sampai Sprint 14.

Committed Deliverables:
- Parser CLI `--scale` untuk preset `large`.
- Mapping scale ke jumlah owner target.
- Backward compatibility preset `large` lama.
- Large seeder yang mengisi modul wallet, subscription, training, reconciliation, dan partner demo.
- Testing dan dokumentasi Sprint 14e.

Completed:
- Menambahkan field `Scale` pada opsi seeder.
- Menambahkan field `From`, `To`, dan `Variation` pada opsi seeder.
- Menambahkan validasi `--scale` hanya untuk preset `large`.
- Menambahkan default `scale=10` untuk menjaga perilaku lama preset `large`.
- Menambahkan rentang tanggal `--from/--to` untuk menggantikan pemakaian utama `--as-of`.
- Menambahkan `--variation` (0..1) untuk mengatur persebaran data harian.
- Menambahkan mapping skala `1..10` ke jumlah owner target.
- Memperluas `seedDemoLarge()`:
  - user Supervisor/Sales proporsional;
  - owner dan outlet multi-varian;
  - lead dan interaction;
  - training report;
  - sales closing;
  - wallet top-up/debit;
  - subscription order linked closing;
  - hanging order;
  - partner referral demo.
- Memperbaiki partner demo agar referral mengambil lead pertama yang tersedia, bukan kode lead hard-coded yang tidak ada.
- Memperbarui README project.
- Menambahkan dokumentasi `docs/sprint-14e/README.md`.

Not Completed / Carry Over:
- Item: smoke test Docker penuh.
- Penyebab: command `docker` tidak tersedia di PATH shell sesi ini walaupun user menyampaikan Docker sudah terpasang.
- Estimasi ulang: 10-15 menit untuk verifikasi lokal di mesin user setelah PATH Docker aktif.

Demo Evidence:
- Endpoint/Swagger: tidak relevan, perubahan berada di CLI seeder.
- Command:
  - `go build .`
  - `go test ./internal/platform/seeder -run "Test(Parse|LargeSeed)" -count=1`
- Dokumentasi: `docs/sprint-14e/README.md`

Quality:
- Unit/integration test:
  - Assertion test parser dan mapping scale PASS.
  - Catatan Windows cleanup: `seeder.test.exe: Access is denied` terjadi setelah test selesai.
- Migration status: tidak ada migration baru.
- Docker build: tidak diuji di sesi ini karena executable Docker tidak tersedia di PATH shell.
- Defect terbuka:
  - tidak ada defect fungsional baru yang teridentifikasi dari compile/test.

Impediments:
- Docker CLI tidak dapat dipanggil dari shell sesi ini.

Risiko Baru:
- Risiko: mapping `scale=9` bernilai `1000`, tidak monotonic terhadap `scale=8`.
- Dampak: volume dataset scale 9 lebih kecil dari scale 8.
- Mitigasi: saat ini implementasi mengikuti briefing apa adanya dan sudah didokumentasikan; jika stakeholder ingin koreksi ke nilai lain, perubahan cukup di mapping skala tanpa memengaruhi kontrak CLI.
- Owner: Product/System Analyst

Keputusan yang Dibutuhkan:
- Apakah mapping `scale=9 => 1000` memang final, atau perlu direvisi pada sprint berikutnya.

Rencana Sprint Berikutnya:
- Lakukan smoke test Docker penuh di environment yang memiliki executable Docker aktif.
- Jika diperlukan, tambah preset laporan count otomatis pasca seeding agar QA lebih mudah memverifikasi isi dataset.
