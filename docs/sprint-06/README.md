# API Testing Report - Sprint 06

## Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 06 - Call Customer, Remark, Follow-up, dan Training |
| Tanggal Testing | 23 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:18083/api/v1` saat smoke test; default project mengikuti `APP_PORT` |
| Testing Tool | Manual smoke test via PowerShell |
| Migration Version | `20260723000500` |

## Testing Summary

| Module | Total Case | Passed | Failed | Status |
| --- | ---: | ---: | ---: | --- |
| Customer Interaction & Remark | 6 | 6 | 0 | PASS |
| Follow-up | 1 | 1 | 0 | PASS |
| Training | 3 | 3 | 0 | PASS |
| Visibility & History | 2 | 2 | 0 | PASS |
| **Total** | **12** | **12** | **0** | **PASS** |

## Header

```http
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json
```

## Endpoint Utama

| Nama Route | Method | Path | Fungsi |
| --- | --- | --- | --- |
| List Interaction | GET | `/customer-interactions` | List call/chat sesuai visibility actor. |
| List Follow-up | GET | `/follow-ups` | List interaksi yang memiliki jadwal follow-up. |
| List Lead Interaction | GET | `/leads/{lead_id}/interactions` | List interaksi pada satu lead. |
| Create Interaction | POST | `/leads/{lead_id}/interactions` | Mencatat call/chat, remark, dan follow-up. |
| Stage History | GET | `/leads/{lead_id}/stage-history` | Riwayat perubahan stage/score. |
| List Training | GET | `/trainings` | List training sesuai visibility actor. |
| Detail Training | GET | `/trainings/{training_id}` | Detail satu training sesuai visibility actor. |
| Schedule Training | POST | `/leads/{lead_id}/trainings` | Menjadwalkan training. |
| Reschedule Training | POST | `/trainings/{training_id}/reschedule` | Mengubah jadwal training. |
| Complete Training | POST | `/trainings/{training_id}/complete` | Menyelesaikan training. |
| Cancel Training | POST | `/trainings/{training_id}/cancel` | Membatalkan training. |

## Smoke Test Matrix

| Case | Result |
| --- | --- |
| Login Admin/Supervisor/Sales | PASS |
| Lead dibuat dan diberikan ke Sales | PASS |
| Remark 2 mengubah lead menjadi `POTENTIAL` | PASS |
| Remark 1 tidak menurunkan `POTENTIAL` | PASS |
| Follow-up schedule dapat difilter | PASS |
| Training berhasil dijadwalkan | PASS |
| Training berhasil di-reschedule | PASS |
| Training berhasil diselesaikan | PASS |
| Remark 3 mencatat stage `CLOSING` sementara | PASS |
| Remark 0 membuat `INVALID` dan ownership kembali ke Supervisor | PASS |
| Sales kehilangan visibility setelah remark 0 | PASS |
| Stage history tercatat | PASS |

## Contoh Request

### POST `/leads/{lead_id}/interactions`

```http
POST /api/v1/leads/24/interactions
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "type": "CALL",
  "interaction_at": "2026-07-23T10:00:00+07:00",
  "remark_score": 2,
  "note": "Customer tertarik demo",
  "customer_response": "Minta dijadwalkan training",
  "follow_up_at": "2026-07-28T10:00:00+07:00",
  "follow_up_note": "Follow-up jadwal training"
}
```

Response `201 Created`:

```json
{
  "data": {
    "id": 13,
    "lead_id": 24,
    "type": "CALL",
    "remark_score": 2,
    "remark_code": "POTENTIAL",
    "stage_before": "POSSIBLE",
    "stage_after": "POTENTIAL",
    "follow_up_at": "2026-07-28T03:00:00Z"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### GET `/follow-ups`

```http
GET /api/v1/follow-ups?follow_up_from=2026-07-28&follow_up_to=2026-07-30&limit=10
Authorization: Bearer {sales_access_token}
Accept: application/json
```

Response `200 OK`: list interaksi yang memiliki `follow_up_at`.

### POST `/leads/{lead_id}/trainings`

```http
POST /api/v1/leads/24/trainings
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "training_type": "ONLINE",
  "scheduled_at": "2026-07-30T13:00:00+07:00",
  "meeting_url": "https://meet.example.test/sprint-06",
  "note": "Demo aplikasi Piposmart"
}
```

Response `201 Created`: training berstatus `SCHEDULED`.

Catatan:

- `trainer_name` dan `participant_name` tidak lagi menjadi field request API training.
- Identitas owner/lead dikembalikan dari backend melalui `owner_*` dan `lead_code`.

### GET `/trainings/{training_id}`

```http
GET /api/v1/trainings/4
Authorization: Bearer {sales_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "id": 4,
    "lead_id": 24,
    "lead_code": "OWN-00024-LEAD-01",
    "owner_id": 24,
    "owner_code": "OWN-00024",
    "owner_name": "Owner Laundry 024",
    "training_type": "ONLINE",
    "status": "SCHEDULED",
    "scheduled_at": "2026-07-30T06:00:00Z",
    "meeting_url": "https://meet.example.test/sprint-06",
    "note": "Demo aplikasi Piposmart"
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### POST `/trainings/{training_id}/complete`

```json
{
  "completed_at": "2026-07-31T15:30:00+07:00",
  "result_note": "Owner memahami penggunaan kasir dan outlet"
}
```

Response `200 OK`: training berstatus `COMPLETED`.

## Contoh Error

### Remark Score Tidak Valid

```http
POST /api/v1/leads/24/interactions
Authorization: Bearer {sales_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "type": "CALL",
  "remark_score": 9
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SCORE",
    "message": "score remark harus 0 sampai 3",
    "request_id": "generated-request-id"
  }
}
```

### Training yang Sudah Completed Di-complete Ulang

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_TRANSITION",
    "message": "transisi status tidak valid",
    "request_id": "generated-request-id"
  }
}
```

## Conclusion

Sprint 06 API untuk call/chat customer, remark policy, follow-up, stage history,
dan training berjalan sesuai briefing dan smoke test.

**Overall API Testing Status:** `PASS`
