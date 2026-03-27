$ErrorActionPreference = "Stop"

$goServices = @(
  "services/proposal",
  "services/customer",
  "services/document",
  "services/credit-analysis",
  "services/fraud-analysis",
  "services/notification",
  "services/workflow",
  "services/bff"
)

foreach ($service in $goServices) {
  Write-Host "==> go test ./$service"
  Push-Location $service
  try {
    go test ./...
  }
  finally {
    Pop-Location
  }
}

Write-Host "==> web typecheck"
Push-Location "apps/web"
try {
  npm.cmd run typecheck
  npm.cmd run build
}
finally {
  Pop-Location
}

