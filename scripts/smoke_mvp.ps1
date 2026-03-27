$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$smokeDbPort = 55432
$smokeDbUrl = "postgres://credit_flow:credit_flow@localhost:$smokeDbPort/credit_flow?sslmode=disable"
$smokeDbContainer = "credit-flow-smoke-postgres"
$services = @(
  @{ Name = "proposal"; Path = Join-Path $root "services/proposal"; Port = 8081; Env = @{ PROPOSAL_SERVICE_DATABASE_URL = $smokeDbUrl } },
  @{ Name = "customer"; Path = Join-Path $root "services/customer"; Port = 8082; Env = @{ CUSTOMER_SERVICE_DATABASE_URL = $smokeDbUrl } },
  @{ Name = "document"; Path = Join-Path $root "services/document"; Port = 8083; Env = @{ DOCUMENT_SERVICE_DATABASE_URL = $smokeDbUrl } },
  @{ Name = "credit-analysis"; Path = Join-Path $root "services/credit-analysis"; Port = 8085; Env = @{} },
  @{ Name = "fraud-analysis"; Path = Join-Path $root "services/fraud-analysis"; Port = 8086; Env = @{} },
  @{ Name = "notification"; Path = Join-Path $root "services/notification"; Port = 8087; Env = @{ NOTIFICATION_SERVICE_DATABASE_URL = $smokeDbUrl } },
  @{ Name = "workflow"; Path = Join-Path $root "services/workflow"; Port = 8084; Env = @{ WORKFLOW_SERVICE_PROPOSAL_URL = "http://localhost:8081"; WORKFLOW_SERVICE_CUSTOMER_URL = "http://localhost:8082"; WORKFLOW_SERVICE_DOCUMENT_URL = "http://localhost:8083"; WORKFLOW_SERVICE_CREDIT_URL = "http://localhost:8085"; WORKFLOW_SERVICE_FRAUD_URL = "http://localhost:8086"; WORKFLOW_SERVICE_NOTIFICATION_URL = "http://localhost:8087" } },
  @{ Name = "bff"; Path = Join-Path $root "services/bff"; Port = 8080; Env = @{ PROPOSAL_SERVICE_URL = "http://localhost:8081"; CUSTOMER_SERVICE_URL = "http://localhost:8082"; DOCUMENT_SERVICE_URL = "http://localhost:8083"; WORKFLOW_SERVICE_URL = "http://localhost:8084"; NOTIFICATION_SERVICE_URL = "http://localhost:8087" } }
)

$processes = @()
$logFiles = @{}

function Wait-ForHealth {
  param(
    [string]$Url,
    [int]$Retries = 80
  )

  for ($i = 0; $i -lt $Retries; $i++) {
    try {
      $response = Invoke-RestMethod -Uri $Url -Method Get -TimeoutSec 2
      if ($response.status -eq "ok") {
        return
      }
    }
    catch {
      Start-Sleep -Milliseconds 500
    }
  }

  throw "Health check falhou para $Url"
}

function Test-TcpPort {
  param(
    [string]$HostName,
    [int]$Port
  )

  try {
    $client = New-Object System.Net.Sockets.TcpClient
    $async = $client.BeginConnect($HostName, $Port, $null, $null)
    $wait = $async.AsyncWaitHandle.WaitOne(500)
    if (-not $wait) {
      $client.Close()
      return $false
    }
    $client.EndConnect($async)
    $client.Close()
    return $true
  }
  catch {
    return $false
  }
}

try {
  Write-Host "==> Subindo PostgreSQL dedicado do smoke test em localhost:$smokeDbPort"
  docker rm -f $smokeDbContainer | Out-Null 2>$null
  docker run -d --name $smokeDbContainer -e POSTGRES_DB=credit_flow -e POSTGRES_USER=credit_flow -e POSTGRES_PASSWORD=credit_flow -p "${smokeDbPort}:5432" postgres:16-alpine | Out-Null
  Start-Sleep -Seconds 3

  foreach ($service in $services) {
    Write-Host "==> Iniciando $($service.Name)"
    $stdout = Join-Path $env:TEMP "credit-flow-$($service.Name)-stdout.log"
    $stderr = Join-Path $env:TEMP "credit-flow-$($service.Name)-stderr.log"
    $logFiles[$service.Port] = @{ stdout = $stdout; stderr = $stderr; name = $service.Name }

    $envAssignments = @()
    foreach ($key in $service.Env.Keys) {
      $value = $service.Env[$key].Replace("'", "''")
      $envAssignments += "`$env:$key='$value'"
    }
    $envPrefix = ""
    if ($envAssignments.Count -gt 0) {
      $envPrefix = ($envAssignments -join "; ") + "; "
    }
    $command = $envPrefix + "go run ./cmd/api"
    $process = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile", "-Command", $command -WorkingDirectory $service.Path -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    $processes += $process
  }

  foreach ($service in $services) {
    try {
      Wait-ForHealth "http://localhost:$($service.Port)/healthz"
    }
    catch {
      if ($logFiles.ContainsKey($service.Port)) {
        Write-Host "==> stdout $($service.Name)"
        if (Test-Path $logFiles[$service.Port].stdout) {
          Get-Content $logFiles[$service.Port].stdout
        }
        Write-Host "==> stderr $($service.Name)"
        if (Test-Path $logFiles[$service.Port].stderr) {
          Get-Content $logFiles[$service.Port].stderr
        }
      }
      throw
    }
  }

  Write-Host "==> Criando proposta"
  $proposal = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/proposals" -Method Post

  Write-Host "==> Salvando cliente"
  Invoke-RestMethod -Uri "http://localhost:8080/api/v1/proposals/$($proposal.proposal_id)/customer" -Method Post -ContentType "application/json" -Body (@{
    full_name = "Maria Silva"
    cpf = "12345678901"
    birth_date = "1990-01-01"
    email = "maria@example.com"
    phone = "11999999999"
    monthly_income = 5000
    address = "Rua Exemplo, 123"
  } | ConvertTo-Json) | Out-Null

  Write-Host "==> Gerando upload"
  $document = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/proposals/$($proposal.proposal_id)/documents/upload-url" -Method Post -ContentType "application/json" -Body (@{
    document_type = "id_front"
    file_name = "rg.jpg"
    content_type = "image/jpeg"
  } | ConvertTo-Json)

  Write-Host "==> Confirmando documento"
  Invoke-RestMethod -Uri "http://localhost:8080/api/v1/proposals/$($proposal.proposal_id)/documents/$($document.document_id)/received" -Method Post | Out-Null

  Write-Host "==> Aguardando decisao final"
  $finalStatuses = @("approved", "rejected", "manual_review", "awaiting_additional_documents")
  $proposalState = $null
  for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Milliseconds 500
    $proposalState = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/proposals/$($proposal.proposal_id)" -Method Get
    if ($finalStatuses -contains $proposalState.status) {
      break
    }
  }

  if ($null -eq $proposalState -or -not ($finalStatuses -contains $proposalState.status)) {
    throw "Proposta nao chegou a um status final no tempo esperado"
  }

  Write-Host "==> Resultado final: $($proposalState.status)"
  Write-Host "==> Analises registradas: $($proposalState.analysis_results.Count)"
  Write-Host "==> Eventos de timeline: $($proposalState.status_history.Count)"
  Write-Host "==> Notificacoes: $($proposalState.notifications.Count)"
}
finally {
  foreach ($process in $processes) {
    try {
      if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
      }
    }
    catch {
    }
  }
  docker rm -f $smokeDbContainer | Out-Null 2>$null
}
