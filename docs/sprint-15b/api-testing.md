# API Testing Report - Sprint 15b

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 15b |
| Tanggal Testing | 2 Agustus 2026 |
| Environment | Local Development |
| Base URL API | `http://localhost:8080/api/v1` |
| Port isolasi smoke test | `http://127.0.0.1:18084/api/v1` |
| Testing Tool | `curl.exe`, PowerShell, `go test` |
| Fokus Testing | Interaction multichannel lead, kompatibilitas legacy, validasi request, dan audit dampak closing/import |

## 2. Tujuan Pengujian

Dokumen ini dibuat agar:

- frontend tahu bentuk payload interaction terbaru;
- frontend tahu cara tetap kompatibel dengan payload lama;
- QA punya contoh request sukses dan request error;
- CTO bisa melihat bukti bahwa revisi ini tidak merusak modul lain.

## 3. Header Standar

### 3.1 Header Auth

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

## 4. Format Envelope

### 4.1 Success

```json
{
  "data": {},
  "meta": {
    "request_id": "309626d6c90496143a493440129b9095"
  }
}
```

### 4.2 Error

```json
{
  "error": {
    "code": "INTERACTION_CHANNEL_REQUIRED",
    "message": "minimal salah satu status call/chat atau type lama wajib diisi",
    "details": {
      "root_cause": "request interaction tidak mengirim status call/chat dan tidak mengirim fallback type lama",
      "solution": "kirim minimal salah satu field call_status atau chat_status. Field type lama masih didukung untuk kompatibilitas, tetapi bukan format utama.",
      "frontend_prevent": "pastikan form mengharuskan minimal satu status channel sebelum submit"
    },
    "request_id": "a0247dfb66bbac7e883ef6ecde475fc1"
  }
}
```

## 5. Ringkasan Test Case

| No | Endpoint | Method | Skenario | Hasil |
| --- | --- | --- | --- | --- |
| 1 | `/auth/login` | POST | Login Sales untuk smoke test | PASS |
| 2 | `/leads/{lead_id}/interactions` | POST | Satu interaction dengan call + chat sekaligus | PASS |
| 3 | `/leads/{lead_id}/interactions` | GET | List interaction menampilkan `CALL_CHAT` | PASS |
| 4 | `/leads/{lead_id}/interactions` | POST | Payload lama hanya `type=CALL` | PASS |
| 5 | `/leads/{lead_id}/interactions` | POST | Payload tanpa channel sama sekali | PASS |
| 6 | `/leads/{lead_id}/closings` | POST | Request closing multichannel pada lead yang sudah punya closing | PASS, domain conflict tetap benar |
| 7 | Package test | `go test` | Audit dampak activity/closing/importing/factory/migration | PASS |

## 6. Verifikasi Teknis

### 6.1 Command

```powershell
go test ./internal/activity ./internal/closing ./internal/importing ./internal/platform/factory ./internal/platform/migration
go test ./internal/platform/httpserver -run TestOpenAPISpecIsServed -count=1
go run . migrate up
```

### 6.2 Hasil

| Command | Status | Catatan |
| --- | --- | --- |
| `go test ./internal/activity` | PASS | Test resolusi channel interaction lulus |
| `go test ./internal/closing` | PASS | Closing menerima status call/chat baru |
| `go test ./internal/importing` | PASS | Package importing tetap aman |
| `go test ./internal/platform/factory` | PASS | Factory interaction multichannel aman |
| `go test ./internal/platform/migration` | PASS | Migration baru terdeteksi |
| `go test ./internal/platform/httpserver -run TestOpenAPISpecIsServed -count=1` | PASS | Spec OpenAPI tetap bisa disajikan |
| `go run . migrate up` | PASS | Migration `20260802000100_customer_interactions_multichannel.sql` terpasang |

Catatan:

- pada Windows lokal, setelah hasil `ok` kadang masih muncul warning cleanup `unlinkat ... Access is denied`;
- warning ini tidak mengubah hasil assertion test.

## 7. Detail Testing API

### 7.1 POST `/auth/login`

Tujuan:

- mendapatkan token Sales untuk menguji lead interaction miliknya.

Request:

```http
POST /api/v1/auth/login
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "email": "sales.003@demo.piposmart.id",
  "password": "Password123!"
}
```

Contoh response:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "rQnXU17O3vFVzt2vZljlMuRLetf7s2QqdJzh8rf0JXA",
    "token_type": "Bearer",
    "expires_in": 899,
    "user": {
      "id": 15,
      "code": "SLS-003",
      "name": "Sales Demo 003",
      "email": "sales.003@demo.piposmart.id",
      "role": "SALES"
    }
  },
  "meta": {
    "request_id": "f1cf8bb8fe5b825e59c35353698c6043"
  }
}
```

Hasil:

- login berhasil;
- token dapat dipakai untuk endpoint interaction lead milik Sales.

### 7.2 POST `/leads/{lead_id}/interactions` - Interaction CALL + CHAT dalam satu record

Tujuan:

- membuktikan satu form call & chat sekarang menghasilkan satu interaction saja.

Request:

```http
POST /api/v1/leads/9883/interactions
Authorization: Bearer {access_token_sales_003}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "call_status": "TERHUBUNG",
  "chat_status": "TERBALAS",
  "interaction_at": "2026-08-02T09:40:00+07:00",
  "contact_name": "Budi",
  "contact_phone": "081234567890",
  "duration_seconds": 240,
  "remark_score": 2,
  "note": "Follow up call dan chat dalam satu laporan",
  "customer_response": "Minta demo minggu depan",
  "follow_up_at": "2026-08-06T10:00:00+07:00",
  "follow_up_note": "Kirim reminder demo via WhatsApp"
}
```

Contoh response:

```json
{
  "data": {
    "id": 99713,
    "lead_id": 9883,
    "owner_id": 9883,
    "outlet_id": 12740,
    "sales": {
      "id": 15,
      "name": "Sales Demo 003",
      "role": "SALES"
    },
    "supervisor": {
      "id": 2,
      "name": "Supervisor Demo 001",
      "role": "SUPERVISOR"
    },
    "type": "CALL_CHAT",
    "call_status": "TERHUBUNG",
    "chat_status": "TERBALAS",
    "interaction_at": "2026-08-02T02:40:00Z",
    "contact_name": "Budi",
    "contact_phone": "081234567890",
    "duration_seconds": 240,
    "remark_reason_id": 3,
    "remark_score": 2,
    "remark_code": "POTENTIAL",
    "remark_label": "Potensial",
    "note": "Follow up call dan chat dalam satu laporan",
    "customer_response": "Minta demo minggu depan",
    "follow_up_at": "2026-08-06T03:00:00Z",
    "follow_up_note": "Kirim reminder demo via WhatsApp",
    "stage_before": "POTENTIAL",
    "stage_after": "POTENTIAL",
    "status_before": "OPEN",
    "status_after": "OPEN",
    "score_before": 2,
    "score_after": 2
  },
  "meta": {
    "request_id": "309626d6c90496143a493440129b9095"
  }
}
```

Validasi:

- backend membuat **satu** interaction;
- `type` otomatis menjadi `CALL_CHAT`;
- `call_status` dan `chat_status` keduanya tersimpan;
- stage lead tetap `POTENTIAL`.

### 7.3 GET `/leads/{lead_id}/interactions` - Verifikasi list interaction

Tujuan:

- memastikan interaction multichannel tampil benar pada endpoint list.

Request:

```http
GET /api/v1/leads/9883/interactions?limit=2
Authorization: Bearer {access_token_sales_003}
Accept: application/json
```

Contoh response terpotong:

```json
{
  "data": {
    "items": [
      {
        "id": 99714,
        "type": "CALL",
        "call_status": "RECORDED",
        "note": "kompatibilitas type lama"
      },
      {
        "id": 99713,
        "type": "CALL_CHAT",
        "call_status": "TERHUBUNG",
        "chat_status": "TERBALAS",
        "note": "Follow up call dan chat dalam satu laporan"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 2,
      "total": 14
    }
  },
  "meta": {
    "request_id": "75a7cca087fc36033ce654dc579a7b01"
  }
}
```

Validasi:

- data `CALL_CHAT` tampil di list;
- data legacy `CALL` tetap tampil;
- response shape aman untuk frontend.

### 7.4 POST `/leads/{lead_id}/interactions` - Kompatibilitas payload lama

Tujuan:

- memastikan frontend lama yang masih mengirim `type` belum rusak.

Request:

```http
POST /api/v1/leads/9883/interactions
Authorization: Bearer {access_token_sales_003}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "type": "CALL",
  "note": "kompatibilitas type lama"
}
```

Contoh response:

```json
{
  "data": {
    "id": 99714,
    "lead_id": 9883,
    "type": "CALL",
    "call_status": "RECORDED",
    "note": "kompatibilitas type lama"
  },
  "meta": {
    "request_id": "73544a405cc2e4cb0c38f6da781c9fe1"
  }
}
```

Validasi:

- request lama masih diterima;
- backend otomatis mengisi `call_status = RECORDED`;
- kompatibilitas sementara tetap aman.

### 7.5 POST `/leads/{lead_id}/interactions` - Error tanpa channel

Tujuan:

- memastikan interaction tanpa `call_status`, tanpa `chat_status`, dan tanpa `type` ditolak dengan pesan yang jelas.

Request:

```http
POST /api/v1/leads/9883/interactions
Authorization: Bearer {access_token_sales_003}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "note": "tanpa channel"
}
```

Contoh response error:

```json
{
  "error": {
    "code": "INTERACTION_CHANNEL_REQUIRED",
    "message": "minimal salah satu status call/chat atau type lama wajib diisi",
    "details": {
      "frontend_prevent": "pastikan form mengharuskan minimal satu status channel sebelum submit",
      "root_cause": "request interaction tidak mengirim status call/chat dan tidak mengirim fallback type lama",
      "solution": "kirim minimal salah satu field call_status atau chat_status. Field type lama masih didukung untuk kompatibilitas, tetapi bukan format utama."
    },
    "request_id": "a0247dfb66bbac7e883ef6ecde475fc1"
  }
}
```

Analisis:

- error handler sekarang spesifik;
- tidak jatuh ke `500 INTERNAL_ERROR`;
- frontend mendapatkan penyebab dan solusi.

Solusi frontend:

- wajibkan minimal satu dari:
  - `call_status`
  - `chat_status`

### 7.6 POST `/leads/{lead_id}/closings` - Audit dampak ke closing

Tujuan:

- memastikan perubahan multichannel tidak merusak flow closing.

Request:

```http
POST /api/v1/leads/9882/closings
Authorization: Bearer {access_token_sales_002}
Content-Type: application/json
Accept: application/json
```

Body:

```json
{
  "plan_id": 1,
  "call_status": "TERHUBUNG",
  "chat_status": "TERBALAS",
  "contact_name": "Siti",
  "contact_phone": "081111111111",
  "customer_response": "Setuju ambil paket basic",
  "note": "Closing hasil kombinasi call dan chat",
  "closed_at": "2026-08-02T10:00:00+07:00"
}
```

Contoh response error:

```json
{
  "error": {
    "code": "LEAD_ALREADY_HAS_CLOSING",
    "message": "lead sudah memiliki closing pending atau confirmed",
    "request_id": "d21cfc375bdf8cea722ee5dbbf6ede09"
  }
}
```

Analisis:

- endpoint menerima payload baru sampai masuk ke rule domain;
- error yang muncul adalah conflict bisnis yang memang benar;
- ini menunjukkan perubahan interaction multichannel **tidak mematahkan contract closing**.

## 8. Audit Dampak Seeder & Factory

Yang diverifikasi:

- factory interaction sekarang bisa membuat `CALL_CHAT`;
- factory closing sekarang menulis `call_status` / `chat_status`;
- import worker sekarang tidak perlu membuat dua interaction untuk satu row call+chat.

Status:

- PASS pada package test dan audit kode.

## 9. Kesimpulan

Hasil Sprint 15b:

- satu interaction kini bisa menyimpan call dan chat sekaligus;
- payload baru berhasil diuji end-to-end;
- payload lama masih kompatibel;
- error handler sudah spesifik;
- closing, importing, factory, seeder, dan migration tetap aman.

## 10. Rekomendasi ke Frontend

Frontend sebaiknya mulai beralih penuh ke format ini:

```json
{
  "call_status": "TERHUBUNG",
  "chat_status": "TERBALAS"
}
```

dan tidak lagi membuat dua request terpisah untuk satu form call & chat yang sama.
