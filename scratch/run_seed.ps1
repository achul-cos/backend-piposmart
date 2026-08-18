$ErrorActionPreference = "Stop"
Write-Host "Running migrate clear..."
go run . migrate clear
Write-Host "Running bootstrap-admin..."
go run . bootstrap-admin
Write-Host "Running seed demo preset real..."
go run . seed demo --preset=real --seed=20260723 --from=2026-07-01 --to=2026-07-01
Write-Host "Done seeding!"
Write-Host "Starting API server..."
go run . api
