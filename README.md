# Backend CRM Piposmart

Backend internal CRM PT Piposmart Digital Indonesia. Fondasi baru memakai Go,
Gin, GORM, MySQL 8, Goose, dan OpenAPI.

Sprint 1 menyediakan:

- entrypoint `cmd/crm`;
- konfigurasi tervalidasi dari environment dan `.env`;
- API dan worker dengan graceful shutdown;
- structured logging, request ID, panic recovery, serta CORS eksplisit;
- liveness dan readiness check;
- migration command berbasis Goose;
- Dockerfile multi-stage dan Docker Compose;
- pipeline test, vet, build, dan container build.

> Kode CRUD prototipe masih berada di repository untuk referensi sementara,
> tetapi tidak digunakan entrypoint baru. Model dan migration domain baru akan
> menggantikannya pada Sprint 2.

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
go run ./cmd/crm migrate up
go run ./cmd/crm api
```

Worker dijalankan pada terminal lain:

```powershell
go run ./cmd/crm worker
```

Command yang tersedia:

```text
crm api
crm worker
crm migrate up
crm migrate down
crm migrate status
crm migrate version
crm version
crm help
```

Seeder dan bootstrap Admin akan ditambahkan pada Sprint 2 dan Sprint 3.

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

```powershell
go test ./...
go vet ./...
go build ./cmd/crm
```

Build container:

```powershell
docker build -t crm-piposmart-backend:local .
```

## Struktur Fondasi

```text
cmd/crm/                       Entry point API, worker, dan migration
internal/app/                  Lifecycle executable
internal/platform/config/      Konfigurasi dan validasi environment
internal/platform/database/    Koneksi dan pool MySQL
internal/platform/httpserver/  Router, middleware, health, OpenAPI
internal/platform/httpx/       Response envelope API
internal/platform/logging/     Structured logging
internal/platform/migration/   Goose runner
migrations/                    SQL migration
```
