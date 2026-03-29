# Alarmes e Dashboards Minimos

## Fontes

- logs estruturados JSON com `correlation_id`;
- endpoint `/metrics` nos servicos `bff`, `proposal`, `workflow` e `notification`;
- smoke test ponta a ponta em `scripts/smoke_mvp.ps1`.

## Dashboard operacional minimo

- throughput por servico: `total_requests`;
- erros por servico: `total_errors`;
- latencia media por servico: `average_ms`;
- requisicoes em voo: `inflight_requests`;
- distribuicao por rota a partir do mapa `paths`.

## Dashboard operacional expandido

- `workflow.queue.dlq_depth`: backlog real de DLQ a ser tratado;
- `workflow.queue.retried`: tendencia de retries por janela;
- `bff.webhooks.outcome:rate_limited`: bursts bloqueados por rate limit;
- `bff.webhooks.outcome:invalid_provider`: rejeicoes por politica de parceiro;
- `bff.webhooks.outcome:duplicate_ignored`: callbacks deduplicados;
- `bff.webhook_cleanup.runs` e `bff.webhook_cleanup.removed_total`: higiene da auditoria;
- `GET /internal/operations/overview`: resumo consolidado para triagem manual.

## Alarmes minimos

- `bff` indisponivel por mais de 2 minutos;
- `workflow` com `total_errors` crescente por 5 minutos;
- `proposal` com latencia media acima de 500 ms por 10 minutos;
- `notification` com erro em mais de 5% das requisicoes no periodo de 10 minutos;
- falha do smoke test apos deploy em `dev`.

## Thresholds recomendados

- `workflow.queue.dlq_depth > 0`: alerta critico imediato;
- `workflow.queue.retried > 0` por 10 minutos: alerta warning;
- `bff.webhooks.outcome:rate_limited > 5` em 5 minutos por parceiro: alerta warning;
- `bff.webhooks.outcome:invalid_provider > 0`: alerta warning imediato;
- `bff.webhook_cleanup.runs = 0` por mais de 24h em ambiente ativo: alerta info;
- `operations overview` com `alerts.severity=critical`: abrir incidente operacional.

## Acao operacional recomendada

- correlacionar erro pelo `X-Correlation-Id` do front ate os servicos internos;
- validar saude em `/healthz` antes de reciclar instancia;
- usar o smoke test como gate de aceite apos rollout;
- consultar `GET /internal/operations/overview` ou `scripts/ops_overview.ps1` antes de atuar manualmente;
- abrir incidente apenas se houver impacto confirmado no fluxo de criacao, envio de documento ou consolidacao final.
