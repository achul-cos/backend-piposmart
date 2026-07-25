# Glosarium Teknis — Backend CRM Piposmart

Dokumen ini menjelaskan istilah-istilah teknis yang dipakai di seluruh backend CRM Piposmart.
Setiap istilah dijelaskan dalam dua lapis:

- **Penjelasan teknis** — apa artinya secara tepat.
- 🔑 **Analogi** — perumpamaan sederhana supaya mudah dibayangkan.
- 📍 **Di project ini** — di mana istilah itu benar-benar muncul, supaya bisa langsung dicari.

Daftar isi:

1. [Bahasa & Tooling Dasar](#1-bahasa--tooling-dasar)
2. [Arsitektur & Pola Kode](#2-arsitektur--pola-kode)
3. [Database — Konsep Umum](#3-database--konsep-umum)
4. [Database — Konkurensi & Konsistensi](#4-database--konkurensi--konsistensi)
5. [Keamanan & Autentikasi](#5-keamanan--autentikasi)
6. [HTTP & API](#6-http--api)
7. [Uang & Finansial](#7-uang--finansial)
8. [Background Processing (Worker & Job)](#8-background-processing-worker--job)
9. [Data Dummy & Testing](#9-data-dummy--testing)
10. [Istilah Domain Bisnis Piposmart](#10-istilah-domain-bisnis-piposmart)

---

## 1. Bahasa & Tooling Dasar

### Go (Golang)
Bahasa pemrograman yang dipakai untuk menulis seluruh backend ini. Dikompilasi menjadi satu file
binary (`.exe` di Windows), cepat, dan dirancang untuk server.

🔑 **Analogi**: Kalau backend adalah sebuah restoran, Go adalah bahasa yang dipakai seluruh koki
untuk menulis resep. Semua "resep" (kode) ditulis dengan tata bahasa Go.

📍 File berakhiran `.go`, dijalankan dengan `go run .`.

### Gin
*Framework* (kerangka kerja) untuk membuat HTTP API di Go. Gin yang menerima request dari internet,
mencocokkannya ke fungsi yang tepat (routing), lalu mengembalikan response.

🔑 **Analogi**: Gin itu seperti **resepsionis + satpam** di lobi kantor. Saat tamu (request) datang
dan bilang "saya mau ke bagian Komisi", Gin yang mengarahkan ke ruangan yang benar, sekaligus
mengecek apakah tamu itu boleh masuk.

📍 Terlihat di semua file `handler.go` (`*gin.Context`, `rg.Group(...)`).

### GORM
*ORM* (Object-Relational Mapping) untuk Go — pustaka yang menjembatani antara kode Go dan tabel
database, tapi di project ini pemakaiannya minimal (kebanyakan query ditulis SQL langsung).

🔑 **Analogi**: Penerjemah antara "bahasa kode" dan "bahasa database". Di project ini penerjemah ini
lebih sering dilewati — kita bicara langsung ke database pakai SQL mentah karena lebih terkontrol.

### MySQL
Sistem database yang menyimpan seluruh data (user, owner, closing, komisi, dll). Versi yang dipakai
adalah **MySQL 8**. Datanya tersimpan dalam bentuk **tabel** (mirip spreadsheet raksasa).

🔑 **Analogi**: Lemari arsip perusahaan. Setiap laci = satu tabel; setiap map di dalam laci = satu
baris data.

### Goose
Alat untuk menjalankan **migration** database (lihat [Migration](#migration)). Goose yang mencatat
"struktur database sudah sampai versi berapa" dan menjalankan perubahan yang belum diterapkan.

🔑 **Analogi**: Mandor renovasi yang punya buku catatan. Dia tahu persis renovasi mana yang sudah
dikerjakan dan mana yang belum, jadi tidak pernah mengerjakan hal yang sama dua kali.

📍 `go run . migrate up`, folder `migrations/`.

### OpenAPI / Swagger
**OpenAPI** adalah format standar untuk mendeskripsikan seluruh endpoint API dalam satu file
(`openapi.yaml`) — endpoint apa saja, parameter apa, response-nya seperti apa. **Swagger** adalah
halaman web interaktif yang membaca file itu dan menampilkannya sebagai dokumentasi yang bisa
langsung dicoba.

🔑 **Analogi**: OpenAPI itu **buku menu restoran** yang lengkap (semua hidangan + bahan + harga).
Swagger itu **menu digital di layar** yang bisa disentuh untuk langsung memesan.

📍 `internal/platform/httpserver/openapi.yaml`, dibuka di `/swagger/index.html`.

---

## 2. Arsitektur & Pola Kode

### Layered Architecture (Handler → Service → Repository)
Setiap modul dipecah menjadi 3 lapisan dengan tugas jelas, dan setiap lapisan hanya bicara ke
lapisan di bawahnya:

1. **Handler** — menerima request HTTP, membaca input, memanggil Service, mengembalikan response.
   Tidak berisi logika bisnis.
2. **Service** — otak/aturan bisnis. Cek izin (role), validasi, urutan langkah. Tidak menyentuh
   SQL langsung.
3. **Repository** — satu-satunya yang bicara ke database (menulis SQL).

🔑 **Analogi**: Restoran.
- **Handler** = pelayan yang mencatat pesanan dan mengantar makanan.
- **Service** = kepala dapur yang memutuskan urutan masak dan mengecek apakah pesanan valid.
- **Repository** = koki yang benar-benar mengambil bahan dari gudang (database).

Pelayan tidak masuk gudang; koki tidak melayani pelanggan. Tiap orang punya perannya, jadi kalau ada
masalah, mudah tahu di mana.

📍 Tiap modul di `internal/*` punya `handler.go`, `service.go`, `repository.go`.

### Domain Module (Modul Domain)
Satu folder di `internal/` yang menangani satu area bisnis secara utuh. Contoh: `internal/partner`
mengurus mitra, `internal/kpi` mengurus penilaian performa. Masing-masing mandiri.

🔑 **Analogi**: Departemen di perusahaan. Ada departemen HRD, Keuangan, Marketing — masing-masing
punya ruangan sendiri, staf sendiri, dan tanggung jawab sendiri, tapi bisa saling berkoordinasi.

📍 `activity`, `catalog`, `closing`, `customer`, `identity`, `kpi`, `lead`, `partner`,
`subscription`, `target`, `wallet`.

### Platform Package (Paket Infrastruktur)
Berbeda dari modul domain, `internal/platform/*` berisi kode "fondasi" yang dipakai bersama oleh
semua modul: koneksi database, logging, konfigurasi, job queue, dll. Bukan area bisnis, tapi
peralatan bersama.

🔑 **Analogi**: Fasilitas gedung — listrik, AC, lift, wifi. Semua departemen memakainya, tapi bukan
milik departemen tertentu.

📍 `internal/platform/config`, `database`, `httpserver`, `jobqueue`, `logging`, dll.

### DTO (Data Transfer Object) — Request & Response struct
Struktur data yang khusus dibuat untuk "bentuk" data yang masuk (Request) atau keluar (Response) di
API. Berbeda dari struktur data internal supaya kita bisa **menyembunyikan** field sensitif dan
mengontrol persis apa yang publik lihat.

🔑 **Analogi**: Formulir. `CreatePartnerRequest` adalah formulir isian pendaftaran mitra;
`PartnerResponse` adalah surat balasan resmi. Data internal (misal nomor rekening lengkap) tidak
ikut di surat balasan — hanya 4 digit terakhir.

📍 `types.go` di tiap modul, misal `CreateKpiDefinitionRequest`, `SalesKpiResultResponse`.

### Response Envelope (Amplop Response)
Semua response API dibungkus format seragam: sukses selalu `{ "data": ..., "meta": {...} }`, error
selalu `{ "error": { "code", "message", "request_id" } }`. Jadi frontend selalu tahu di mana
mencari isinya.

🔑 **Analogi**: Amplop surat berkop perusahaan. Apa pun isi suratnya, formatnya selalu sama: ada
kop, ada isi, ada nomor surat. Penerima tidak perlu menebak-nebak.

📍 `internal/platform/httpx/response.go` (`httpx.Success`, `httpx.Error`).

### Middleware
Kode yang "menyisip" di antara request masuk dan handler, untuk memproses sesuatu lebih dulu —
misal mengecek token login, mencatat log, atau menempelkan request ID.

🔑 **Analogi**: Pos pemeriksaan sebelum masuk gedung. Sebelum tamu sampai ke ruangan tujuan, dia
lewat pemeriksaan ID dulu. Kalau gagal, tidak usah lanjut.

📍 `identity.AuthMiddleware(...)` dipasang di `router.go` untuk route yang butuh login.

### Handler / Service / Repository — kenapa dipisah?
Supaya **satu perubahan tidak merembet ke mana-mana**. Kalau cara menyimpan data berubah, cukup ubah
Repository. Kalau aturan bisnis berubah, cukup ubah Service. Kalau format API berubah, cukup ubah
Handler.

🔑 **Analogi**: Rumah dengan sekring terpisah per ruangan. Kalau korslet di dapur, lampu kamar tidak
ikut mati.

---

## 3. Database — Konsep Umum

### Migration
File berisi perintah SQL untuk **mengubah struktur** database (membuat tabel, menambah kolom, dll).
Setiap migration punya bagian **Up** (menerapkan perubahan) dan **Down** (membatalkannya). Dijalankan
berurutan sesuai nomor.

🔑 **Analogi**: Instruksi renovasi bertahap. "Langkah 5: bangun tembok baru" (Up). Kalau salah,
ada juga "Langkah 5 dibatalkan: robohkan tembok itu" (Down). Buku catatan mandor (Goose) memastikan
tiap langkah cuma dikerjakan sekali.

📍 Folder `migrations/`, contoh `20260725000200_sales_target_kpi_ranking.sql`.

### Reversible Migration (Migration yang bisa dibalik)
Migration yang bagian Down-nya benar-benar mengembalikan keadaan seperti sebelum Up dijalankan. Diuji
dengan pola **up → down → up** untuk memastikan tidak ada yang rusak saat dibatalkan.

🔑 **Analogi**: Tombol "undo" yang benar-benar berfungsi. Kamu bisa maju, mundur, maju lagi, tanpa
data jadi kacau.

### Schema
Struktur/rancangan tabel database: nama tabel, kolom apa saja, tipe datanya, dan aturannya.

🔑 **Analogi**: Denah/blueprint bangunan. Sebelum barang (data) masuk, dendahnya sudah menentukan
ada ruang apa saja dan ukurannya berapa.

### Foreign Key (FK) — Kunci Asing
Kolom yang "menunjuk" ke baris di tabel lain, memastikan hubungan antar-data tetap valid. Misal
`sales_targets.sales_id` menunjuk ke `users.id` — tidak mungkin membuat target untuk user yang tidak
ada.

🔑 **Analogi**: Nomor KTP yang dicantumkan di formulir. Formulir kepesertaan menyebut nomor KTP; kalau
KTP-nya tidak terdaftar, formulir ditolak. FK memastikan "orang yang dirujuk memang ada".

📍 `CONSTRAINT fk_sales_targets_sales FOREIGN KEY (sales_id) REFERENCES users(id)`.

### Index
Struktur bantu yang membuat pencarian data jadi cepat, dengan "mengurutkan" data berdasarkan kolom
tertentu di belakang layar.

🔑 **Analogi**: Daftar isi + indeks di belakang buku. Tanpa indeks, mencari kata "komisi" berarti
membaca seluruh buku halaman per halaman. Dengan indeks, langsung loncat ke halaman yang tepat.

📍 `KEY idx_job_queue_dispatch (status, available_at)`.

### Unique Key / Unique Constraint (Kunci Unik)
Aturan yang melarang ada dua baris dengan nilai yang sama pada kolom tertentu. Database sendiri yang
menolak duplikat, bukan cuma kode aplikasi.

🔑 **Analogi**: Aturan "satu email = satu akun". Kalau ada yang coba daftar dengan email yang sudah
dipakai, sistem menolak di level paling dasar.

📍 `UNIQUE KEY uq_sales_targets_period (sales_id, metric_code_id, period_year, period_month)` —
memastikan satu Sales hanya punya satu target per metric per bulan.

### CHECK Constraint
Aturan yang memaksa nilai kolom memenuhi syarat tertentu, dijaga oleh database. Misal bulan harus
1–12, atau status harus salah satu dari daftar yang diizinkan.

🔑 **Analogi**: Kolom formulir yang hanya menerima jawaban tertentu. Kolom "bulan" tidak akan
menerima angka 13 — pulpennya menolak menulis.

📍 `CONSTRAINT chk_sales_targets_month CHECK (period_month BETWEEN 1 AND 12)`.

### Enum (Enumerasi)
Sekumpulan nilai yang sah dan terbatas untuk sebuah kolom/field. Contoh: status komisi hanya boleh
`PENDING`, `APPROVED`, `PAID`, atau `CANCELLED`.

🔑 **Analogi**: Pilihan ganda. Jawabannya harus salah satu dari A/B/C/D — tidak boleh jawaban bebas.

📍 Status payout: `PENDING | PAID | CANCELLED`.

### Soft Delete vs Hard Delete
- **Soft delete**: data tidak benar-benar dihapus, hanya ditandai "terhapus" (mengisi kolom
  `deleted_at`). Bisa dipulihkan (restore), dan tetap ada untuk audit.
- **Hard delete** (force): data dihapus permanen dari database.

🔑 **Analogi**: Soft delete = memindahkan file ke **Recycle Bin** (masih bisa dikembalikan). Hard
delete = **Shift+Delete** (hilang selamanya).

📍 Owner/outlet punya keduanya: `DELETE /owners/{id}` (soft) vs `DELETE /owners/{id}/force` (hard),
plus `/restore`.

### NULL / Nullable
`NULL` artinya "tidak ada nilai / belum diisi", berbeda dari 0 atau string kosong. Kolom yang boleh
kosong disebut *nullable*.

🔑 **Analogi**: Kolom "tanggal meninggal" di data kependudukan. Untuk yang masih hidup, kolomnya
NULL — bukan "tanggal 0", tapi memang belum/tidak ada.

📍 Di Go direpresentasikan `sql.NullString`, `sql.NullInt64`, dll. `paid_at DATETIME NULL`.

---

## 4. Database — Konkurensi & Konsistensi

Bagian ini penting karena berhubungan dengan uang dan performa — kesalahan di sini bisa berakibat
data ganda atau saldo salah.

### Transaction (Transaksi Database)
Sekelompok operasi database yang diperlakukan sebagai **satu kesatuan**: semua berhasil, atau semua
dibatalkan. Tidak ada keadaan setengah jadi.

🔑 **Analogi**: Transfer uang antar rekening. Uang keluar dari rekening A **dan** masuk ke rekening B
harus terjadi bersamaan. Kalau salah satu gagal, keduanya dibatalkan — mustahil uang "hilang di
tengah jalan".

📍 `tx, err := r.db.BeginTx(ctx, nil)` di banyak repository. Recompute KPI melakukan
delete-lalu-insert dalam satu transaksi.

### Atomic (Atomik)
Sifat "utuh, tak terbagi" — merujuk pada operasi yang tidak bisa terinterupsi di tengah. Biasanya
dicapai lewat transaksi. (Lihat Transaction di atas.)

🔑 **Analogi**: Menyalakan/mematikan saklar. Tidak ada posisi "setengah nyala". Entah nyala, entah
mati.

### Idempotent / Idempotency (Idempoten)
Sifat operasi yang **kalau dijalankan berkali-kali, hasilnya sama seperti dijalankan sekali**.
Penting untuk mencegah efek ganda (misal transfer terhitung dua kali karena tombol diklik dua kali).

🔑 **Analogi**: Tombol lift. Menekan tombol lantai 3 sekali atau lima kali, hasilnya sama: lift ke
lantai 3. Tidak jadi ke lantai 15.

📍 Sync komisi: closing yang sudah punya komisi tidak diproses ulang. Recompute KPI: dijalankan
ulang untuk bulan yang sama menghasilkan angka identik. Wallet top-up pakai `idempotency_key`.

### Row Lock (Penguncian Baris) — `FOR UPDATE`
Saat sebuah transaksi mengunci baris data tertentu, transaksi lain harus **menunggu** sampai kunci
dilepas sebelum boleh mengubah baris itu. Mencegah dua proses mengedit data yang sama bersamaan.

🔑 **Analogi**: Kamar pas di toko baju. Selama kamu di dalam dan menguncinya, orang lain harus
menunggu di luar. Setelah kamu keluar, baru giliran berikutnya masuk.

📍 `SELECT ... FOR UPDATE` di `AssignPIC` (partner) dan `CreatePayout`. Memastikan tidak ada dua PIC
aktif atau dua payout yang membatch komisi yang sama.

### `SKIP LOCKED`
Varian penguncian: kalau sebuah baris sedang dikunci proses lain, **jangan menunggu — lewati saja**
dan ambil baris berikutnya. Berguna untuk worker yang berebut ambil pekerjaan.

🔑 **Analogi**: Beberapa kurir mengambil paket dari rak. Kalau satu paket sedang dipegang kurir lain,
kurir ini tidak berdiri diam menunggu — dia langsung ambil paket lain. Hasilnya tidak ada dua kurir
membawa paket yang sama, dan semua kurir tetap sibuk.

📍 `internal/platform/jobqueue/repository.go` — `ClaimNext` memakai `FOR UPDATE SKIP LOCKED` supaya
banyak worker bisa mengambil job berbeda tanpa bentrok.

### Concurrency Guard (Penjaga Konkurensi)
Mekanisme untuk mencegah masalah saat banyak hal terjadi **bersamaan** (concurrent). Bisa berupa row
lock, unique constraint, atau kombinasi keduanya.

🔑 **Analogi**: Sistem antrean bank dengan nomor. Meski 50 orang datang bersamaan, sistem memastikan
hanya satu yang dilayani per teller, tanpa rebutan.

📍 Fix "satu PIC aktif per partner" di Sprint 11
(`20260724001100_partner_assignment_concurrency_guard.sql`).

### Generated Column (Kolom Terhitung) — VIRTUAL vs STORED
Kolom yang nilainya **otomatis dihitung** dari kolom lain, bukan diisi manual.
- **VIRTUAL**: dihitung saat dibaca (tidak makan tempat penyimpanan).
- **STORED**: dihitung saat ditulis lalu disimpan.

Dipakai untuk trik "hanya boleh satu baris aktif" — kolom terhitung yang bernilai NULL saat baris
tidak aktif, dikombinasikan dengan unique key.

🔑 **Analogi**: Kolom "total harga" di struk yang otomatis = harga × jumlah. Kasir tidak mengetiknya
manual; muncul sendiri. VIRTUAL = dihitung pas struk dicetak; STORED = sudah dicatat sejak awal.

📍 `active_commission_key ... VIRTUAL` di `partner_payout_items` — mengunci "satu komisi hanya boleh
di satu payout aktif".

> ⚠️ **Catatan penting proyek ini**: MySQL 8.0.46 menolak menambah kolom STORED lewat `ALTER TABLE`
> pada tabel yang punya foreign key (error 1215 yang menyesatkan). Solusinya pakai VIRTUAL. Ini
> pelajaran yang berulang di Sprint 11–12.

### Race Condition (Kondisi Balapan)
Bug yang muncul saat hasil program bergantung pada "siapa cepat" antara dua proses yang berjalan
bersamaan — dan urutannya tidak bisa dipastikan. Concurrency guard dibuat justru untuk mencegah ini.

🔑 **Analogi**: Dua orang menarik uang dari ATM rekening yang sama di dua mesin berbeda pada detik
yang sama. Tanpa pengaman, saldo bisa terpotong salah. "Balapan" siapa yang datanya tersimpan
duluan menentukan hasil — dan itu berbahaya.

---

## 5. Keamanan & Autentikasi

### Autentikasi vs Otorisasi
- **Autentikasi** (authentication): membuktikan **siapa kamu** (login).
- **Otorisasi** (authorization): menentukan **apa yang boleh kamu lakukan** (izin).

🔑 **Analogi**: Autentikasi = menunjukkan KTP di lobi (membuktikan kamu Budi). Otorisasi = kartu
akses yang menentukan lantai mana saja yang boleh kamu masuki (Budi hanya boleh ke lantai 3).

### JWT (JSON Web Token)
Token/tiket digital yang dikirim server saat login, berisi identitas user yang sudah
"ditandatangani" secara kriptografis sehingga tidak bisa dipalsukan. Dikirim di tiap request untuk
membuktikan kita sudah login.

🔑 **Analogi**: Gelang festival yang tidak bisa dipalsu. Setelah masuk (login), kamu dapat gelang.
Selama gelang menempel, kamu tidak perlu tunjukkan tiket lagi di tiap wahana — cukup tunjukkan
gelang.

📍 Dikirim di header `Authorization: Bearer <token>`.

### Access Token vs Refresh Token
- **Access token**: tiket berumur pendek (di project ini ±15 menit) untuk mengakses API.
- **Refresh token**: tiket berumur panjang untuk **menukar** access token baru saat yang lama
  kedaluwarsa, tanpa perlu login ulang.

🔑 **Analogi**: Access token = tiket parkir 15 menit. Refresh token = kartu member yang bisa dipakai
minta tiket parkir baru berulang kali tanpa mendaftar ulang.

### Token Rotation (Rotasi Token)
Setiap kali refresh token dipakai, ia langsung **hangus** dan diganti yang baru. Jadi kalau ada yang
mencuri refresh token lama, token itu sudah tidak berlaku.

🔑 **Analogi**: Password sekali pakai (OTP). Begitu dipakai, otomatis kedaluwarsa. Pencuri yang dapat
kode lama tidak bisa apa-apa.

📍 Sprint 3, `/api/v1/auth/refresh`.

### Argon2id / Password Hashing
**Hashing** adalah mengubah password menjadi kode acak searah yang **tidak bisa dikembalikan** ke
password asli. **Argon2id** adalah algoritma hashing modern yang sengaja dibuat lambat supaya sulit
ditebak paksa. Database tidak pernah menyimpan password asli — hanya hash-nya.

🔑 **Analogi**: Menggiling daging menjadi bakso. Dari daging bisa jadi bakso, tapi dari bakso mustahil
dikembalikan jadi potongan daging semula. Saat login, sistem menggiling password yang kamu ketik lalu
membandingkan bakso-nya, bukan dagingnya.

📍 `internal/platform/password`.

### Hash
Hasil dari proses hashing — kode dengan panjang tetap yang mewakili data asli. Selain untuk password,
dipakai juga untuk mendeteksi file duplikat (SHA-256 di import, Sprint 14).

🔑 **Analogi**: Sidik jari. Setiap orang menghasilkan sidik jari unik; dari sidik jari kamu tidak
bisa merekonstruksi orangnya, tapi bisa memastikan "ini orang yang sama".

### RBAC (Role-Based Access Control)
Sistem pengaturan izin berdasarkan **peran** (role). Setiap user punya role (`ADMIN`, `SUPERVISOR`,
`SALES`), dan role menentukan apa yang boleh diakses.

🔑 **Analogi**: Kartu akses karyawan berdasarkan jabatan. Manajer bisa masuk ruang server; staf biasa
tidak. Yang menentukan bukan nama orangnya, tapi jabatannya.

📍 Cek `actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor` di banyak service.

### Permission
Izin spesifik yang lebih halus dari role. Misal `owners.manage`, `users.manage_sales`. Satu role
bisa punya banyak permission.

🔑 **Analogi**: Rincian di kartu akses. Selain "level Manajer", kartu bisa mencantumkan izin detail:
"boleh buka gudang", "boleh approve cuti".

### Bearer Token
Cara mengirim token di header HTTP: `Authorization: Bearer <token>`. Kata "Bearer" berarti "pembawa"
— siapa pun yang membawa token ini dianggap sah (karena itu token harus dijaga).

🔑 **Analogi**: Tiket bioskop tanpa nama. Siapa pun yang memegang tiket itu boleh masuk. Maka jangan
sampai jatuh ke tangan orang lain.

### Encrypted at Rest (Terenkripsi saat Disimpan)
Data sensitif diacak sebelum disimpan ke database, dan hanya bisa dibaca dengan kunci. Beda dari
hashing, enkripsi **bisa** dikembalikan (dengan kunci yang benar).

🔑 **Analogi**: Brankas. Dokumen penting disimpan terkunci; hanya yang punya kunci bisa membukanya
kembali. (Bandingkan hashing = mesin penghancur kertas — tidak bisa dibalik.)

📍 Nomor rekening mitra: `bank_account_encrypted` disimpan terenkripsi, API hanya menampilkan
`bank_account_masked` (`****1234`).

### Masking
Menyembunyikan sebagian data sensitif saat ditampilkan, hanya memperlihatkan sedikit bagian.

🔑 **Analogi**: Struk ATM yang menampilkan `****1234` alih-alih nomor kartu lengkap.

---

## 6. HTTP & API

### API (Application Programming Interface)
"Pintu" yang disediakan backend supaya program lain (frontend, aplikasi mobile) bisa meminta data
atau mengirim perintah, dengan aturan yang jelas.

🔑 **Analogi**: Loket pemesanan. Kamu tidak masuk ke dapur; kamu memesan lewat loket dengan format
tertentu, dan makanan keluar dari loket itu.

### Endpoint / Route
Satu alamat spesifik di API yang menangani satu hal. Contoh: `GET /api/v1/kpi/ranking` adalah
endpoint untuk mengambil ranking.

🔑 **Analogi**: Nomor ekstensi telepon kantor. Ekstensi 101 = Keuangan, 102 = HRD. Tiap nomor
menghubungkan ke bagian tertentu.

### HTTP Method (GET, POST, PUT, PATCH, DELETE)
Jenis "kata kerja" dari request:
- **GET** — mengambil/membaca data (tidak mengubah apa pun).
- **POST** — membuat data baru.
- **PUT** — mengganti data secara utuh.
- **PATCH** — mengubah sebagian data.
- **DELETE** — menghapus data.

🔑 **Analogi**: Kata kerja pada perpustakaan. GET = "lihat buku", POST = "daftarkan buku baru", PUT =
"ganti buku ini dengan edisi baru", PATCH = "perbaiki judul yang salah ketik", DELETE = "keluarkan
buku dari koleksi".

### Status Code (200, 201, 400, 401, 403, 404, 409, 500)
Kode angka 3 digit yang menandakan hasil request:
- **2xx sukses**: `200 OK`, `201 Created`, `202 Accepted` (diterima, diproses belakangan), `204 No
  Content`.
- **4xx salah dari sisi pengirim**: `400` (input tidak valid), `401` (belum login), `403` (tidak
  punya izin), `404` (tidak ditemukan), `409` (bentrok, misal data duplikat).
- **5xx salah dari sisi server**: `500` (error tak terduga).

🔑 **Analogi**: Balasan standar pelayan. 200 = "pesanan siap"; 400 = "pesanan Anda tidak jelas"; 401
= "Anda belum pesan/registrasi"; 403 = "menu ini bukan untuk Anda"; 404 = "menu tidak ada"; 409 =
"meja itu sudah dipesan orang"; 500 = "dapur kami kebakaran, maaf".

### Request ID
Kode unik yang ditempelkan ke setiap request. Muncul di response **dan** di log server, sehingga
kalau ada error, kita bisa mencocokkan keluhan user dengan catatan log yang persis.

🔑 **Analogi**: Nomor resi/tiket komplain. Saat menelepon customer service, kamu sebut nomor resi,
dan mereka langsung menemukan riwayat kasusmu.

📍 Ada di `meta.request_id` dan `error.request_id` pada tiap response.

### CORS (Cross-Origin Resource Sharing)
Aturan keamanan browser: secara default, website di domain A tidak boleh memanggil API di domain B.
Server harus **secara eksplisit mengizinkan** domain mana yang boleh mengaksesnya.

🔑 **Analogi**: Daftar tamu yang boleh masuk acara. Backend punya daftar: "hanya
`http://localhost:3000` (frontend kita) yang boleh memanggil". Website asing ditolak di pintu.

📍 CORS di-set eksplisit di config.

### Pagination (Halaman Data)
Membagi data yang banyak menjadi potongan per halaman (`page` + `limit`), supaya tidak mengirim
ribuan baris sekaligus. Response menyertakan `total` supaya frontend tahu ada berapa halaman.

🔑 **Analogi**: Hasil pencarian Google. Tidak menampilkan sejuta hasil sekaligus — hanya 10 per
halaman, dengan tombol "Next".

📍 `?page=1&limit=10`, response `pagination: { page, limit, total }`.

### Bulk Operation (Operasi Massal)
Endpoint yang memproses banyak data dalam satu request, alih-alih satu per satu.

🔑 **Analogi**: Fotokopi 100 lembar sekaligus dengan sekali klik, bukan menekan tombol 100 kali.

📍 `POST /owners/bulk`, `POST /leads/bulk/assign-sales`, `POST /sales-targets/bulk`.

### Query Parameter vs Path Parameter vs Body
- **Path parameter**: bagian dari URL yang menunjuk resource spesifik — `/owners/{owner_id}`.
- **Query parameter**: filter/opsi setelah `?` — `/owners?city=Bandung&page=2`.
- **Body**: data isi request (biasanya JSON) untuk POST/PUT/PATCH.

🔑 **Analogi**: Path = alamat rumah (Jl. Merdeka **No. 20**). Query = catatan tambahan ("yang pagar
merah, lantai 2"). Body = isi paket yang kamu antar ke alamat itu.

---

## 7. Uang & Finansial

### Decimal vs Float (kenapa uang TIDAK pakai float)
`float` (bilangan desimal biasa di komputer) menyimpan angka secara **tidak presisi** — `0.1 + 0.2`
bisa jadi `0.30000000004`. Untuk uang itu bencana. Karena itu uang disimpan sebagai **DECIMAL** di
database dan diproses sebagai **string** atau **integer cents** di kode.

🔑 **Analogi**: Mengukur dengan penggaris karet vs penggaris besi. Float itu penggaris karet — kadang
sedikit melar. Untuk memotong emas (uang), kamu wajib pakai penggaris besi (decimal) yang presisinya
pasti.

📍 `final_amount DECIMAL(18,2)`; helper `parseMoneyToCents` mengubah `"150000.00"` jadi integer cents
supaya perhitungan bebas galat pembulatan.

### Cents (Sen)
Menyimpan/menghitung uang dalam satuan terkecil (sen) sebagai bilangan bulat, bukan pecahan. Rp
1.500,50 disimpan sebagai `150050`. Menghindari masalah presisi float.

🔑 **Analogi**: Menghitung jarak dalam milimeter (bilangan bulat) daripada meter dengan koma. Lebih
susah salah.

### Ledger (Buku Besar)
Catatan **semua** transaksi keuangan secara berurutan dan tidak diubah — hanya ditambah. Saldo tidak
disimpan sebagai satu angka yang diedit, tapi dihitung dari total seluruh catatan.

🔑 **Analogi**: Buku tabungan/rekening koran. Setiap setoran dan penarikan dicatat satu baris. Saldo
= hasil menjumlahkan semua baris. Kamu tidak menghapus/mengubah baris lama; kalau ada koreksi, kamu
tulis baris baru.

📍 `wallet_transactions` (Sprint 9): credit, debit, adjustment, refund.

### Reconciliation (Rekonsiliasi)
Proses mencocokkan dua catatan yang seharusnya sama, untuk memastikan tidak ada selisih. Di sini:
mencocokkan **pembelian paket** (order) dengan **closing** yang dibuat Sales.

🔑 **Analogi**: Mencocokkan struk belanja dengan mutasi rekening di akhir bulan. Kalau ada transaksi
di rekening yang tidak ada struknya (atau sebaliknya), itu ditandai untuk diperiksa.

📍 Sprint 10. Order tanpa closing yang cocok masuk `reconciliation_issues` sebagai `HANGING_ORDER`.

### Snapshot
Menyalin dan **membekukan** data pada satu momen, supaya perubahan di masa depan tidak mengubah
catatan lama. Saat closing dibuat, harga/paket/promo disalin ke closing itu — kalau nanti harga
paket naik, closing lama tetap memakai harga saat itu.

🔑 **Analogi**: Memotret. Foto pernikahan tahun 2020 tetap menunjukkan wajah 2020, meski orangnya
kini sudah berubah. Snapshot "memotret" harga saat transaksi terjadi.

📍 `package_snapshot_json`, `plan_snapshot_json`, `promotion_snapshot_json` di `sales_closings`.

### Effective Dating (Berlaku Mulai Tanggal)
Aturan/harga yang punya masa berlaku (`effective_from` s/d `effective_to`). Alih-alih mengedit aturan
lama, kita membuat aturan baru dengan tanggal berlaku berbeda. Yang dipakai adalah aturan yang aktif
pada tanggal transaksi.

🔑 **Analogi**: Daftar harga bermusim. "Harga promo berlaku 1–31 Juli." Struk tanggal 15 Juli otomatis
memakai harga promo; struk 5 Agustus memakai harga normal. Kita tidak menimpa daftar harga lama —
kita simpan keduanya dengan periode masing-masing.

📍 `commission_rules.effective_from/effective_to` (Sprint 12), `subscription_plans`, `promotions`.

### Double Counting (Penghitungan Ganda)
Kesalahan menghitung satu pemasukan dua kali. Dijaga ketat di modul keuangan.

🔑 **Analogi**: Menghitung uang THR dua kali karena tercatat di dua buku catatan. Terlihat "kaya" di
laporan, padahal uangnya cuma satu.

📍 Aturan: revenue diambil dari top-up (`paid_at`), bukan dari order subscription, supaya tidak
dihitung dua kali.

---

## 8. Background Processing (Worker & Job)

### Worker
Proses terpisah dari API yang berjalan di latar belakang, mengerjakan tugas yang berat atau tidak
perlu langsung selesai saat user menunggu.

🔑 **Analogi**: Bagian dapur/gudang di belakang restoran. Pelayan (API) mencatat pesanan lalu langsung
melayani tamu lain; koki di dapur (worker) mengerjakan masakan yang lama tanpa membuat tamu berdiri
menunggu di depan kasir.

📍 `go run . worker`, `RunWorker` di `internal/app/api.go`.

### Job Queue (Antrean Pekerjaan)
Daftar tugas (job) yang menunggu dikerjakan worker, disimpan di tabel database `job_queue`. Di
project ini **tanpa Redis** — murni MySQL (keputusan teknis yang dikunci sejak awal).

🔑 **Analogi**: Tumpukan tiket pesanan yang digantung di dapur. Koki mengambil tiket paling atas,
mengerjakannya, lalu ambil berikutnya.

📍 `internal/platform/jobqueue`, tabel `job_queue` (Sprint 13).

### Dispatch
Proses mengambil satu job dari antrean dan menjalankan fungsi yang sesuai dengan jenis job itu.

🔑 **Analogi**: Mandor membagikan tiket pesanan ke koki yang tepat: tiket "steak" ke koki grill,
tiket "salad" ke koki dingin.

📍 `jobqueue.Dispatch(...)` dipanggil tiap "detak" worker.

### Retry & Backoff
- **Retry**: kalau job gagal, coba lagi (sampai batas `max_attempts`, default 5).
- **Backoff**: jeda antar percobaan yang makin lama, supaya tidak membanjiri sistem dengan
  percobaan beruntun.

🔑 **Analogi**: Menelepon nomor yang sibuk. Kamu tidak menelepon terus-menerus tanpa jeda — kamu
tunggu makin lama tiap gagal (30 detik, lalu 1 menit, dst). Setelah 5 kali gagal, kamu menyerah dan
mencatat "tidak bisa dihubungi".

📍 `Fail(...)` di `jobqueue/repository.go`; job yang habis percobaan jadi `FAILED` dengan
`last_error`.

### Stale Job Reclaim (Merebut Job Mandek)
Kalau worker mati di tengah mengerjakan job (job "nyangkut" di status `PROCESSING` terlalu lama), job
itu dikembalikan ke antrean supaya bisa dikerjakan worker lain.

🔑 **Analogi**: Koki pingsan saat memegang tiket pesanan. Setelah sekian menit tiketnya tidak
selesai, mandor mengambil kembali tiket itu dan memberikannya ke koki lain — pesanan tamu tidak
terlantar.

📍 `ReclaimStale(...)` dengan batas `WORKER_STALE_JOB_TIMEOUT` (default 15 menit).

### Poll Interval (Interval Pengecekan)
Seberapa sering worker mengecek antrean untuk job baru. Default 3 detik.

🔑 **Analogi**: Koki melirik tumpukan tiket tiap beberapa detik untuk lihat ada pesanan baru atau
tidak.

📍 `WORKER_POLL_INTERVAL`.

### Heartbeat (Detak Jantung)
Sinyal berkala yang menunjukkan worker masih hidup dan sehat (di sini: ping ke database tiap
interval). Dulu worker **hanya** melakukan ini (stub), sebelum Sprint 13 menambahkan pemrosesan job
sungguhan.

🔑 **Analogi**: Alat pantau detak jantung di rumah sakit. Selama berbunyi "bip" teratur, kita tahu
pasien (worker) masih hidup.

### Async (Asinkron)
Cara kerja "kirim perintah sekarang, hasilnya belakangan". User meminta recompute KPI, langsung dapat
"OK, sedang diproses" (nomor job), lalu mengecek hasilnya nanti — tidak menunggu di tempat.

🔑 **Analogi**: Cuci mobil. Kamu menyerahkan mobil, dapat nomor antrean, lalu pergi ngopi. Kamu tidak
berdiri menonton sampai selesai; kamu balik lagi nanti mengecek.

📍 `POST /kpi/recompute` mengembalikan `202 Accepted` + `job_id`; cek `GET /kpi/jobs/{id}`.

### Graceful Shutdown (Mati dengan Rapi)
Saat aplikasi diminta berhenti, ia **menyelesaikan request yang sedang berjalan** dulu sebelum
benar-benar mati, bukan langsung memutus di tengah jalan.

🔑 **Analogi**: Toko mau tutup. Kasir tidak mengusir pelanggan yang sedang membayar — dia selesaikan
transaksi yang ada dulu, baru mengunci pintu. Bukan tiba-tiba mematikan lampu saat orang masih di
kasir.

---

## 9. Data Dummy & Testing

### Factory
Kode yang **membuat data dummy** (palsu tapi realistis) untuk keperluan testing dan demo. Contoh:
membuat user, owner, closing bohongan.

🔑 **Analogi**: Pabrik manekin. Menghasilkan "orang-orangan" lengkap dengan baju untuk memajang
display toko — bukan manusia asli, tapi cukup mirip untuk keperluan pajangan.

📍 `internal/platform/factory` — `BuildUser`, `CreateOwner`, `BuildClosing`, dll.

### Seeder
Kode yang **mengisi database dengan data awal**. Ada dua jenis:
- **Master seed**: data referensi wajib (role, permission, metric codes, paket awal).
- **Demo seed**: data contoh untuk demo/testing, dengan **preset** `minimal` atau `large`.

🔑 **Analogi**: Menanam benih di kebun kosong. Master seed = menanam pohon-pohon pokok yang harus ada.
Demo seed = menaruh contoh tanaman hias supaya kebun terlihat "hidup" saat dipamerkan.

📍 `go run . seed master`, `go run . seed demo --preset=minimal`.

### Deterministic (Deterministik)
Sifat "selalu menghasilkan output yang sama untuk input yang sama". Seeder deterministik: dengan
`--seed=20260725` yang sama, data yang dibuat selalu identik — bisa direproduksi.

🔑 **Analogi**: Resep kue yang tepat takarannya. Dengan bahan dan takaran yang persis sama, kuenya
selalu jadi sama, siapa pun yang membuat, kapan pun.

📍 Parameter `--seed=<angka>` mengunci generator acak supaya hasilnya konsisten.

### Preset (minimal / standard / large)
Ukuran/skala data demo. `minimal` = sedikit data untuk cek cepat; `large` = data skala besar (Sprint
11b/11c) untuk uji beban (load test).

🔑 **Analogi**: Ukuran porsi di restoran: porsi kecil untuk cicip, porsi jumbo untuk uji "seberapa
kenyang".

### Unit Test
Pengujian **potongan kecil** kode secara terisolasi (satu fungsi), tanpa database atau jaringan.
Cepat dan banyak.

🔑 **Analogi**: Mengetes satu baut apakah kuat, sebelum dipasang ke mesin. Tes komponen tunggal.

📍 `*_test.go`, misal `TestParsePercent`, `TestClassify`, `TestValidateDecimal`.

### Integration Test
Pengujian **beberapa bagian bekerja sama**, biasanya melibatkan database sungguhan.

🔑 **Analogi**: Mengetes apakah mesin menyala setelah semua baut dan komponen terpasang — bukan cuma
per komponen.

### Smoke Test
Uji cepat untuk memastikan "fitur utama tidak langsung meledak" — menjalankan alur pokok dari ujung
ke ujung (end-to-end) lewat API sungguhan.

🔑 **Analogi**: Menyalakan mesin mobil baru dan mengecek apakah ada asap (smoke). Bukan uji lengkap,
tapi cukup untuk tahu "tidak terbakar begitu dinyalakan".

📍 Tiap sprint diakhiri smoke test via `curl` di database terisolasi, didokumentasikan di
`docs/sprint-XX/README.md`.

### `go build` / `go vet` / `go test`
- **`go build`**: memastikan kode bisa dikompilasi jadi program (tidak ada error sintaks).
- **`go vet`**: pemeriksa statis yang menangkap bug halus yang lolos dari compiler.
- **`go test`**: menjalankan semua test.

🔑 **Analogi**: Sebelum mobil keluar pabrik — build = "mobilnya jadi utuh"; vet = "inspektur QC cek
cacat tersembunyi"; test = "test drive di berbagai kondisi".

📍 "Quality gate" wajib bersih sebelum sprint dinyatakan selesai.

### End-to-End (E2E)
Pengujian seluruh alur dari titik awal user sampai hasil akhir, seolah-olah user sungguhan
memakainya.

🔑 **Analogi**: Menguji jalur pengiriman paket dari pemesanan online sampai paket tiba di depan pintu
— bukan cuma mengetes satu tahap.

---

## 10. Istilah Domain Bisnis Piposmart

Istilah khusus bisnis CRM ini (bukan istilah pemrograman umum).

### Owner & Outlet
- **Owner**: pemilik usaha laundry (pelanggan bisnis Piposmart).
- **Outlet**: cabang/gerai fisik milik seorang owner. Satu owner bisa punya banyak outlet.

🔑 **Analogi**: Owner = pemilik waralaba; Outlet = tiap gerai cabangnya.

📍 Sprint 4. `owners`, `outlets`.

### Lead
Calon pelanggan / prospek yang sedang di-follow-up tim Sales. Satu lead terhubung ke satu owner.

🔑 **Analogi**: Nama di daftar "orang yang mungkin tertarik beli". Belum tentu jadi pelanggan, tapi
layak dihubungi.

📍 Sprint 5. `customer_leads`.

### Assignment & PIC
- **Assignment**: penugasan sebuah lead/mitra ke seorang penanggung jawab.
- **PIC (Person In Charge)**: orang yang bertanggung jawab atas lead/mitra tersebut.

Aturan penting: satu lead/mitra hanya boleh punya **satu PIC aktif** dalam satu waktu.

🔑 **Analogi**: Dokter penanggung jawab pasien. Satu pasien punya satu dokter utama pada satu waktu —
kalau ganti dokter, dokter lama otomatis lepas tanggung jawab.

📍 Sprint 5 (lead), Sprint 11 (partner). `lead_assignments`, `partner_assignments`.

### Stage & Remark Score (0–3)
- **Stage**: tahap perjalanan lead: `NEW` → `POSSIBLE` → `POTENTIAL` → `CLOSING` (atau `INVALID`).
- **Remark score**: skor 0–3 yang diberi Sales setelah menghubungi customer, yang menentukan
  perpindahan stage:
  - `0` → lead `INVALID` (tidak potensial, dikembalikan ke Supervisor).
  - `1` → `POSSIBLE`.
  - `2` → `POTENTIAL`.
  - `3` → `CLOSING` (siap dibuatkan laporan penjualan).

🔑 **Analogi**: Tahapan pendekatan (PDKT). Baru kenalan (NEW), mungkin cocok (POSSIBLE), serius
(POTENTIAL), jadian/closing. Skor tiap "kencan" menentukan naik atau putus.

📍 Sprint 6. `customer_interactions.remark_score`, `lead_stage_histories`.

### Interaction (Call / Chat)
Catatan aktivitas menghubungi customer (telepon atau chat), bersifat **append-only** (hanya
ditambah, tidak diubah/dihapus).

🔑 **Analogi**: Buku catatan panggilan yang tintanya permanen. Kamu boleh menambah catatan baru, tapi
tidak boleh menghapus/menimpa catatan lama — untuk jejak audit.

📍 `customer_interactions`.

### Closing
Laporan penjualan yang dibuat saat lead berhasil "closing" (deal). Memuat snapshot paket, harga,
promo, dan status `PENDING_RECONCILIATION` → `CONFIRMED` / `REJECTED`.

🔑 **Analogi**: Nota penjualan resmi. Dibuat saat transaksi terjadi, membekukan detail harga saat itu.

📍 Sprint 8. `sales_closings`.

### Catalog: Package, Plan, Promotion, Benefit
- **Package**: paket produk (BASIC, BUSINESS, PRO).
- **Plan**: varian paket berdasarkan tenor/durasi + harga.
- **Promotion**: promo/diskon dengan masa berlaku.
- **Benefit**: keuntungan dari promo (gratis bulan, diskon, perangkat, dll).

🔑 **Analogi**: Menu paket seluler. Package = "Paket Internet"; Plan = "10GB/bulan seharga X"; Promo =
"diskon Lebaran"; Benefit = "bonus kuota malam".

📍 Sprint 7. `subscription_packages`, `subscription_plans`, `promotions`.

### Wallet & Top-up
- **Wallet**: dompet saldo milik owner.
- **Top-up**: pengisian saldo. Saldo dipakai untuk membeli subscription.

🔑 **Analogi**: Saldo e-money (GoPay/OVO). Isi saldo dulu (top-up), lalu belanja dari saldo.

📍 Sprint 9. `wallet_accounts`, `wallet_transactions`.

### Subscription, Order, Period
- **Subscription Order**: pembelian paket yang mendebit saldo wallet.
- **Subscription**: langganan aktif hasil dari order.
- **Period**: rentang masa aktif langganan (durasi tetap `tenure_months × 30 hari`).

🔑 **Analogi**: Berlangganan Netflix. Order = klik "beli paket"; Subscription = status "aktif";
Period = "berlaku 1 Juli–31 Juli".

📍 Sprint 10. `subscription_orders`, `subscriptions`, `subscription_periods`.

### Partner (Mitra), Partner Type, Referral
- **Partner**: mitra bisnis (supplier, distributor, agent, komunitas referral).
- **Partner Type**: kategori mitra, yang membawa rate komisi default.
- **Referral**: customer lead yang direkomendasikan oleh mitra.

🔑 **Analogi**: Program "member get member". Mitra mereferensikan calon pelanggan; kalau jadi, mitra
dapat komisi.

📍 Sprint 11. `partners`, `partner_types`, `partner_referrals`.

### Commission (Komisi) & Mode: PERCENTAGE / FIXED / TIER
Imbalan untuk mitra saat referral-nya menghasilkan closing `CONFIRMED`. Rate-nya bisa:
- **PERCENTAGE**: persen dari nilai closing.
- **FIXED**: nominal tetap.
- **TIER**: bertingkat berdasarkan **volume closing per bulan** — makin banyak closing, rate bisa
  naik.

🔑 **Analogi TIER**: Bonus penjualan berjenjang. Jual 1–3 unit dapat 2% per unit; jual unit ke-4 dan
seterusnya dapat 5%. Makin rajin, makin besar persentasenya.

📍 Sprint 12 + addendum. `partner_commissions`, `commission_rules`, `commission_tiers`.

### Commission Rule (Overlay yang Additive)
Aturan komisi opsional yang effective-dated dan bisa spesifik per paket. Kalau ada rule yang cocok,
dipakai; kalau tidak, **jatuh kembali (fallback)** ke rate flat default partner type. Rule ini
"menimpa" default hanya jika cocok — tidak menghapusnya.

🔑 **Analogi**: Harga promo yang menempel di atas harga normal. Kalau lagi ada promo yang berlaku,
pakai harga promo; kalau tidak, pakai harga normal yang selalu ada di belakang.

📍 `resolveCommissionRule` di `partner/repository.go`.

### Payout & Payout Item (Batching)
- **Payout**: satu pencairan yang **membatch** (menggabung) beberapa komisi `APPROVED` milik satu
  mitra.
- **Payout Item**: tiap komisi yang masuk ke dalam payout.

Komisi yang sudah masuk payout aktif tidak bisa dibayar individual (**double-pay guard**). Membatalkan
payout melepas komisi kembali ke `APPROVED` (**soft-release**).

🔑 **Analogi**: Amplop gaji yang berisi beberapa slip bonus sekaligus. Alih-alih mencairkan tiap bonus
satu-satu, semuanya dibayar dalam satu amplop. Kalau amplop dibatalkan, tiap slip bonus kembali ke
status "siap dibayar".

📍 Sprint 12 addendum. `partner_payouts`, `partner_payout_items`.

### Metric Code
Kode standar untuk jenis metrik yang bisa diukur: `CONFIRMED_CLOSING_COUNT`, `CALL_CUSTOMER_COUNT`,
`TRAINING_COUNT`, dll. Sudah ada sejak Sprint 2, jadi fondasi untuk Target & KPI.

🔑 **Analogi**: Satuan pengukuran standar (kg, meter, liter). Semua orang setuju "kg" artinya apa,
jadi bisa dipakai konsisten di mana-mana.

📍 `metric_codes` (Sprint 2), dipakai Sprint 13.

### Sales Target: Bulk vs Override
- **Bulk-set target**: menetapkan target default untuk **semua** Sales aktif yang belum punya target
  di periode itu — **tidak pernah menimpa** yang sudah ada.
- **Override**: menetapkan target khusus untuk **satu** Sales — **selalu menang**.

🔑 **Analogi**: Kebijakan cuti kantor. Bulk = "semua karyawan dapat jatah 12 hari" (default). Override
= "si A khusus dapat 15 hari" (pengecualian yang menimpa default, hanya untuk dia).

📍 Sprint 13. `POST /sales-targets/bulk` vs `PUT /sales-targets/{salesID}`.

### KPI (Key Performance Indicator) — Definition, Weight, Threshold
- **KPI Definition**: konfigurasi bobot & ambang untuk satu metric di satu periode.
- **Weight (bobot)**: seberapa penting metric itu; total semua bobot aktif per periode wajib **100%**.
- **Threshold**: ambang klasifikasi — `threshold_achieved` (tercapai) dan `threshold_near` (hampir
  tercapai).

🔑 **Analogi**: Rapor sekolah dengan bobot mata pelajaran. Matematika bobot 40%, Bahasa 30%, dst
(total 100%). Nilai akhir = rata-rata berbobot. Di atas 90 = "sangat baik" (threshold), 75–89 =
"baik".

📍 Sprint 13. `kpi_definitions`.

### KPI Recompute & Classification
- **Recompute**: menghitung ulang skor KPI semua Sales untuk satu periode, dijalankan async lewat
  worker, dan **idempoten**.
- **Classification**: hasil akhir per Sales: `ACHIEVED` / `NEAR_ACHIEVED` / `NOT_ACHIEVED`.

🔑 **Analogi**: Menghitung rapor akhir semester. Bisa dihitung ulang kapan saja dari nilai-nilai yang
ada; hasilnya sama kalau nilainya sama. Hasil akhir: lulus / hampir / tidak lulus.

📍 Sprint 13. `POST /kpi/recompute`, `sales_kpi_results`.

### Ranking
Urutan peringkat Sales berdasarkan total skor KPI dalam satu periode. Hanya Admin/Supervisor yang
melihat daftar penuh; Sales hanya melihat posisinya sendiri.

🔑 **Analogi**: Papan peringkat kelas. Wali kelas (Admin/Supervisor) lihat ranking seluruh murid;
tiap murid (Sales) hanya diberi tahu peringkatnya sendiri, bukan daftar lengkap.

📍 Sprint 13. `GET /kpi/ranking`, kolom `rank_position` (dihitung dengan fungsi `RANK()`).

### Visibility Scoping (Pembatasan Keterlihatan)
Aturan siapa boleh melihat data siapa, berdasarkan role: Admin lihat semua, Supervisor lihat timnya,
Sales lihat miliknya sendiri. Diterapkan konsisten di banyak modul dengan pola yang sama
(`visibilityWhere`).

🔑 **Analogi**: Akses folder di komputer kantor. Direktur lihat semua folder; manajer lihat folder
timnya; staf lihat foldernya sendiri. Folder yang sama, tapi yang terlihat berbeda tergantung siapa
yang login.

📍 `visibilityWhere(actor)` di `lead`, `closing`, `target`, `kpi`.

---

## Catatan Penutup

Glosarium ini akan diperbarui seiring bertambahnya modul. Kalau menemukan istilah baru yang belum
ada di sini saat membaca kode atau dokumentasi sprint, tambahkan entri baru mengikuti format yang
sama: **penjelasan teknis + 🔑 analogi + 📍 lokasi di project**.

Untuk konteks lebih dalam per fitur, lihat:
- `BACKEND_PLAN_SPRINT.md` — roadmap 18 sprint.
- `docs/sprint-XX/` — laporan & testing tiap sprint.
- `internal/platform/httpserver/openapi.yaml` — kontrak API lengkap (buka di `/swagger`).
