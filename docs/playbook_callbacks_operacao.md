# Playbook de Operacao de Callbacks

## Objetivo

Padronizar a resposta operacional para falhas, bursts e reprocessamentos dos callbacks de `storage`, `credit` e `fraud`.

## Sinais principais

- `GET http://localhost:18080/metrics`: acompanhar contadores em `webhooks` e `webhook_cleanup`;
- `GET http://localhost:18080/internal/webhooks/audit?proposal_id=<proposal_id>`: correlacionar eventos recebidos com a proposta;
- `GET http://localhost:18084/internal/dlq`: inspecionar jobs esgotados do workflow;
- `POST http://localhost:18080/internal/webhooks/audit/{eventId}/replay-release`: liberar replay manual de um evento deduplicado;
- `POST http://localhost:18080/internal/webhooks/audit/cleanup`: remover registros de auditoria expirados;
- `POST http://localhost:18084/internal/dlq/reprocess`: reencaminhar item da DLQ no workflow.

## Classificacao rapida

- `invalid_signature`, `invalid_timestamp`, `stale_webhook`: problema de autenticacao ou janela temporal do parceiro;
- `invalid_provider`: callback chegou por rota valida, mas com provider fora da allowlist configurada;
- `rate_limited`: burst acima do permitido para a combinacao rota + parceiro;
- `duplicate_ignored`: replay bloqueado pelo dedupe compartilhado;
- `replay_store_error`, `rate_limit_store_error`: indisponibilidade da infraestrutura de controle no BFF.

## Procedimento para burst ou rate limit

1. Verificar `webhooks.outcome:rate_limited` e os contadores por `type:<rota>|outcome:rate_limited`.
2. Confirmar no audit se o burst veio de um parceiro especifico por `provider`.
3. Validar se o limite atual esta coerente nas variaveis `BFF_*_RATE_LIMIT` e `BFF_*_RATE_WINDOW_SECONDS`.
4. Se necessario, aliviar temporariamente o limite e repetir o envio do parceiro.
5. Se algum evento valido ficou bloqueado, reprocessar pelo parceiro com novo `event_id` ou liberar replay manual quando aplicavel.

## Procedimento para replay e deduplicacao

1. Consultar `GET /internal/webhooks/audit?event_id=<eventId>`.
2. Se o evento estiver como `duplicate_ignored` e precisar ser refeito, executar `POST /internal/webhooks/audit/{eventId}/replay-release`.
3. Solicitar novo envio do callback ou repetir o teste com o mesmo `event_id`.
4. Confirmar no audit a transicao para `processed` ou a causa final de rejeicao.

## Procedimento para limpeza de auditoria

1. Verificar `retention_expires_at` nos registros antigos.
2. Executar `POST /internal/webhooks/audit/cleanup`.
3. Conferir `webhook_cleanup.runs` e `webhook_cleanup.removed_total` em `/metrics`.
4. Repetir a limpeza depois de incidentes extensos ou em rotina operacional agendada.

## Procedimento para DLQ do workflow

1. Listar itens em `GET /internal/dlq`.
2. Cruzar `proposal_id` com a timeline e a auditoria de callbacks no BFF.
3. Corrigir a causa raiz: provider, payload, dependencia externa ou estado da proposta.
4. Reprocessar via `POST /internal/dlq/reprocess`.
5. Validar a proposta final no front ou em `GET /api/v1/proposals/{proposalId}`.
