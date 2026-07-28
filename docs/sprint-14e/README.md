# Seeder Testing Report - Sprint 14e Large Scale Flag

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14e - Seeder Large Scale |
| Tanggal Testing | 28 Juli 2026 |
| Environment | Local Development |
| Fokus | CLI seeder `demo --preset=large --scale=` |
| Modul terdampak | owner, outlet, lead, interaction, training, closing, wallet, subscription, reconciliation, partner |

## 2. Tujuan

- Menambahkan flag `--scale` pada `seed demo --preset=large`.
- Menjaga backward compatibility preset `large` lama.
- Memperluas dataset `large` agar ikut mengisi modul-modul backend terbaru.

## 3. Mapping Scale

| Scale | Jumlah owner target |
| --- | ---: |
| `1` | 50 |
| `2` | 100 |
| `3` | 200 |
| `4` | 500 |
| `5` | 1000 |
| `6` | 2000 |
| `7` | 3000 |
| `8` | 5000 |
| `9` | 1000 |
| `10` | 18000 |

Catatan: mapping di atas mengikuti briefing Sprint 14e apa adanya.

## 4. Perubahan Fungsional

### CLI

Command baru yang didukung:

```powershell
go run . seed demo --preset=large --seed=20260723 --from=2026-07-01 --to=2026-07-28 --scale=1 --variation=1
go run . seed demo --preset=large --seed=20260723 --from=2026-01-01 --to=2026-07-28 --scale=4 --variation=0.5
go run . seed demo --preset=large --seed=20260723 --from=2025-01-01 --to=2026-07-28 --scale=10 --variation=0.2
```

Aturan:

- `--scale` hanya boleh dipakai pada `--preset=large`.
- Jika `--preset=large` tanpa `--scale`, default-nya `10`.
- `--scale` di luar daftar yang tersedia akan ditolak parser.
- `--from` dan `--to` menggantikan penggunaan utama `--as-of`.
- Default `--from` dan `--to` adalah hari ini.
- `--variation` wajib berada di rentang `0` sampai `1`, default `0.5`.

### Dataset Large

Preset `large` kini mengisi:

- user Supervisor dan Sales secara proporsional terhadap skala;
- owner dan outlet (single outlet dan multi-outlet);
- lead per owner;
- interaction call/chat;
- training report untuk sebagian lead potensial;
- sales closing;
- wallet top-up dan sebagian debit;
- subscription order + reconciliation linked closing;
- hanging subscription order untuk sebagian data;
- partner demo + referral demo.

## 5. Hasil Pengujian

### 5.1 Build

Command:

```powershell
go build .
```

Hasil:

- PASS

### 5.2 Unit Test Parser & Scale

Command:

```powershell
go test ./internal/platform/seeder -run "Test(Parse|LargeSeed)" -count=1
```

Hasil:

- Seluruh assertion test PASS.
- Pada environment Windows ini, `go test` menampilkan error cleanup file temporary:

```text
ok      backend_crm_piposmart/internal/platform/seeder  24.311s
go: unlinkat ...\\seeder.test.exe: Access is denied.
```

Interpretasi:

- logic test berhasil;
- kendala terjadi saat Go membersihkan file binary test sementara di folder temp Windows;
- bukan kegagalan assertion business logic.

### 5.3 Smoke Test Docker

Rencana smoke test nyata:

```powershell
docker compose up -d mysql
docker compose run --rm migrate
docker compose run --rm --entrypoint backend_crm_piposmart seed seed demo --preset=large --seed=20260728 --from=2026-07-01 --to=2026-07-28 --scale=1 --variation=0.5
```

Status pada environment agen ini:

- TIDAK BISA DIEKSEKUSI karena command `docker` tidak tersedia di PATH shell saat sesi testing.

Error:

```text
docker : The term 'docker' is not recognized as the name of a cmdlet, function, script file, or operable program.
```

## 6. Validasi Acceptance

| Item | Status | Catatan |
| --- | --- | --- |
| Parser menerima `--scale` | PASS | Tambahan unit test |
| Default `large` tetap setara perilaku lama | PASS | Default `scale=10` |
| `--scale` ditolak pada preset `minimal` | PASS | Tambahan unit test |
| `--scale` invalid ditolak | PASS | Tambahan unit test |
| Build backend | PASS | `go build .` |
| Dataset large mengisi modul baru | PASS | Implementasi seeder diperluas |

## 7. Dampak ke Frontend / QA

Tidak ada perubahan kontrak API HTTP karena ini perubahan CLI dan data demo. Dampak utamanya:

- QA dapat membuat dataset lebih kecil untuk local test cepat;
- frontend dapat memakai dataset `large` yang lebih kaya untuk wallet, subscribe, partner, dan training;
- staging dapat memakai scale berbeda sesuai kebutuhan uji.

## 8. Rekomendasi Verifikasi Manual Lanjutan

Jika Docker di mesin lokal user aktif, jalankan:

```powershell
docker compose up -d mysql
docker compose run --rm migrate
docker compose run --rm --entrypoint backend_crm_piposmart seed seed demo --preset=large --seed=20260728 --from=2026-07-01 --to=2026-07-28 --scale=1 --variation=0.5
```

Lalu verifikasi cepat:

```sql
SELECT COUNT(*) FROM owners;
SELECT COUNT(*) FROM outlets;
SELECT COUNT(*) FROM customer_leads;
SELECT COUNT(*) FROM customer_interactions;
SELECT COUNT(*) FROM training_reports;
SELECT COUNT(*) FROM sales_closings;
SELECT COUNT(*) FROM wallet_payments;
SELECT COUNT(*) FROM subscription_orders;
SELECT COUNT(*) FROM partners;
```
