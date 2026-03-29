param(
  [string]$BffBaseUrl = "http://localhost:18080"
)

$uri = "$BffBaseUrl/internal/operations/overview"

Write-Host "==> Consultando visao operacional em $uri"
try {
  $response = Invoke-RestMethod -Method Get -Uri $uri
} catch {
  Write-Error "Falha ao consultar overview operacional. Atualize a stack local e tente novamente."
  throw
}

Write-Host ""
Write-Host "Gerado em: $($response.generated_at)"
Write-Host "DLQ: $($response.workflow_dead_letters.count)"
Write-Host "Callbacks recentes: $($response.callback_audit_summary.recent_count)"
Write-Host "Rate limited: $($response.callback_audit_summary.rate_limited)"
Write-Host "Invalid provider: $($response.callback_audit_summary.invalid_provider)"

if ($response.workflow_metrics) {
  Write-Host ""
  Write-Host "Workflow queue:"
  Write-Host "  Enqueued: $($response.workflow_metrics.queue.enqueued)"
  Write-Host "  Processed: $($response.workflow_metrics.queue.processed)"
  Write-Host "  Retried: $($response.workflow_metrics.queue.retried)"
  Write-Host "  Dead letter: $($response.workflow_metrics.queue.dead_letter)"
  Write-Host "  DLQ depth: $($response.workflow_metrics.queue.dlq_depth)"
}

Write-Host ""
if ($response.alerts.Count -eq 0) {
  Write-Host "Sem alertas operacionais."
} else {
  Write-Host "Alertas:"
  foreach ($alert in $response.alerts) {
    Write-Host "  [$($alert.severity)] $($alert.code) - $($alert.message)"
  }
}
