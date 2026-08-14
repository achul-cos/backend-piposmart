Write-Host 'Clearing DB...'
go run . migrate clear
go run . migrate up
go run . seed master
go run . bootstrap-admin
Write-Host 'Seeding real...'
go run . seed demo --preset=real --seed=20260723 --from=2026-07-01 --to=2026-07-01
Write-Host 'DONE'
