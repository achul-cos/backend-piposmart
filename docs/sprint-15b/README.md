# API Documentation Update - Sprint 15b Interaction Multichannel

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 15b |
| Tanggal Update | 2 Agustus 2026 |
| Environment | Local Development |
| Fokus | Revisi model interaction lead agar satu interaction dapat menyimpan call dan chat sekaligus |

## 2. Ringkasan Perubahan

Sprint 15b memperbaiki logika interaksi lead.

Sebelumnya, saat frontend mengirim laporan call dan chat untuk satu aktivitas yang sama, backend dapat membuat dua interaction terpisah. Setelah revisi ini:

- satu interaction dapat memuat `call_status`;
- satu interaction dapat memuat `chat_status`;
- satu interaction dapat memuat keduanya sekaligus;
- `type` sekarang diturunkan otomatis menjadi:
  - `CALL`
  - `CHAT`
  - `CALL_CHAT`

## 3. Perubahan API

### 3.1 Endpoint yang terdampak langsung

- `POST /api/v1/leads/{lead_id}/interactions`
- `GET /api/v1/leads/{lead_id}/interactions`
- `GET /api/v1/leads/{lead_id}/interactions/all`
- `GET /api/v1/leads/{lead_id}/interactions/all-deleted`
- `POST /api/v1/leads/{lead_id}/closings`

### 3.2 Request field baru

#### Interaction Lead

- `call_status`
- `chat_status`

#### Closing Lead

- `call_status`
- `chat_status`

### 3.3 Backward compatibility

Field lama berikut masih diterima:

- `type`

Namun untuk frontend baru, format yang direkomendasikan adalah:

```json
{
  "call_status": "TERHUBUNG",
  "chat_status": "TERBALAS"
}
```

## 4. Perubahan Response

Interaction sekarang dapat mengembalikan:

- `type: CALL`
- `type: CHAT`
- `type: CALL_CHAT`

serta field:

- `call_status`
- `chat_status`

Contoh:

```json
{
  "id": 99713,
  "type": "CALL_CHAT",
  "call_status": "TERHUBUNG",
  "chat_status": "TERBALAS"
}
```

## 5. Audit Dampak

| Area | Hasil Audit |
| --- | --- |
| Lead interaction create | Aman |
| Lead interaction list/filter | Aman |
| Closing create | Aman |
| Import `SALES_CALL_CHAT` | Aman |
| Factory & Seeder | Aman |
| OpenAPI | Diperbarui |

## 6. Temuan Penting Saat Validasi

### 6.1 Validasi HTTP untuk interaction kosong

Temuan:

- setelah perubahan service/repository, request tanpa channel harusnya ditolak dengan pesan domain yang jelas;
- sebelumnya kondisi ini berpotensi jatuh ke error generic.

Status:

- **Sudah diperbaiki**

Kode error final:

- `INTERACTION_CHANNEL_REQUIRED`

### 6.2 Port lokal 8080 sempat dipakai instance backend lama

Temuan:

- smoke test awal sempat masuk ke proses backend lama yang masih aktif;
- hasilnya tidak mencerminkan kode Sprint 15b terbaru.

Status:

- **Sudah diisolasi**

Validasi final Sprint 15b dilakukan pada port:

- `http://127.0.0.1:18084`

## 7. Verifikasi Teknis

Command yang dijalankan:

```powershell
gofmt -w internal/activity/... internal/closing/... internal/importing/... internal/platform/factory/... internal/platform/migration/...
go test ./internal/activity ./internal/closing ./internal/importing ./internal/platform/factory ./internal/platform/migration
go test ./internal/platform/httpserver -run TestOpenAPISpecIsServed -count=1
go run . migrate up
```

Hasil:

- logic package utama terdampak: **PASS**
- OpenAPI served test: **PASS**
- migration sprint 15b: **PASS**

Catatan Windows:

- pada lokal Windows masih muncul warning cleanup `go: unlinkat ... Access is denied`;
- warning ini muncul setelah package test berstatus `ok`, sehingga tidak mengubah hasil assertion test.

## 8. Catatan Integrasi Frontend

Frontend sekarang sebaiknya memperlakukan form call dan chat sebagai satu payload interaction.

Aturan aman:

1. jika user hanya mengisi hasil call -> kirim `call_status`;
2. jika user hanya mengisi hasil chat -> kirim `chat_status`;
3. jika user mengisi keduanya -> kirim dua field sekaligus;
4. jangan lagi mengirim dua request interaction hanya karena satu form memiliki section call dan section chat.

## 9. Dokumen Pendamping

- [Sprint 15b Summary](./sprint-15b.md)
- [Sprint 15b API Testing](./api-testing.md)
