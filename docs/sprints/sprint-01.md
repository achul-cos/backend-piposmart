# Sprint 1 — Fondasi Project dan Container

## Sprint Goal

Backend dapat dijalankan secara konsisten dari environment lokal maupun
container, dengan konfigurasi tervalidasi dan health check yang dapat
diobservasi.

## Status

`AMBER` — seluruh quality gate Go dan validasi statis container lulus. Runtime
Docker belum dapat diuji karena Docker Engine tidak tersedia pada workstation;
Docker build tetap diwajibkan oleh CI.

## Deliverable

- [x] Entry point `cmd/crm`.
- [x] Config loader `.env` dan environment validation.
- [x] Structured JSON/text logging.
- [x] Explicit CORS, request ID, access log, dan panic recovery.
- [x] Liveness dan readiness check.
- [x] Graceful shutdown API dan worker.
- [x] Goose migration command.
- [x] `.env.example`, `.gitignore`, dan `.dockerignore`.
- [x] Multi-stage Dockerfile non-root.
- [x] Compose MySQL, migration, API, dan worker.
- [x] OpenAPI/Swagger untuk endpoint Sprint 1.
- [x] GitHub Actions test, vet, build, dan Docker build.
- [x] Dokumentasi setup lokal.
- [x] Entrypoint prototipe dinonaktifkan dan binary hasil build tidak lagi
  disimpan di Git.

## Demo Evidence

```text
GET /health/live
GET /health/ready
GET /api/v1/status
GET /swagger/index.html
```

## Quality Gate

- [x] `go test ./...`
- [x] `go vet ./...`
- [x] `go build ./cmd/crm`
- [ ] Dockerfile build
- [x] Compose YAML dan service validation melalui automated test

Docker dan Compose tidak tersedia pada workstation saat implementasi. Build
container tetap menjadi quality gate CI.

## Risiko dan Catatan

- Model serta route prototipe masih ada tetapi tidak dipakai entrypoint baru.
  Penghapusannya dilakukan bersamaan dengan fresh schema Sprint 2.
- Migration yang ada masih migration prototipe. Baseline schema ERD baru adalah
  deliverable Sprint 2.
- Worker saat ini hanya menjalankan lifecycle dan database heartbeat. Job queue
  domain ditambahkan pada Sprint berikutnya.
