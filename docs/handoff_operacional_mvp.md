# Handoff Operacional do MVP

## O que esta pronto

- jornada ponta a ponta local com `web`, `bff`, `workflow`, servicos de dominio, MinIO, Mailpit e `mock-oidc`;
- upload real de documentos, workflow assincrono, callbacks externos, auditoria e rate limit;
- observabilidade basica com `/metrics`, DLQ, overview operacional e smoke automatizado.

## Comandos principais

- subir stack: `powershell -ExecutionPolicy Bypass -File .\scripts\up_local_stack.ps1`
- derrubar stack: `powershell -ExecutionPolicy Bypass -File .\scripts\down_local_stack.ps1`
- validar codigo: `powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1`
- smoke ponta a ponta: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_docker_stack.ps1`
- overview operacional: `powershell -ExecutionPolicy Bypass -File .\scripts\ops_overview.ps1`
- readiness completa: `powershell -ExecutionPolicy Bypass -File .\scripts\release_readiness.ps1`

## Rotas de operacao

- front: `http://localhost:3000`
- BFF health: `http://localhost:18080/healthz`
- BFF metrics: `http://localhost:18080/metrics`
- BFF operations overview: `http://localhost:18080/internal/operations/overview`
- workflow health: `http://localhost:18084/healthz`
- workflow DLQ: `http://localhost:18084/internal/dlq`
- Mailpit: `http://localhost:8025`
- MinIO console: `http://localhost:9001`

## Quando atuar manualmente

- usar `ops_overview.ps1` para triagem inicial;
- consultar a auditoria do BFF antes de liberar replay;
- consultar a DLQ do workflow antes de reprocessar;
- executar `cleanup` da auditoria apenas depois de confirmar que nao ha investigacao ativa em curso.

## Limites conhecidos do MVP

- thresholds e alertas ainda sao locais e documentais, nao integrados a um stack de monitoramento externo;
- callbacks de parceiros seguem mockados no ambiente local;
- operacao esta otimizada para readiness de desenvolvimento e demonstracao, nao para producao regulada.

## Referencia de encerramento

- usar `docs/encerramento_mvp.md` como marco de fechamento do backlog do MVP e ponto de partida para um backlog `pos-mvp`.
