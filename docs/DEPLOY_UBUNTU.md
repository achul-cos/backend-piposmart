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

Alur praktis yang dipakai sehari-hari:

1. Di lokal, push perubahan backend ke repo:

   ```bash
   git add .
   git commit -m "update backend"
   git push public main
   ```

2. Di VPS, ambil update source:

   ```bash
   cd /opt/backend-piposmart
   git pull
   ```

3. Rebuild dan jalankan ulang container:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
   ```

Catatan:

- Bila update hanya menyentuh backend `api`, bisa rebuild service itu saja:

  ```bash
  docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps api
  ```

- Bila update hanya menyentuh worker:

  ```bash
  docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps worker
  ```

- `setup` pada backend ini akan otomatis menjalankan `migrate up`, `seed master`, dan `bootstrap-admin` sesuai env saat service tersebut dijalankan ulang.

## 6A. Kalau ingin update `.env.production`

Perubahan file env tidak otomatis masuk ke container yang sedang berjalan. Setelah edit `.env.production`, service terkait harus di-recreate.

Langkah aman:

1. Edit env:

   ```bash
   nano .env.production
   ```

2. Simpan perubahan, lalu jalankan recreate:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d
   ```

3. Kalau perubahan env memengaruhi image atau source code juga, pakai build sekalian:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
   ```

Panduan cepat memilih service yang perlu dijalankan ulang:

- Jika mengubah env API seperti `APP_PORT`, `APP_ENV`, `JWT_*`, `CORS_*`, `AUTH_*`, `OPENAPI_*`, jalankan ulang `api`.
- Jika mengubah env worker seperti `WORKER_*`, queue, polling, atau koneksi database, jalankan ulang `worker`.
- Jika mengubah env setup seperti `SETUP_RUN_*`, `BOOTSTRAP_ADMIN_*`, atau ingin menjalankan migrate/seed lagi, jalankan `setup` secara manual.

Contoh recreate service tertentu saja:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --no-deps api
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --no-deps worker
```

Kalau yang berubah adalah env untuk proses setup, jalankan manual:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup
```

Contoh:

- Setelah ganti `CORS_ALLOWED_ORIGINS`, cukup recreate `api`.
- Setelah ganti `WORKER_POLL_INTERVAL`, cukup recreate `worker`.
- Setelah ganti `BOOTSTRAP_ADMIN_PASSWORD`, perubahan tidak akan mengubah password admin lama secara otomatis; env itu dipakai saat proses bootstrap dijalankan lagi.

## 6B. Troubleshooting `ContainerConfig` di docker-compose lama

Pada beberapa VPS yang masih memakai `docker-compose 1.29.2`, proses recreate container bisa gagal dengan error:

```text
KeyError: 'ContainerConfig'
```

Kalau ini terjadi, jangan menebak nama container. Cari dulu nama container yang benar:

Untuk `api`:

```bash
docker ps -a --format '{{.ID}} {{.Names}}' | grep backend-piposmart_api
```

Untuk `worker`:

```bash
docker ps -a --format '{{.ID}} {{.Names}}' | grep backend-piposmart_worker
```

Untuk `setup`:

```bash
docker ps -a --format '{{.ID}} {{.Names}}' | grep backend-piposmart_setup
```

Setelah nama container muncul, copy nama atau ID yang benar dari hasil command di atas, baru hapus:

```bash
docker rm -f <nama-atau-id-container>
```

Contoh:

```bash
docker rm -f backend-piposmart_api_1
docker rm -f backend-piposmart_worker_1
docker rm -f 4dcb203aaded_backend-piposmart_setup_1
```

Lalu buat ulang service yang dibutuhkan:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps api
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps worker
```

Kalau error `ContainerConfig` muncul saat habis update `.env.production`, pakai alur ini:

1. Edit dan simpan `.env.production`
2. Cari container lama:

   ```bash
   docker ps -a --format '{{.ID}} {{.Names}}' | grep backend-piposmart_api
   docker ps -a --format '{{.ID}} {{.Names}}' | grep backend-piposmart_worker
   ```

3. Hapus container yang memang mau di-recreate:

   ```bash
   docker rm -f <nama-atau-id-api>
   docker rm -f <nama-atau-id-worker>
   ```

4. Jalankan ulang:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --no-deps api
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --no-deps worker
   ```

5. Verifikasi:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production ps
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production logs --tail 50 api
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production logs --tail 50 worker
   ```

Kalau perlu recreate penuh setelah container lama dibersihkan:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build
```

## 6C. Menjalankan `seed demo` di VPS

Backend ini sengaja menolak `seed demo` ketika `APP_ENV=production`.

Kalau mencoba langsung:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup seed demo --preset=real
```

hasilnya akan ditolak dengan pesan:

```text
demo seeder ditolak pada environment production
```

Jadi sebelum menjalankan demo seed, yang harus dilakukan dulu adalah menonaktifkan mode production sementara.

### Opsi paling praktis

1. Edit env:

   ```bash
   nano .env.production
   ```

2. Ubah sementara:

   ```env
   APP_ENV=production
   ```

   menjadi:

   ```env
   APP_ENV=staging
   ```

3. Jalankan demo seed:

   ```bash
   docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup seed demo --preset=real
   ```

Contoh lain:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup seed demo --preset=minimal
docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup seed demo --preset=large --scale=2 --variation=0.5
docker-compose -f compose.prod.external-db.yaml --env-file .env.production run --rm setup seed demo --preset=real --scale=1 --variation=0.5
```

Setelah selesai, kembalikan lagi env ke production:

```env
APP_ENV=production
```

Lalu restart service utama:

```bash
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps api
docker-compose -f compose.prod.external-db.yaml --env-file .env.production up -d --build --no-deps worker
```

### Catatan penting

- Jangan jalankan `seed demo` pada database production live kecuali memang sengaja ingin mengisi data demo.
- `preset=real` tetap termasuk demo seed, bukan data production sungguhan.
- Perintah `run --rm setup seed demo ...` tidak perlu menulis `/app/crm` lagi, karena image backend sudah punya `ENTRYPOINT ["/app/crm"]`.

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
