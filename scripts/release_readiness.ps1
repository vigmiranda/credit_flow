param(
  [switch]$SkipVerify,
  [switch]$SkipSmoke,
  [switch]$SkipOverview,
  [switch]$KeepStackUp
)

$ErrorActionPreference = "Stop"

Write-Host "==> Validando docker compose"
docker compose -f infra/docker/docker-compose.yml config | Out-Null

if (-not $SkipVerify) {
  Write-Host "==> Executando verify.ps1"
  powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1
}

Write-Host "==> Subindo stack local"
powershell -ExecutionPolicy Bypass -File .\scripts\up_local_stack.ps1

try {
  if (-not $SkipSmoke) {
    Write-Host "==> Executando smoke_docker_stack.ps1"
    powershell -ExecutionPolicy Bypass -File .\scripts\smoke_docker_stack.ps1
  }

  if (-not $SkipOverview) {
    Write-Host "==> Coletando overview operacional"
    powershell -ExecutionPolicy Bypass -File .\scripts\ops_overview.ps1
  }
}
finally {
  if (-not $KeepStackUp) {
    Write-Host "==> Derrubando stack local"
    powershell -ExecutionPolicy Bypass -File .\scripts\down_local_stack.ps1
  }
}

Write-Host "==> Readiness final concluida"
