# Backend CRM Piposmart

Backend internal CRM PT Piposmart Digital Indonesia. Fondasi baru memakai Go,
Gin, GORM, MySQL 8, Goose, dan OpenAPI.

Fondasi saat ini menyediakan:

- entrypoint root `main.go`, sehingga command lokal cukup `go run . ...`;
- konfigurasi tervalidasi dari environment dan `.env`;
- API dan worker dengan graceful shutdown;
- structured logging, request ID, panic recovery, serta CORS eksplisit;
- liveness dan readiness check;
- migration command berbasis Goose;
- baseline schema, factory, dan seeder awal;
- Dockerfile multi-stage dan Docker Compose;
- pipeline test, vet, build, dan container build.

## Prasyarat

- Go sesuai versi pada `go.mod`;
- MySQL 8 untuk menjalankan tanpa Docker; atau
- Docker Engine dan Docker Compose v2 untuk environment container.

## Menjalankan dengan Docker

Salin konfigurasi contoh:

```powershell
Copy-Item .env.example .env
```

Ganti minimal `DB_PASSWORD` dan `MYSQL_ROOT_PASSWORD` untuk kebutuhan lokal.
Kemudian:

```powershell
docker compose up --build
```

Service yang dijalankan:

- `mysql`: database MySQL 8;
- `migrate`: menjalankan Goose sekali sebelum API;
- `api`: HTTP API di `http://localhost:8080`;
- `worker`: fondasi background worker.

Endpoint demo:

```text
GET http://localhost:8080/
GET http://localhost:8080/health/live
GET http://localhost:8080/health/ready
GET http://localhost:8080/api/v1/status
GET http://localhost:8080/swagger/index.html
```

Menghentikan environment:

```powershell
docker compose down
```

Data MySQL berada pada named volume. Gunakan `docker compose down --volumes`
hanya ketika memang ingin menghapus seluruh data development.

## Menjalankan secara Lokal

Salin `.env.example` menjadi `.env`, kemudian ubah:

```dotenv
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=crm_piposmart
DB_USER=crm_user
DB_PASSWORD=your-local-password
MIGRATION_DIR=./migrations
UPLOAD_DIR=./storage/uploads
EXPORT_DIR=./storage/exports
```

Jalankan migration dan API:

```powershell
go run . migrate up
go run . seed master
go run . bootstrap-admin
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
go run . api
```

Worker dijalankan pada terminal lain:

```powershell
go run . worker
```

Command yang tersedia:

```text
crm api
crm worker
crm migrate up
crm migrate down
crm migrate status
crm migrate version
crm seed master
crm seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
crm bootstrap-admin
crm version
crm help
```

`bootstrap-admin` membuat akun Admin awal berdasarkan variable
`BOOTSTRAP_ADMIN_*` di `.env`. Seeder demo juga membuat akun berikut:

```text
admin.001@demo.piposmart.id / Password123!
supervisor.001@demo.piposmart.id / Password123!
sales.001@demo.piposmart.id / Password123!
sales.002@demo.piposmart.id / Password123!
```

Contoh login dan memakai access token:

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/auth/login `
  -ContentType 'application/json' `
  -Body '{"email":"admin.001@demo.piposmart.id","password":"Password123!"}'

$token = $login.data.access_token
Invoke-RestMethod `
  -Method Get `
  -Uri http://localhost:8080/api/v1/auth/me `
  -Headers @{ Authorization = "Bearer $token" }
```

Contoh membuat Sales baru:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/sales `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType 'application/json' `
  -Body '{"name":"Sales Baru","email":"sales.baru@demo.piposmart.id","phone":"6281212345678"}'
```

## Konfigurasi

`.env.example` adalah kontrak konfigurasi. File `.env` tidak boleh dikomit.
Environment yang diberikan oleh OS atau container selalu menang terhadap nilai
dari `.env`.

Production harus:

- menggunakan secret manager untuk database dan key autentikasi;
- memakai origin CORS eksplisit;
- memakai database user non-root;
- menjalankan migration sebagai release job, bukan saat setiap replica API
  dimulai.

## Verifikasi

Jalankan perintah berikut dari root backend:

```powershell
cd C:\piposmart\backend_crm_piposmart
```

### `go test`

`go test` menjalankan seluruh unit test dan integration test ringan pada semua
package Go. Gunakan ini setiap selesai mengubah logic, migration helper,
factory, seeder, handler, service, atau repository.

```powershell
go test ./...
```

Jika muncul error permission pada Go build cache di Windows, gunakan cache lokal
di workspace:

```powershell
$env:GOCACHE='C:\piposmart\backend_crm_piposmart\.cache\go-build'
go test ./...
```

### `go vet`

`go vet` mengecek potensi bug statis yang sering tidak tertangkap compiler,
misalnya format string salah, struct tag mencurigakan, atau penggunaan API yang
rawan keliru. Jalankan sebelum commit atau sebelum laporan Sprint.

```powershell
go vet ./...
```

### `go build`

`go build` memastikan aplikasi bisa dikompilasi menjadi binary. Karena
entrypoint ada di root `main.go`, command build cukup:

```powershell
go build .
```

Di Windows, command ini menghasilkan file binary seperti
`backend_crm_piposmart.exe`. File `.exe` sudah di-ignore oleh Git, jadi tidak
perlu dikomit.

Untuk menjalankan binary hasil build:

```powershell
.\backend_crm_piposmart.exe help
.\backend_crm_piposmart.exe migrate status
```

### Quality gate harian

Sebelum lanjut Sprint berikutnya atau sebelum commit besar, jalankan paket
lengkap ini:

```powershell
$env:GOCACHE='C:\piposmart\backend_crm_piposmart\.cache\go-build'
go test ./...
go vet ./...
go build .
git diff --check
```

Build container:

```powershell
docker build -t crm-piposmart-backend:local .
```

## Struktur Fondasi

```text
main.go                        Entry point API, worker, migration, dan seeder
internal/app/                  Lifecycle executable
internal/platform/config/      Konfigurasi dan validasi environment
internal/platform/database/    Koneksi dan pool MySQL
internal/platform/factory/     Factory data dummy deterministik
internal/platform/httpserver/  Router, middleware, health, OpenAPI
internal/platform/httpx/       Response envelope API
internal/platform/logging/     Structured logging
internal/platform/migration/   Goose runner
internal/platform/seeder/      Master dan demo seeder
migrations/                    SQL migration
```
