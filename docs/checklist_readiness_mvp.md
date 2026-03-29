# Checklist de Readiness do MVP

## Pre-requisitos

- Docker Desktop funcional com engine Linux ativo;
- `go`, `npm` e PowerShell disponiveis no ambiente;
- portas `3000`, `18080`, `18084`, `19090`, `15432`, `6379`, `9000`, `9001`, `8025` livres;
- variaveis de ambiente locais coerentes com `.env.example`.

## Validacao tecnica

- `powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1`
- `docker compose -f infra/docker/docker-compose.yml config`
- `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_docker_stack.ps1`
- `powershell -ExecutionPolicy Bypass -File .\scripts\ops_overview.ps1`

## Comando unico recomendado

- `powershell -ExecutionPolicy Bypass -File .\scripts\release_readiness.ps1`

## Criterios de aceite

- front acessivel em `http://localhost:3000`;
- `BFF` saudavel em `http://localhost:18080/healthz`;
- `workflow` saudavel em `http://localhost:18084/healthz`;
- smoke ponta a ponta finaliza proposta em status terminal;
- `ops overview` sem alertas criticos e com `DLQ = 0`;
- Mailpit e MinIO acessiveis;
- backlog atualizado com o ciclo fechado.

## Sinais de nao prontidao

- qualquer erro no `verify.ps1`;
- `docker compose config` invalido;
- proposta nao chega a status final no smoke;
- `ops overview` com alerta critico;
- `workflow_dead_letters.count > 0`.
