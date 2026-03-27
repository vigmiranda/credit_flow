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
- `use_case/plano_implementacao_itau_cartoes.md`: plano macro original

## Servicos disponiveis no momento

- `services/proposal`: cria, consulta e atualiza status da proposta
- `services/customer`: cadastra e consulta cliente por proposta
- `services/document`: gera upload URL, lista documentos e confirma recebimento
- `services/bff`: agrega os servicos core para o front
- `apps/web`: jornada inicial em Next.js consumindo o BFF

## Subida local

1. Subir o banco:
   `docker compose -f infra/docker/docker-compose.yml up -d`
2. Subir os servicos Go em terminais separados:
   `go run ./cmd/api` em `services/proposal`
   `go run ./cmd/api` em `services/customer`
   `go run ./cmd/api` em `services/document`
   `go run ./cmd/api` em `services/bff`
3. Subir o front:
   `npm install`
   `npm run dev` em `apps/web`

## Enderecos padrao

- Front web: `http://localhost:3000`
- BFF: `http://localhost:8080`
- Proposal Service: `http://localhost:8081`
- Customer Service: `http://localhost:8082`
- Document Service: `http://localhost:8083`
