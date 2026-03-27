# Eventos Assincronos

## Objetivo

Definir os eventos iniciais do fluxo de proposta para o MVP, preservando o modelo assincrono desde o primeiro corte.

## Regras gerais

- todos os eventos devem carregar `event_id`, `event_type`, `occurred_at` e `correlation_id`;
- `proposal_id` e obrigatorio em todos os eventos de negocio;
- payloads devem ser pequenos e orientados ao fato ocorrido, nao ao estado inteiro do agregado;
- consumidores devem ser idempotentes.

## Envelope padrao

```json
{
  "event_id": "evt_123",
  "event_type": "proposal.created",
  "occurred_at": "2026-03-27T10:00:00Z",
  "correlation_id": "corr_123",
  "proposal_id": "prop_123",
  "payload": {}
}
```

## Eventos do MVP

### `proposal.created`

Disparado quando a proposta e criada com sucesso.

Payload:

- `protocol`
- `status`

Consumidores iniciais:

- auditoria
- notificacao

### `customer.upserted`

Disparado quando os dados principais do cliente forem persistidos.

Payload:

- `customer_id`
- `cpf_masked`
- `email`

Consumidores iniciais:

- auditoria
- workflow

### `document.upload_url_requested`

Disparado quando a URL de upload e gerada.

Payload:

- `document_id`
- `document_type`
- `file_key`

Consumidores iniciais:

- auditoria

### `document.received`

Disparado apos a confirmacao de upload do documento.

Payload:

- `document_id`
- `document_type`
- `file_key`

Consumidores iniciais:

- workflow
- analise documental

### `document.analysis.completed`

Disparado ao finalizar a analise documental.

Payload:

- `document_id`
- `result`
- `reason`

Consumidores iniciais:

- workflow

### `credit.analysis.completed`

Disparado ao finalizar a analise de credito.

Payload:

- `result`
- `score`
- `reason`

Consumidores iniciais:

- workflow

### `fraud.analysis.completed`

Disparado ao finalizar a analise de fraude.

Payload:

- `result`
- `score`
- `reason`

Consumidores iniciais:

- workflow

### `proposal.status.changed`

Disparado sempre que a proposta muda de estado.

Payload:

- `previous_status`
- `current_status`
- `reason`

Consumidores iniciais:

- notificacao
- auditoria
- timeline

### `notification.requested`

Disparado para solicitar notificacao ao cliente.

Payload:

- `channel`
- `template`
- `variables`

Consumidores iniciais:

- notification service

