# Ambiente Local

## Stack local atual

- `postgres`: persistencia principal do MVP
- `redis`: cache e futura coordenacao assincrona
- `mailpit`: caixa SMTP local para notificacoes
- `minio`: storage S3-compatible para upload real do fluxo documental
- `mock-oidc`: issuer local para validar o fluxo OIDC do front

## Subida rapida

Pre-requisito: `Docker Desktop` ativo no host.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\up_local_stack.ps1
```

Ou diretamente:

```powershell
docker compose -f infra/docker/docker-compose.yml up -d --build
```

## Enderecos padrao

- PostgreSQL: `localhost:15432`
- Redis: `localhost:6379`
- Mailpit SMTP: `localhost:1025`
- Mailpit UI: `http://localhost:8025`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`
- BFF: `http://localhost:18080`
- Workflow: `http://localhost:18084`
- Mock OIDC: `http://localhost:19090`
- Web: `http://localhost:3000`

## Observacoes

- o `docker-compose` sobe toda a stack local do MVP, incluindo front e servicos Go;
- os servicos internos Go ficam acessiveis pela rede Docker e expostos ao host via `web`, `bff`, `workflow` e `mock-oidc`;
- o upload documental usa o bucket `proposal-documents` no MinIO;
- o `workflow service` usa Redis como fila de processamento e reaplica retries configuraveis;
- o `workflow service` registra DLQ e profundidade da fila em `/metrics`;
- a inspecao da DLQ fica disponivel em `GET http://localhost:18084/internal/dlq` e o reprocessamento em `POST http://localhost:18084/internal/dlq/reprocess`;
- os callbacks externos podem ser ligados por `WORKFLOW_EXTERNAL_CREDIT_CALLBACKS=true` e `WORKFLOW_EXTERNAL_FRAUD_CALLBACKS=true`;
- o `BFF` usa `redis://redis:6379/2` na stack Docker para deduplicacao compartilhada dos webhooks;
- o `BFF` persiste auditoria em Redis com prefixo `bff:webhook:audit` e retencao configuravel por `BFF_WEBHOOK_AUDIT_RETENTION_SECONDS`;
- as notificacoes do `notification service` sao entregues por SMTP local no Mailpit;
- o front sobe em modo `oidc` contra o `mock-oidc`, mas pode voltar para `AUTH_MODE=mock` se necessario;
- `scripts/smoke_docker_stack.ps1` valida a stack Docker ponta a ponta sem precisar subir processos Go manuais;
- o smoke test isolado segue disponivel em `scripts/smoke_mvp.ps1` para diagnostico fora do compose.
