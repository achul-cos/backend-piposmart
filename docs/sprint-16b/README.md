# Sprint 16b — Subscription Upgrade Prorata

## Ringkasan

Sprint 16b menambahkan alur upgrade paket subscription yang sedang aktif.

Contoh kasus bisnis:

- owner sudah membeli paket Basic;
- paket sudah berjalan;
- owner ingin naik ke paket yang lebih tinggi, misalnya Pro;
- masa aktif lama tidak di-reset dari nol;
- yang ditagihkan hanyalah sisa hari yang belum terpakai, dihitung prorata dari harga plan target.

## Prinsip Bisnis

- upgrade hanya berlaku untuk subscription yang masih aktif;
- upgrade hanya boleh menuju package dengan level lebih tinggi;
- subscription lama dipotong pada tanggal efektif upgrade;
- backend membuat subscription baru untuk sisa hari yang belum habis;
- harga upgrade dihitung prorata:
  - `harga plan target / duration_days plan target * sisa hari`
- admin dapat melakukan backfill:
  - `purchased_at` dapat diisi tanggal/waktu pembayaran sebenarnya;
  - `effective_start_date` dapat diisi tanggal mulai upgrade sebenarnya.

## Perubahan Backend

- route baru:
  - `POST /api/v1/subscriptions/{subscription_id}/upgrades`
- upgrade sekarang bisa menerima `closing_id` opsional
- jika `closing_id` dikirim:
  - backend akan auto reconcile ke closing sales
  - jika nominal closing berbeda dengan nominal upgrade prorata, backend melakukan partial confirm otomatis
  - omset closing dipin ke nominal upgrade yang benar
- filter baru pada list subscription order:
  - `order_type`
  - `source_subscription_id`
- order upgrade dicatat sebagai `order_type = UPGRADE`
- wallet ledger upgrade dicatat dengan `source_type = SUBSCRIPTION_UPGRADE`
- histori upgrade menyimpan:
  - subscription sumber
  - tanggal mulai efektif upgrade
  - tanggal akhir original subscription
  - sisa hari
  - harga harian prorata
  - snapshot package/plan lama

## Dampak Data

Ketika upgrade dibuat:

1. subscription lama dipotong pada `effective_start_date`;
2. subscription lama diubah menjadi `CANCELED`;
3. dibuat subscription baru untuk sisa hari;
4. dibuat subscription order upgrade;
5. wallet owner didebit sesuai hasil prorata;
6. dibuat issue hanging order jika upgrade belum terkait closing sales.

Jika `closing_id` dikirim:

1. order upgrade langsung masuk flow reconciliation;
2. order dapat berstatus `RECONCILED`;
3. closing sales ikut dikonfirmasi;
4. jika nominal closing awal berbeda, nominal closing diubah ke nominal upgrade aktual agar KPI/omset tidak salah hitung.

## Catatan

Sprint ini fokus ke fondasi backend dan histori transaksi upgrade.

Pencatatan upgrade:

- sekarang sudah bisa diarahkan langsung ke closing sales lewat `closing_id`;
- jika tidak dikirim `closing_id`, order upgrade tetap akan masuk sebagai hanging order dan bisa direview kemudian;
- frontend disarankan membaca briefing implementasi pada dokumen terpisah.

## Briefing Frontend

Lihat dokumen:

- [frontend-briefing.md](./frontend-briefing.md)
