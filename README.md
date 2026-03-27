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
- `services/document`: gera upload URL, lista documentos e confirma recebimento
- `services/workflow`: orquestra as analises simuladas do MVP
- `services/credit-analysis`: simulador de analise de credito
- `services/fraud-analysis`: simulador de analise de fraude
- `services/notification`: registra o historico de notificacoes simuladas
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

- `postgres` em `localhost:5432`
- `redis` em `localhost:6379`
- `mailpit` UI em `http://localhost:8025`
- `minio` API em `http://localhost:9000`
- `minio` console em `http://localhost:9001`
- `proposal` em `http://localhost:8081`
- `customer` em `http://localhost:8082`
- `document` em `http://localhost:8083`
- `workflow` em `http://localhost:8084`
- `credit-analysis` em `http://localhost:8085`
- `fraud-analysis` em `http://localhost:8086`
- `notification` em `http://localhost:8087`
- `bff` em `http://localhost:8080`
- `web` em `http://localhost:3000`

## Autenticacao inicial

- `apps/web` agora protege `/` por cookie de sessao e redireciona para `/login`
- `AUTH_MODE=mock` habilita login operacional local
- `AUTH_MODE=oidc` preserva a configuracao-base para integracao futura com issuer real

## Validacao rapida

- local: `powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1`
- smoke do MVP: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_mvp.ps1`
- CI: `.github/workflows/validate.yml`
- build de imagens: `.github/workflows/build-images.yml`

## Observabilidade inicial

- `GET /metrics` disponivel em `bff`, `proposal`, `workflow` e `notification`
- logs estruturados em JSON com `correlation_id`, `path`, `status_code` e `duration_ms`

## Seguranca minima atual

- `GET /api/v1/proposals/{proposalId}` retorna `cpf`, `email`, `phone` e `recipient` mascarados
- `notification service` persiste destinatario mascarado e copia criptografada
- servicos stateful aceitam `DATABASE_URL` e equivalentes via variaveis `*_FILE`

## Enderecos padrao

- Front web: `http://localhost:3000`
- BFF: `http://localhost:8080`
- Proposal Service: `http://localhost:8081`
- Customer Service: `http://localhost:8082`
- Document Service: `http://localhost:8083`
- Workflow Service: `http://localhost:8084`
- Credit Analysis Service: `http://localhost:8085`
- Fraud Analysis Service: `http://localhost:8086`
- Notification Service: `http://localhost:8087`
