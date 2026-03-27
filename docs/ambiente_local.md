# Ambiente Local

## Stack local atual

- `postgres`: persistencia principal do MVP
- `redis`: cache e futura coordenacao assíncrona
- `mailpit`: caixa SMTP local para evolucao das notificacoes
- `minio`: storage S3-compatible para evolucao do fluxo documental

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

- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Mailpit SMTP: `localhost:1025`
- Mailpit UI: `http://localhost:8025`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`
- Proposal Service: `http://localhost:8081`
- Customer Service: `http://localhost:8082`
- Document Service: `http://localhost:8083`
- Workflow Service: `http://localhost:8084`
- Credit Analysis Service: `http://localhost:8085`
- Fraud Analysis Service: `http://localhost:8086`
- Notification Service: `http://localhost:8087`
- BFF: `http://localhost:8080`
- Web: `http://localhost:3000`

## Observacoes

- o `docker-compose` agora sobe toda a stack local do MVP, incluindo front e servicos Go;
- `Redis`, `Mailpit` e `MinIO` ainda entram como componentes auxiliares para os proximos ciclos;
- o smoke test segue isolado do stack principal para evitar conflito de portas e dados.
