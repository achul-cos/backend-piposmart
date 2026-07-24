# Sprint 11 - Partner, PIC Assignment, Referral, dan Call Interaction

## Sprint

Sprint 11

## Periode

24 Juli 2026

## Status

`GREEN`

Sprint Goal tercapai: Pengelolaan tipe partner, data partner dengan rekening bank terenkripsi (AES-GCM) & response masked, penugasan PIC (Person In Charge) aktif dengan aturan single active assignment, pencatatan interaksi (CALL/CHAT) mitra, dan rujukan partner ke customer lead (referral) dengan pencegahan duplikat.

## Sprint Goal

Pengelolaan data partner, penugasan PIC aktif, interaksi panggilan mitra, dan referral lead dari partner berjalan aman, konsisten, serta bebas double counting.

## Committed Deliverables

- Migrasi database `20260724000900_partner_pic_referral.sql`.
- CRUD Partner Type & Partner.
- Enkripsi nomor rekening partner (AES-GCM) & response masked (`****1234`).
- Penugasan PIC partner (single active assignment invariant).
- Pelepasan PIC partner (release assignment).
- Pencatatan interaksi mitra (CALL dan CHAT).
- Referrals dari partner ke customer lead (unique pair invariant).
- Factory partner (`internal/platform/factory/partner.go`).
- Seeder master partner types & demo partner data.
- Unit test package `internal/partner`.
- OpenAPI diperbarui ke `0.11.0-sprint-11`.
- API Testing Report Sprint 11.

## Completed

- [x] Migration `partners`, `partner_assignments`, `partner_interactions`, dan `partner_referrals`.
- [x] API list/detail/create/update/deactivate partner.
- [x] API list/create/update partner type.
- [x] Rekening bank terenkripsi AES-GCM di DB (`bank_account_encrypted`) dan masked di JSON (`bank_account_masked`).
- [x] Rekening bank bernilai `null` bila tidak diisi (tidak mengembalikan string kosong).
- [x] API assign PIC ke partner dan otomatis mendegradasi penugasan aktif sebelumnya.
- [x] API release PIC dari partner.
- [x] API list & record interaksi partner (CALL / CHAT).
- [x] API list & create referral partner ke customer lead.
- [x] Invariant unique referral: mencegah duplikat partner-lead referral (mengembalikan status `409 CONFLICT`).
- [x] Otomatisasi `assigned_by_id` dari JWT context `identity.CurrentUser(c)`.
- [x] Factory partner type, partner, assignment, interaction, dan referral.
- [x] Seeder master partner types dan demo partner data.
- [x] OpenAPI diperbarui ke `0.11.0-sprint-11`.
- [x] API Testing Report Sprint 11 dibuat.

## Not Completed / Carry Over

Tidak ada carry over blocker untuk scope Sprint 11.

Catatan teknis untuk sprint berikutnya:
- Pada Sprint 12 (Komisi), data referral confirmed dan partner assignment aktif akan menjadi dasar perhitungan komisi mitra.
- Pada Sprint 13 (KPI), interaksi CALL/CHAT partner dapat dimasukkan sebagai indikator keaktifan penanganan partner oleh PIC (Sales/Supervisor).

## Demo Evidence

Smoke test API lokal dijalankan pada port `8080`.

| Area | Evidence | Result |
| --- | --- | --- |
| Auth | Login Admin & Sales dummy | PASS |
| Partner Type | `GET /api/v1/partner-types` | PASS |
| Create Partner | `POST /api/v1/partners` dengan `bank_account` | PASS |
| Masked Bank | Response `bank_account_masked = "****5678"`, `bank_account_encrypted` hidden | PASS |
| List Partner | `GET /api/v1/partners?limit=5` | PASS |
| Detail Partner | `GET /api/v1/partners/1` | PASS |
| Update Partner | `PUT /api/v1/partners/1` | PASS |
| Assign PIC | `POST /api/v1/partners/1/assignments` dengan Sales user | PASS |
| Active PIC | `GET /api/v1/partners/1/assignments/active` | PASS |
| Re-assign PIC | Assign user baru ke partner 1 -> assignment lama non-aktif | PASS |
| Release PIC | `DELETE /api/v1/partners/1/assignments/release` | PASS |
| Interaction | `POST /api/v1/partners/1/interactions` type `CALL` | PASS |
| List Interaction| `GET /api/v1/partners/1/interactions` | PASS |
| Create Referral| `POST /api/v1/partners/1/referrals` dengan `lead_id = 1` | PASS |
| Duplicate Ref | Repeat create referral sama -> `409 DUPLICATE_REFERRAL` | PASS |
| Error Handling | Tanpa JWT token -> `401 UNAUTHENTICATED` | PASS |
| Error Handling | Invalid ID parameter -> `400 VALIDATION_ERROR` | PASS |
| Error Handling | Partner not found -> `404 NOT_FOUND` | PASS |

Dokumen API Testing detail:
- `docs/sprint-11/README.md`

## Seeder Evidence

Command seed demo:

```powershell
go run . seed master
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
```

Data demo Sprint 11:

| Scenario | Partner | Alur | Expected |
| --- | --- | --- | --- |
| Supplier POS | SUP-001 (PT Hardware Maju POS) | Terdaftar dengan rekening bank encrypted & last4 5678, diassign ke SPV-001, terdapat catatan CALL interaksi penawaran bundle. | Rekening masked `****5678`, PIC aktif SPV-001. |
| Distributor Software | DIS-001 (CV Digital Software Solution) | Terdaftar dengan rekening last4 1234, diassign ke SLS-001. | PIC aktif SLS-001, penugasan tercatat rapi. |
| Referral Community | REF-001 (Komunitas UMKM Kopi) | Merujuk lead Owner Kopi Kenangan (LEAD-000001). | Referral terhubung ke lead, duplicate attempt ditolak. |

## Quality

| Quality Gate | Result | Catatan |
| --- | --- | --- |
| Migration up | PASS | Database berhasil naik sampai `20260724000900_partner_pic_referral.sql`. |
| Seeder master/demo | PASS | `seed master` dan `seed demo minimal` berhasil. |
| `go build ./...` | PASS | Seluruh package dapat di-compile tanpa error. |
| `go vet ./...` | PASS | Tidak ada error static analysis. |
| `go test ./internal/partner/...` | PASS | Unit test partner package 100% PASS. |
| OpenAPI updated | PASS | Version `0.11.0-sprint-11`, route & schema Sprint 11 ditambahkan. |
| API smoke test | PASS | Success & error cases terdokumentasi di `docs/sprint-11/README.md`. |

## Defect Found During Testing

| Defect | Dampak | Root Cause | Fix | Status |
| --- | --- | --- | --- | --- |
| Type mismatch pada `p.BankAccountEncrypted` (string vs []byte) dan `p.Phone` (`sql.NullString` vs `*string`). | Kompilasi package `internal/partner` gagal. | Tipe data pada struct internal belum dikonversi secara tepat saat penyusunan `PartnerResponse`. | Menggunakan helper `NewPartnerResponse` dan melakukan cast `[]byte(enc)` secara aman. | CLOSED |
| `BankAccountMasked` mengembalikan pointer ke string kosong `&""` jika rekening tidak diisi. | JSON response menampilkan `"bank_account_masked": ""` bukan `null`. | Mask helper tidak memeriksa keabsahan `BankAccountLast4.Valid`. | Membuat helper `maskedAccountPtr` yang mengembalikan `nil` jika `BankAccountLast4` tidak valid. | CLOSED |

## Defect Terbuka

Tidak ada defect blocker atau critical untuk scope Sprint 11.

## Impediments

Tidak ada impediment teknis pada sesi ini.

## Rencana Sprint Berikutnya

Sprint 12 - Partner Commission & Earning Activation.
Fokus: Skema komisi mitra, integrasi reconciliation closing confirmed, debit/credit wallet komisi, serta rincian laporan earning partner.
