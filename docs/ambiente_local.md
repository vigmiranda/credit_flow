# Ambiente Local

## Stack local atual

- `postgres`: persistencia principal do MVP
- `redis`: cache e futura coordenacao assincrona
- `mailpit`: caixa SMTP local para notificacoes
- `minio`: storage S3-compatible para upload real do fluxo documental

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
- Web: `http://localhost:3000`

## Observacoes

- o `docker-compose` sobe toda a stack local do MVP, incluindo front e servicos Go;
- os servicos internos Go ficam acessiveis pela rede Docker e expostos ao host apenas via `web` e `bff`;
- o upload documental usa o bucket `proposal-documents` no MinIO;
- o `workflow service` usa Redis como fila de processamento e reaplica retries configuraveis;
- as notificacoes do `notification service` sao entregues por SMTP local no Mailpit;
- `scripts/smoke_docker_stack.ps1` valida a stack Docker ponta a ponta sem precisar subir processos Go manuais;
- o smoke test isolado segue disponivel em `scripts/smoke_mvp.ps1` para diagnostico fora do compose.
