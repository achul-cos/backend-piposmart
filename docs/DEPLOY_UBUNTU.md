# Deploy Backend ke VPS Ubuntu

Dokumen ini untuk rilis backend `1.0 beta` via Docker di VPS Ubuntu.

## Prasyarat

- Ubuntu 22.04/24.04
- Docker Engine + Docker Compose Plugin
- Port `8080` terbuka jika backend diakses langsung, atau port internal jika di belakang reverse proxy

## 1. Install Docker

```bash
sudo apt update
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## 2. Clone project

```bash
git clone <repo-backend-anda> backend_crm_piposmart
cd backend_crm_piposmart
```

## 3. Siapkan env production

```bash
cp .env.production.example .env.production
```

Wajib ganti minimal:

- `DB_PASSWORD`
- `MYSQL_ROOT_PASSWORD`
- `JWT_ACCESS_SECRET`
- `BOOTSTRAP_ADMIN_PASSWORD`
- `DATA_ENCRYPTION_KEY`
- `CORS_ALLOWED_ORIGINS`
- `AUTH_COOKIE_DOMAIN`

Catatan:

- `SETUP_RUN_DEMO_SEED=false` sudah default untuk production.
- `SETUP_RUN_BOOTSTRAP_ADMIN=true` aman untuk first deploy, tapi setelah admin awal terbentuk biasanya bisa diubah ke `false`.
- `DB_HOST=mysql` dan `EXTERNAL_DB_NETWORK=mysql-stack_default` adalah default aman untuk dua mode deploy berbeda:
  - `compose.prod.yaml` untuk MySQL bawaan stack backend
  - `compose.prod.external-db.yaml` untuk MySQL container existing di VPS

## 4A. Mode standar: backend + MySQL dalam satu stack

Pakai file ini bila backend ingin membawa container MySQL sendiri:

```bash
docker compose -f compose.prod.yaml --env-file .env.production up -d --build
```

## 4B. Mode VPS existing DB: pakai MySQL container yang sudah ada

Pakai mode ini bila VPS sudah punya MySQL container terpisah, misalnya:

- container: `Core-Database`
- network Docker: `mysql-stack_default`
- alias database di network: `db`

Untuk mode ini, atur `.env.production` seperti berikut:

```env
DB_HOST=db
DB_PORT=3306
EXTERNAL_DB_NETWORK=mysql-stack_default
```

Kalau database target belum ada, buat dulu:

```bash
docker exec -it Core-Database mysql -uroot -p
```

Lalu di shell MySQL:

```sql
CREATE DATABASE IF NOT EXISTS crm_piposmart
CHARACTER SET utf8mb4
COLLATE utf8mb4_unicode_ci;
```

Setelah itu jalankan backend dengan compose khusus external DB:

```bash
docker compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
```

## 5. Verifikasi

```bash
docker compose -f compose.prod.yaml --env-file .env.production ps
docker compose -f compose.prod.yaml --env-file .env.production logs -f setup
docker compose -f compose.prod.yaml --env-file .env.production logs -f api
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

Untuk mode external DB, cukup ganti file compose pada setiap command verifikasi:

```bash
docker compose -f compose.prod.external-db.yaml --env-file .env.production ps
docker compose -f compose.prod.external-db.yaml --env-file .env.production logs -f setup
docker compose -f compose.prod.external-db.yaml --env-file .env.production logs -f api
```

## 6. Update release berikutnya

```bash
git pull
docker compose -f compose.prod.yaml --env-file .env.production up -d --build
```

Untuk mode external DB:

```bash
git pull
docker compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
```

## 7. Rollback cepat

Paling aman pakai image tag per-release. Kalau masih build langsung dari source, rollback berarti checkout commit/tag lama lalu jalankan ulang:

```bash
git checkout <commit-atau-tag-lama>
docker compose -f compose.prod.yaml --env-file .env.production up -d --build
```

Untuk mode external DB:

```bash
git checkout <commit-atau-tag-lama>
docker compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
```

## Catatan audit

- Container backend sekarang ikut membawa folder `asset/`, jadi command seed berbasis file Excel tetap tersedia di environment container.
- Compose production dipisahkan dari `compose.yaml` supaya environment VPS tidak ikut menjalankan demo seed seperti environment development.
