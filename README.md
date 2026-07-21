# Backend CRM Piposmart

Backend API untuk aplikasi frontend CRM Piposmart, dibangun menggunakan [Go](https://go.dev/) dengan framework [Gin](https://gin-gonic.com/), ORM [GORM](https://gorm.io/) (MySQL driver), migration database menggunakan [Goose](https://github.com/pressly/goose), dan dokumentasi API otomatis menggunakan [Swagger](https://github.com/swaggo/swag).

## Tech Stack

- **Language:** Go 1.26.4
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin)
- **ORM:** [GORM](https://gorm.io/) + MySQL Driver
- **Database Migration:** [Goose](https://github.com/pressly/goose)
- **API Documentation:** [Swaggo (gin-swagger)](https://github.com/swaggo/gin-swagger)
- **Fake Data Generator:** [gofakeit](https://github.com/brianvoe/gofakeit)

## Struktur Folder

```
.
├── cmd/            # Command tambahan (mis. db:seed)
├── controllers/     # HTTP handler, tempat menerima request & mengembalikan response
├── database/        # Koneksi database (GORM + MySQL)
├── docs/             # File hasil generate Swagger (docs.go, swagger.json, swagger.yaml)
├── migrations/       # File migration Goose (SQL) + panduan penggunaannya
├── models/           # Struct model/entity GORM
├── repositories/      # Layer akses data ke database
├── requests/          # Struct untuk validasi & binding request body
├── responses/         # Struct untuk format response API
├── routes/            # Pendaftaran route/endpoint API
├── seeders/           # Seeder untuk mengisi data dummy
├── go.mod / go.sum
└── main.go            # Entry point aplikasi
```

Alur request mengikuti pola berlapis:
**Route → Controller → Service → Repository → Database (Model)**, dengan `requests` sebagai validasi input dan `responses` sebagai format output.

## Prasyarat

Sebelum menjalankan project, pastikan sudah terinstall:

- [Go](https://go.dev/dl/) versi 1.26.4 atau lebih baru
- [MySQL Server](https://dev.mysql.com/downloads/) (lokal atau remote)
- [Goose CLI](https://github.com/pressly/goose#install) untuk menjalankan migration

  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

## Instalasi & Menjalankan Project

### 1. Clone repository & install dependencies

```bash
git clone <url-repository-ini>
cd backend_crm_piposmart
go mod tidy
```

### 2. Konfigurasi Database

Buat database MySQL bernama `crm_piposmart`, lalu sesuaikan koneksi database pada `database/mysql.go` jika diperlukan (default menggunakan user `root` tanpa password, host `localhost:3306`):

```go
dsn := "root:root@tcp(localhost:3306)/crm_piposmart?charset=utf8mb4&parseTime=True&loc=Local"
```

### 3. Menjalankan Migration

Migration menggunakan Goose. Jalankan dari dalam folder `migrations`:

```bash
cd migrations
goose mysql "root:root@tcp(localhost:3306)/crm_piposmart?parseTime=true" up
```

> Panduan lengkap seputar migration (up, down, reset, membuat migration baru, contoh query SQL, dsb) tersedia di [`migrations/README.md`](./migrations/README.md).

### 4. Menjalankan Server API

```bash
go run main.go api
```

Server akan berjalan di `http://localhost:8080`.

### 5. Menjalankan Seeder (opsional)

Untuk mengisi data dummy ke database:

```bash
# Seed semua data (sales + customer)
go run main.go seed

# Seed sales saja
go run main.go seed sales

# Seed customer saja (100 data)
go run main.go seed customer
```

## Dokumentasi API (Swagger)

Setelah server dijalankan, dokumentasi API dapat diakses melalui browser di:

```
http://localhost:8080/swagger/index.html
```

Jika terdapat perubahan pada anotasi Swagger di controller, generate ulang dokumentasi dengan [swag CLI](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

## Daftar Endpoint

### Customer

| Method | Endpoint             | Deskripsi                              |
|--------|-----------------------|------------------------------------------|
| GET    | `/customer`            | Menampilkan data customer                |
| POST   | `/customer`            | Membuat data customer baru                |
| PATCH  | `/customer`            | Mengubah data customer                    |
| DELETE | `/customer`            | Menghapus data customer (soft delete)     |
| POST   | `/customer/restore`    | Mengembalikan data customer yang terhapus |
| DELETE | `/customer/force`      | Menghapus data customer secara permanen   |
| GET    | `/customer/all`        | Menampilkan seluruh data customer         |
| GET    | `/customer/deleted`    | Menampilkan data customer yang terhapus   |

### Sales

| Method | Endpoint            | Deskripsi                              |
|--------|----------------------|------------------------------------------|
| GET    | `/sales`              | Menampilkan data sales                    |
| POST   | `/sales`              | Membuat data sales baru                    |
| PATCH  | `/sales`              | Mengubah data sales                        |
| DELETE | `/sales`              | Menghapus data sales (soft delete)         |
| POST   | `/sales/restore`      | Mengembalikan data sales yang terhapus     |
| DELETE | `/sales/force`        | Menghapus data sales secara permanen       |
| GET    | `/sales/deleted`      | Menampilkan data sales yang terhapus       |
| GET    | `/sales/all`          | Menampilkan seluruh data sales             |

Detail lengkap request/response body untuk setiap endpoint dapat dilihat di halaman Swagger.

## Author

Achul
