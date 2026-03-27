# Ambiente Local

## Stack local atual

- `postgres`: persistencia principal do MVP
- `redis`: cache e futura coordenacao assíncrona
- `mailpit`: caixa SMTP local para evolucao das notificacoes
- `minio`: storage S3-compatible para evolucao do fluxo documental

## Subida rapida

```powershell
docker compose -f infra/docker/docker-compose.yml up -d
```

## Enderecos padrao

- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Mailpit SMTP: `localhost:1025`
- Mailpit UI: `http://localhost:8025`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

## Observacoes

- o MVP atual continua funcional mesmo sem Redis, Mailpit e MinIO consumidos diretamente em codigo;
- os componentes foram adicionados para preparar autenticacao, notificacao e storage real nos proximos ciclos;
- o smoke test segue isolado do stack principal para evitar conflito de portas e dados.
