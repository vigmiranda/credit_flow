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

1. Subir o banco:
   `docker compose -f infra/docker/docker-compose.yml up -d`
2. Subir os servicos Go em terminais separados:
   `go run ./cmd/api` em `services/proposal`
   `go run ./cmd/api` em `services/customer`
   `go run ./cmd/api` em `services/document`
   `go run ./cmd/api` em `services/credit-analysis`
   `go run ./cmd/api` em `services/fraud-analysis`
   `go run ./cmd/api` em `services/notification`
   `go run ./cmd/api` em `services/workflow`
   `go run ./cmd/api` em `services/bff`
3. Subir o front:
   `npm install`
   `npm run dev` em `apps/web`

## Validacao rapida

- local: `powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1`
- smoke do MVP: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke_mvp.ps1`
- CI: `.github/workflows/validate.yml`
- build de imagens: `.github/workflows/build-images.yml`

## Observabilidade inicial

- `GET /metrics` disponivel em `bff`, `proposal`, `workflow` e `notification`
- logs estruturados em JSON com `correlation_id`, `path`, `status_code` e `duration_ms`

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
