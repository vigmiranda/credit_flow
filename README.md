# Credit Flow

Repositorio base para a plataforma de captura e analise de propostas de cartoes.

## Estrutura inicial

```text
/apps
  /web
/services
  /bff
  /proposal
  /customer
  /document
  /workflow
  /credit-analysis
  /fraud-analysis
  /notification
  /mock-oidc
/infra
  /docker
  /terraform
/docs
/planning
/use_case
```

## Objetivo do primeiro corte

Entregar um slice vertical demonstravel do fluxo de proposta:

- abertura de proposta;
- cadastro do cliente;
- upload de documentos;
- analises assincronas simuladas;
- consulta de status;
- notificacao basica.

## Documentos principais

- `planning/backlog_implementacao.md`: backlog vivo do projeto
- `docs/padroes_desenvolvimento.md`: convencoes de engenharia
- `docs/workflow_inicial.md`: fluxo do MVP para analises simuladas
- `docs/ambiente_local.md`: stack local e componentes auxiliares
- `docs/seguranca_operacional.md`: mascaramento, secrets via arquivo e criptografia
- `docs/alarmes_dashboards.md`: monitoracao minima para operacao
- `docs/estrategia_deploy.md`: fluxo recomendado de entrega por imagens
- `use_case/plano_implementacao_itau_cartoes.md`: plano macro original

## Servicos disponiveis no momento

- `services/proposal`: cria, consulta e atualiza status da proposta
- `services/customer`: cadastra e consulta cliente por proposta
- `services/document`: registra documentos, grava arquivos no MinIO e lista documentos
- `services/workflow`: enfileira e processa as analises simuladas do MVP com worker e retry
- `services/credit-analysis`: simulador de analise de credito
- `services/fraud-analysis`: simulador de analise de fraude
- `services/notification`: registra o historico de notificacoes e envia e-mail por SMTP local
- `services/mock-oidc`: issuer OIDC mockado para login real na stack local
- `services/bff`: agrega os servicos core para o front
- `apps/web`: jornada inicial em Next.js consumindo o BFF

## Subida local

1. Subir toda a stack com um unico comando:
   `powershell -ExecutionPolicy Bypass -File .\scripts\up_local_stack.ps1`
2. Abrir a aplicacao:
   `http://localhost:3000`
3. Derrubar a stack quando terminar:
   `powershell -ExecutionPolicy Bypass -File .\scripts\down_local_stack.ps1`

Se preferir rodar sem os scripts:
`docker compose -f infra/docker/docker-compose.yml up -d --build`

Pre-requisito:
`Docker Desktop` precisa estar ativo no host.

## Ambiente local auxiliar

- `postgres` em `localhost:15432`
- `redis` em `localhost:6379`
- `mailpit` UI em `http://localhost:8025`
- `minio` API em `http://localhost:9000`
- `minio` console em `http://localhost:9001`
- `mock-oidc` em `http://localhost:19090`
- `bff` em `http://localhost:18080`
- `workflow` operacional em `http://localhost:18084`
- `web` em `http://localhost:3000`
- servicos internos acessiveis apenas pela rede Docker da stack

## Autenticacao inicial

- `apps/web` agora protege `/` por cookie de sessao e redireciona para `/login`
- `AUTH_MODE=mock` habilita login operacional local
- `AUTH_MODE=oidc` habilita inicio de login via issuer configurado, troca de token e validacao criptografica do `id_token` por discovery/JWKS
- a stack Docker sobe por padrao com `AUTH_MODE=oidc` ligado ao `mock-oidc`

## Validacao rapida

- local: `powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1`
- smoke do MVP: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_mvp.ps1`
- smoke da stack Docker: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_docker_stack.ps1`
- CI: `.github/workflows/validate.yml`
- build de imagens: `.github/workflows/build-images.yml`

## Observabilidade inicial

- `GET /metrics` disponivel em `bff`, `proposal`, `workflow` e `notification`
- logs estruturados em JSON com `correlation_id`, `path`, `status_code` e `duration_ms`
- `workflow` usa Redis como fila quando `WORKFLOW_REDIS_URL` estiver configurado e cai para fila em memoria no fallback local
- `/metrics` do `workflow` agora inclui enqueue, process, retry, DLQ e profundidade de fila
- o `workflow` expoe `GET /internal/dlq` e `POST /internal/dlq/reprocess` para operacao local
- o `BFF` aceita webhooks autenticados de `storage`, `credit` e `fraud`, com `event-id` e janela anti-replay
- quando `BFF_REDIS_URL` estiver configurado, a deduplicacao dos webhooks passa a ser compartilhada entre replicas
- `/metrics` do `BFF` agora inclui contadores de callbacks por tipo, parceiro e status operacional
- o `BFF` persiste auditoria dos callbacks, correlaciona essa trilha com `GET /api/v1/proposals/{proposalId}` e expoe `GET /internal/webhooks/audit`, `POST /internal/webhooks/audit/{eventId}/replay-release` e `POST /internal/webhooks/audit/cleanup`

## Seguranca minima atual

- `GET /api/v1/proposals/{proposalId}` retorna `cpf`, `email`, `phone` e `recipient` mascarados
- `notification service` persiste destinatario mascarado e copia criptografada
- servicos stateful aceitam `DATABASE_URL` e equivalentes via variaveis `*_FILE`

## Enderecos padrao

- Front web: `http://localhost:3000`
- BFF: `http://localhost:18080`
- Workflow: `http://localhost:18084`
- Mailpit UI: `http://localhost:8025`
- MinIO Console: `http://localhost:9001`
- Mock OIDC: `http://localhost:19090`
- Webhook de storage: `POST /api/v1/webhooks/storage/document-uploaded` com `X-Webhook-Signature: sha256=<hex>`
- Webhook de credito: `POST /api/v1/webhooks/partners/credit-analysis`
- Webhook de antifraude: `POST /api/v1/webhooks/partners/fraud-analysis`
- Providers permitidos e janela por rota podem ser ajustados por `BFF_ALLOWED_*_PROVIDERS` e `BFF_*_WEBHOOK_MAX_AGE_SECONDS`
