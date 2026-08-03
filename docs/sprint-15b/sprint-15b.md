# Sprint 15b - Revisi Interaction Multichannel

## Ringkasan

Sprint 15b merevisi model `customer_interactions` agar satu interaction dapat menyimpan channel `CALL`, `CHAT`, atau keduanya sekaligus (`CALL_CHAT`) dalam satu record.

Sebelumnya, form laporan call & chat lead berpotensi menghasilkan dua interaction terpisah untuk satu aktivitas follow-up yang sebenarnya sama. Perubahan sprint ini menyatukan model tersebut tanpa memutus kompatibilitas route dan payload lama.

## Tujuan

- satu aktivitas follow-up Sales menghasilkan satu interaction;
- interaction dapat memiliki `call_status`, `chat_status`, atau keduanya;
- import `SALES_CALL_CHAT` tidak lagi menggandakan interaction untuk satu row;
- closing tetap dapat mencatat jejak interaction yang menghasilkan closing;
- factory, seeder, dan filter list tetap bekerja setelah perubahan schema.

## Perubahan Teknis

### Database

Migration baru:

- `20260802000100_customer_interactions_multichannel.sql`

Penambahan kolom:

- `customer_interactions.call_status`
- `customer_interactions.chat_status`

Penambahan index:

- `idx_customer_interactions_call_status_at`
- `idx_customer_interactions_chat_status_at`

### Backend Activity

Perubahan utama:

- `CreateInteractionRequest` sekarang mendukung:
  - `call_status`
  - `chat_status`
  - `type` lama tetap diterima untuk backward compatibility
- `CustomerInteractionResponse` sekarang mengembalikan:
  - `type`
  - `call_status`
  - `chat_status`
- backend menurunkan tipe interaction otomatis:
  - hanya `call_status` -> `CALL`
  - hanya `chat_status` -> `CHAT`
  - keduanya -> `CALL_CHAT`

### Backend Closing

`CreateClosingRequest` sekarang juga menerima:

- `call_status`
- `chat_status`

Tujuannya agar interaction yang dibuat saat closing mengikuti model multichannel yang sama.

### Importing

Worker import `SALES_CALL_CHAT` sekarang mengirim satu request interaction yang bisa membawa dua channel sekaligus, bukan memecahnya menjadi dua interaction terpisah.

### Factory dan Seeder

Factory interaction dan seed data demo diperbarui agar:

- bisa membuat interaction `CALL_CHAT`;
- tetap bisa membuat interaction legacy `CALL` atau `CHAT`;
- data closing factory ikut mengisi `call_status` / `chat_status` yang sesuai.

## Audit Dampak Modul

| Modul | Dampak | Status |
| --- | --- | --- |
| Activity / Lead Interaction | Schema dan request/response berubah | Aman |
| Closing | Interaction pendamping closing ikut berubah | Aman |
| Importing | Row call+chat kini jadi satu interaction | Aman |
| Factory | Data dummy interaction multichannel tersedia | Aman |
| Seeder | Demo seed kompatibel | Aman |
| Analytics | Tetap aman karena basis hitung utama masih per record interaction | Aman |

## Kompatibilitas

Backward compatibility tetap dipertahankan.

Jika client lama masih mengirim:

```json
{
  "type": "CALL"
}
```

maka backend masih menerima request tersebut, lalu mengisi:

- `type = CALL`
- `call_status = RECORDED`

Ini hanya mode kompatibilitas sementara. Format utama yang direkomendasikan untuk frontend baru adalah memakai `call_status` dan `chat_status`.

## Risiko yang Diselesaikan

### Risiko lama

- satu form call & chat menciptakan dua interaction;
- histori interaction menjadi dobel;
- analytics interaksi berpotensi bias;
- import call/chat berpotensi melebihkan jumlah aktivitas real.

### Risiko baru yang dicegah

- request tanpa channel sama sekali sekarang ditolak dengan `400 INTERACTION_CHANNEL_REQUIRED`;
- filter list `type=CALL` dan `type=CHAT` tetap membaca data lama maupun data baru;
- closing validation ikut menerima model multichannel.

## Verifikasi Teknis

Command yang dijalankan:

```powershell
gofmt -w internal/activity/... internal/closing/... internal/importing/... internal/platform/factory/... internal/platform/migration/...
go test ./internal/activity ./internal/closing ./internal/importing ./internal/platform/factory ./internal/platform/migration
go test ./internal/platform/httpserver -run TestOpenAPISpecIsServed -count=1
go run . migrate up
```

Hasil:

- package terdampak compile dan test logic lulus;
- migration sprint 15b berhasil terpasang;
- OpenAPI endpoint tetap tersaji;
- pada Windows lokal masih muncul warning cleanup `unlinkat ... Access is denied`, tetapi assertion test tetap `ok`.

## Output Sprint

- schema interaction multichannel siap;
- request interaction lead siap untuk frontend baru;
- request closing siap untuk frontend baru;
- OpenAPI diperbarui;
- dokumentasi testing sprint 15b tersedia.
