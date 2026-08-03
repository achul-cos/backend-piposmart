# Frontend Briefing — Sprint 16b

## Tujuan

Dokumen ini menjadi acuan frontend untuk implementasi UI/UX fitur upgrade subscription pada Sprint 16b.

Fokus utamanya:

- membuat flow upgrade paket yang mudah dipahami admin;
- menampilkan histori upgrade dengan jelas;
- menjaga data sales/closing/reconciliation tetap terbaca benar dari sisi UI.

## Konteks Bisnis

Upgrade pada CRM Piposmart bukan berarti membuat paket baru dari nol.

Logikanya:

- owner sudah punya subscription aktif;
- owner ingin pindah ke package yang lebih tinggi;
- masa aktif lama tidak dibuang;
- sistem memotong subscription lama pada tanggal efektif upgrade;
- sistem membuat subscription baru untuk sisa hari yang tersisa;
- owner membayar prorata berdasarkan harga plan target untuk sisa hari itu.

Contoh:

- Pro 1 bulan = Rp300.000
- durasi = 30 hari
- sisa masa aktif lama = 3 hari
- tagihan upgrade = Rp30.000

## Endpoint yang Dipakai Frontend

### 1. Buat Upgrade

`POST /api/v1/subscriptions/{subscription_id}/upgrades`

Body:

```json
{
  "plan_id": 7,
  "closing_id": 22,
  "idempotency_key": "sub-upgrade-15-20260803-001",
  "external_reference": "ADMIN-UPGRADE-20260803-001",
  "purchased_at": "2026-08-03T10:30:00Z",
  "effective_start_date": "2026-08-01",
  "note": "Upgrade dari Basic ke Pro untuk sisa masa aktif subscription."
}
```

Keterangan field:

- `plan_id`
  - wajib
  - plan target upgrade
- `closing_id`
  - opsional
  - jika diisi, backend akan mencoba auto reconcile ke closing sales
- `idempotency_key`
  - disarankan selalu diisi frontend
  - mencegah double submit
- `external_reference`
  - opsional, tapi tetap baik untuk audit admin
- `purchased_at`
  - waktu pembayaran upgrade sebenarnya
- `effective_start_date`
  - tanggal mulai upgrade berlaku
- `note`
  - catatan admin

### 2. List Subscription Order

`GET /api/v1/subscription-orders`

Filter baru:

- `order_type=UPGRADE`
- `source_subscription_id={id}`

Contoh:

```http
GET /api/v1/subscription-orders?order_type=UPGRADE&source_subscription_id=15
```

### 3. Detail Subscription Order

`GET /api/v1/subscription-orders/{order_id}`

Response sekarang lebih kaya karena bisa memuat:

- `order`
- `subscription`
- `period`
- `reconciliation`
- `issue`

## Data Baru yang Perlu Diperhatikan Frontend

Pada `order`:

- `order_type`
  - `NEW`
  - `UPGRADE`
- `source_subscription`
  - subscription asal sebelum upgrade
- `upgrade`
  - object detail histori upgrade

Struktur `upgrade`:

```json
{
  "effective_start_date": "2026-08-01",
  "original_end_date": "2026-09-01",
  "remaining_days": 3,
  "daily_price": "10000.00",
  "previous_package": {
    "id": 1,
    "code": "BASIC",
    "name": "Basic",
    "level_order": 1
  },
  "previous_plan": {
    "id": 3,
    "code": "BASIC-1M",
    "name": "Basic 1 Bulan",
    "tenure_months": 1,
    "duration_days": 30,
    "price": "150000.00",
    "currency": "IDR"
  }
}
```

Pada `reconciliation` juga bisa muncul:

- `admin_final_amount`
- `admin_tenure_months`
- `note`
- `reason`

Catatan:

- untuk upgrade prorata, `admin_tenure_months` bisa kosong;
- frontend jangan menganggap field ini selalu ada.

## Rekomendasi UI

### A. Tabel Subscription Order

Tambahkan indikator visual:

- badge `Baru` untuk `order_type = NEW`
- badge `Upgrade` untuk `order_type = UPGRADE`

Jika order upgrade:

- tampilkan source subscription
- tampilkan ringkas:
  - sisa hari
  - tanggal efektif upgrade

Contoh informasi ringkas di row:

- `Upgrade`
- `Dari SUB-00015`
- `Sisa 3 hari • efektif 01/08/2026`

### B. Detail Subscription Order

Untuk order upgrade, tampilkan section khusus:

- Subscription sumber
- Paket sebelumnya
- Plan sebelumnya
- Tanggal mulai efektif upgrade
- Tanggal akhir masa aktif lama
- Sisa hari yang ditagihkan
- Harga harian prorata
- Nominal upgrade

### C. Form Upgrade

Flow minimum yang disarankan:

1. admin pilih subscription aktif;
2. frontend tampilkan ringkasan subscription saat ini:
   - package sekarang
   - plan sekarang
   - active from
   - active until
3. admin pilih plan target upgrade;
4. admin isi:
   - purchased_at
   - effective_start_date
   - closing_id opsional
   - note
5. frontend submit ke route upgrade.

## Validasi yang Sebaiknya Dilakukan Frontend

Sebelum submit, frontend sebaiknya mencegah error yang bisa diprediksi:

### Wajib

- `plan_id` harus terisi
- `idempotency_key` sebaiknya selalu dibuat otomatis

### Tanggal

- `effective_start_date` jangan lebih besar dari tanggal bisnis `purchased_at`
- jika user memilih tanggal masa depan, tampilkan warning sebelum submit

### Closing

Jika frontend menyediakan picker `closing_id`, usahakan filter kandidat closing:

- owner sama
- outlet sama jika memungkinkan
- status closing bukan rejected
- plan closing sama dengan plan target upgrade

Ini penting supaya mengurangi `CLOSING_MISMATCH`.

## Error yang Perlu Ditangani Frontend

### `SUBSCRIPTION_NOT_ACTIVE`

Makna:

- subscription sumber tidak aktif

Perilaku UI:

- sembunyikan tombol upgrade untuk subscription non-active
- atau tampilkan disabled state dengan tooltip

### `UPGRADE_NOT_ALLOWED`

Makna umum:

- tanggal efektif tidak valid
- plan target bukan level upgrade
- sisa hari tidak valid

Perilaku UI:

- tampilkan pesan spesifik yang mudah dipahami admin:
  - “Upgrade hanya bisa ke paket yang lebih tinggi”
  - “Tanggal efektif upgrade tidak valid”

### `CLOSING_MISMATCH`

Makna:

- closing tidak cocok dengan owner/outlet/plan upgrade

Perilaku UI:

- tampilkan error jelas
- minta admin memilih closing lain atau kosongkan dulu `closing_id`

### `IDEMPOTENCY_REQUIRED`

Makna:

- frontend tidak mengirim `idempotency_key` atau `external_reference`

Perilaku UI:

- idealnya tidak terjadi jika frontend selalu generate `idempotency_key`

## Saran UX

- tombol `Upgrade` hanya tampil pada subscription `ACTIVE`
- jangan campur form order baru dengan form upgrade dalam satu form tanpa pembeda
- gunakan label yang tegas:
  - `Order Baru`
  - `Upgrade Paket`
- untuk order upgrade, tampilkan highlight bahwa:
  - ini bukan pembelian full periode baru
  - ini adalah tagihan sisa hari

## Kesimpulan Frontend

Frontend Sprint 16b sebaiknya memperlakukan upgrade sebagai alur khusus, bukan variasi kecil dari order biasa.

Alasan:

- field bisnisnya berbeda;
- validasinya berbeda;
- tampilannya perlu menjelaskan histori perubahan paket;
- data reconciliation/closing-nya juga punya konteks yang berbeda dari order subscription biasa.
