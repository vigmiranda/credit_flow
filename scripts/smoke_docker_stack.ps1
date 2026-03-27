$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Net.Http

$composeFile = "infra/docker/docker-compose.yml"
$bffBaseUrl = "http://localhost:18080"
$tempFile = Join-Path $env:TEMP "credit-flow-smoke-upload.jpg"

function Wait-ForHealth {
  param(
    [string]$Url,
    [int]$Retries = 120
  )

  for ($i = 0; $i -lt $Retries; $i++) {
    try {
      $response = Invoke-RestMethod -Uri $Url -Method Get -TimeoutSec 2
      if ($response.status -eq "ok") {
        return
      }
    }
    catch {
      Start-Sleep -Milliseconds 1000
    }
  }

  throw "Health check falhou para $Url"
}

function Send-MultipartFile {
  param(
    [string]$Url,
    [string]$FilePath,
    [string]$FileName,
    [string]$ContentType
  )

  $handler = [System.Net.Http.HttpClientHandler]::new()
  $client = [System.Net.Http.HttpClient]::new($handler)
  try {
    $content = [System.Net.Http.MultipartFormDataContent]::new()
    $fileBytes = [System.IO.File]::ReadAllBytes($FilePath)
    $fileContent = [System.Net.Http.ByteArrayContent]::new($fileBytes)
    $fileContent.Headers.ContentType = [System.Net.Http.Headers.MediaTypeHeaderValue]::Parse($ContentType)
    $content.Add($fileContent, "file", $FileName)

    $response = $client.PostAsync($Url, $content).GetAwaiter().GetResult()
    $raw = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    if (-not $response.IsSuccessStatusCode) {
      throw "Upload falhou com status $([int]$response.StatusCode): $raw"
    }

    return $raw | ConvertFrom-Json
  }
  finally {
    $client.Dispose()
  }
}

try {
  docker compose -f $composeFile up -d --build *> $null
  Wait-ForHealth "$bffBaseUrl/healthz"

  Write-Host "==> Criando proposta"
  $proposal = Invoke-RestMethod -Uri "$bffBaseUrl/api/v1/proposals" -Method Post

  Write-Host "==> Salvando cliente"
  Invoke-RestMethod -Uri "$bffBaseUrl/api/v1/proposals/$($proposal.proposal_id)/customer" -Method Post -ContentType "application/json" -Body (@{
    full_name = "Maria Silva"
    cpf = "12345678901"
    birth_date = "1990-01-01"
    email = "maria@example.com"
    phone = "11999999999"
    monthly_income = 5000
    address = "Rua Exemplo, 123"
  } | ConvertTo-Json) | Out-Null

  Write-Host "==> Registrando documento"
  $document = Invoke-RestMethod -Uri "$bffBaseUrl/api/v1/proposals/$($proposal.proposal_id)/documents/upload-url" -Method Post -ContentType "application/json" -Body (@{
    document_type = "id_front"
    file_name = "rg.jpg"
    content_type = "image/jpeg"
  } | ConvertTo-Json)

  [System.IO.File]::WriteAllBytes($tempFile, [System.Text.Encoding]::UTF8.GetBytes("fake-image-content"))

  Write-Host "==> Enviando arquivo real"
  $uploaded = Send-MultipartFile -Url "$bffBaseUrl/api/v1/proposals/$($proposal.proposal_id)/documents/$($document.document_id)/content" -FilePath $tempFile -FileName "rg.jpg" -ContentType "image/jpeg"
  if ($uploaded.status -ne "uploaded") {
    throw "Documento nao ficou com status uploaded"
  }

  Write-Host "==> Aguardando decisao final"
  $finalStatuses = @("approved", "rejected", "manual_review", "awaiting_additional_documents")
  $proposalState = $null
  for ($i = 0; $i -lt 90; $i++) {
    Start-Sleep -Milliseconds 1000
    $proposalState = Invoke-RestMethod -Uri "$bffBaseUrl/api/v1/proposals/$($proposal.proposal_id)" -Method Get
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
  Remove-Item $tempFile -ErrorAction SilentlyContinue
}
