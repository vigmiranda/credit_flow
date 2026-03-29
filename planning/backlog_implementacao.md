# Backlog de Implementacao - Plataforma de Propostas de Cartoes

## Objetivo

Transformar o plano macro em um backlog incremental, com entregas pequenas, rastreaveis e atualizaveis a cada ciclo concluido.

## Regra de manutencao

Este arquivo deve ser atualizado ao final de cada ciclo com:

- status das entregas;
- decisoes tomadas;
- riscos novos ou removidos;
- proximo ciclo priorizado.

## Status gerais

- `pendente`: ainda nao iniciado
- `em_andamento`: iniciado no ciclo atual
- `concluido`: entregue e validado
- `bloqueado`: depende de definicao ou entrega externa

## Epicos

### Epico 1 - Fundacao do projeto

Objetivo: preparar a base tecnica e organizacional para iniciar o desenvolvimento.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E1-01 | Definir escopo fechado do MVP | concluido | Nenhuma | Lista objetiva do que entra e do que fica fora da primeira entrega |
| E1-02 | Definir estrutura inicial do repositorio | concluido | E1-01 | Organizacao base de `apps`, `services`, `infra` e `docs/planning` |
| E1-03 | Definir padroes de desenvolvimento | concluido | E1-02 | Convencoes de codigo, branch, commit, env e observabilidade minima |
| E1-04 | Preparar ambiente local minimo | concluido | E1-02 | Stack local com banco, cache e servicos auxiliares |

### Epico 2 - Contratos e dominio

Objetivo: estabilizar os contratos antes da implementacao dos servicos.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E2-01 | Modelar entidades principais | concluido | E1-01 | `Proposal`, `Customer`, `Document`, `AnalysisResult`, `Notification` |
| E2-02 | Definir maquina de estados da proposta | concluido | E2-01 | Fluxo de status, transicoes validas e eventos disparados |
| E2-03 | Definir contratos de API | concluido | E2-01, E2-02 | OpenAPI inicial para front, BFF e servicos |
| E2-04 | Definir eventos assincronos | concluido | E2-02 | Eventos de dominio com payload, origem e consumidores |
| E2-05 | Definir padroes transversais | concluido | E2-03 | Correlation-id, erros padronizados, idempotencia e auditoria |

## Escopo fechado do MVP

### Itens que entram no MVP

- autenticacao inicial do usuario em modo simplificado, preparada para evolucao para OIDC/OAuth2;
- abertura de proposta com geracao de identificador unico;
- cadastro basico do cliente com validacoes essenciais;
- upload de documentos via URL assinada ou simulacao equivalente em ambiente local;
- processamento assincrono inicial para documento, credito e fraude com respostas simuladas;
- consolidacao do status da proposta;
- consulta de status pelo front;
- notificacao basica por mudanca de status;
- trilha minima de auditoria por proposta;
- observabilidade minima com logs estruturados e correlation-id.

### Itens que ficam fora do MVP

- integracoes reais com bureau de credito;
- integracoes reais com motores antifraude;
- OCR ou analise documental real;
- revisao manual operacional;
- motor de regras configuravel por negocio;
- omnicanal completo de notificacoes;
- dashboards executivos e analytics avancado;
- estrategia completa de alta disponibilidade em producao;
- hardening completo de seguranca e LGPD em nivel produtivo.

### Criterios de aceite do MVP

- o usuario consegue criar uma proposta do inicio ao fim sem intervencao manual no banco;
- a proposta transita por estados claros e auditaveis;
- o front consegue consultar e exibir o status consolidado;
- falhas nas analises assincronas nao quebram a jornada principal e geram status tratavel;
- o fluxo pode ser demonstrado localmente com dependencias controladas.

## Decisoes de arquitetura para o MVP

- monorepo inicial com separacao por `apps`, `services`, `infra` e `planning`;
- backend em Go com REST para o primeiro corte;
- front em Next.js com TypeScript;
- banco relacional como fonte principal de verdade para proposta e cliente;
- mensageria ou simulacao de fila local para preservar o modelo assincrono desde o inicio;
- mocks para provedores externos de credito, fraude e documentos;
- foco em entregar um slice vertical antes de expandir cobertura funcional.

## Estrutura inicial recomendada do repositorio

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
/planning
/docs
```

## Modelagem inicial de entidades

### Proposal

- `id`
- `protocol`
- `customer_id`
- `status`
- `created_at`
- `updated_at`
- `submitted_at`
- `decision_at`
- `correlation_id`

### Customer

- `id`
- `proposal_id`
- `full_name`
- `cpf`
- `birth_date`
- `email`
- `phone`
- `monthly_income`
- `address`
- `created_at`
- `updated_at`

### Document

- `id`
- `proposal_id`
- `type`
- `file_key`
- `status`
- `uploaded_at`
- `analyzed_at`

### AnalysisResult

- `id`
- `proposal_id`
- `analysis_type`
- `status`
- `provider`
- `score`
- `reason`
- `created_at`

### Notification

- `id`
- `proposal_id`
- `channel`
- `template`
- `status`
- `sent_at`
- `error_message`

## Maquina inicial de estados da proposta

### Estados

- `created`
- `customer_data_pending`
- `documents_pending`
- `documents_received`
- `document_analysis_in_progress`
- `credit_analysis_in_progress`
- `fraud_analysis_in_progress`
- `manual_review`
- `approved`
- `rejected`
- `awaiting_additional_documents`

### Transicoes principais

- `created` -> `customer_data_pending`
- `customer_data_pending` -> `documents_pending`
- `documents_pending` -> `documents_received`
- `documents_received` -> `document_analysis_in_progress`
- `document_analysis_in_progress` -> `credit_analysis_in_progress`
- `credit_analysis_in_progress` -> `fraud_analysis_in_progress`
- `fraud_analysis_in_progress` -> `approved`
- `fraud_analysis_in_progress` -> `rejected`
- `document_analysis_in_progress` -> `awaiting_additional_documents`
- qualquer etapa de analise -> `manual_review`

### Observacoes

- no MVP, as analises podem rodar em paralelo na implementacao, mas o status consolidado deve permanecer simples para demonstracao;
- `manual_review` entra como estado previsto no dominio, mesmo que a operacao manual ainda nao seja implementada;
- `awaiting_additional_documents` cobre a necessidade de pendencia sem reabrir o fluxo inteiro.

### Epico 3 - Front web inicial

Objetivo: habilitar a jornada principal do usuario no MVP.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E3-01 | Criar aplicacao web base | concluido | E1-02 | App Next.js com estrutura inicial |
| E3-02 | Implementar autenticacao inicial | concluido | E3-01 | Fluxo mockado ou OIDC-ready para evolucao |
| E3-03 | Criar jornada de abertura de proposta | concluido | E3-01, E2-03 | Tela e fluxo para iniciar proposta |
| E3-04 | Criar jornada de cadastro do cliente | concluido | E3-03, E2-03 | Formularios com validacao |
| E3-05 | Criar jornada de upload de documentos | concluido | E3-03, E2-03 | Upload via URL assinada |
| E3-06 | Criar consulta de status da proposta | concluido | E3-03, E2-03 | Tela com status, pendencias e decisao |

### Epico 4 - Backend core

Objetivo: entregar o slice funcional principal da proposta.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E4-01 | Criar BFF inicial | concluido | E2-03 | API unica para o front |
| E4-02 | Implementar Proposal Service | concluido | E2-01, E2-02 | Criacao, consulta e atualizacao da proposta |
| E4-03 | Implementar Customer Service | concluido | E2-01, E2-03 | Cadastro e validacoes basicas de cliente |
| E4-04 | Implementar Document Service | concluido | E2-01, E2-04 | URL assinada, metadados e evento de upload |
| E4-05 | Persistencia inicial | concluido | E4-02, E4-03, E4-04 | Migrations e armazenamento transacional |

### Epico 5 - Workflow e analises assincronas

Objetivo: suportar o processamento desacoplado do fluxo de decisao.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E5-01 | Desenhar workflow inicial da proposta | concluido | E2-02, E2-04 | Fluxo orquestrado do recebimento ate decisao |
| E5-02 | Implementar analise documental simulada | concluido | E5-01, E4-04 | Resultado assincrono inicial para destravar fluxo |
| E5-03 | Implementar analise de credito simulada | concluido | E5-01, E4-02 | Worker/processo com retorno de status |
| E5-04 | Implementar analise de fraude simulada | concluido | E5-01, E4-02 | Worker/processo com retorno de status |
| E5-05 | Consolidar decisao da proposta | concluido | E5-02, E5-03, E5-04 | Atualizacao final ou pendencia da proposta |

### Epico 6 - Notificacao e acompanhamento

Objetivo: informar o cliente e registrar o historico do processo.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E6-01 | Implementar Notification Service basico | concluido | E5-05 | Envio inicial de notificacoes por mudanca de status |
| E6-02 | Registrar historico de comunicacoes | concluido | E6-01 | Trilha de tentativas, sucesso e falha |
| E6-03 | Expor timeline da proposta no front | concluido | E6-02, E3-06 | Visualizacao do andamento para o usuario |

### Epico 7 - Observabilidade, seguranca e operacao

Objetivo: endurecer a solucao para operacao real.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E7-01 | Adicionar logs estruturados | concluido | E4-01, E4-02 | Logs consistentes com correlation-id |
| E7-02 | Adicionar metricas e tracing | concluido | E7-01 | Visibilidade ponta a ponta |
| E7-03 | Implementar mascaramento de dados sensiveis | concluido | E2-05 | Protecao de logs e payloads |
| E7-04 | Configurar secrets e criptografia | concluido | E1-04 | Segregacao segura de credenciais e chaves |
| E7-05 | Definir alarmes e dashboards minimos | concluido | E7-02 | Monitoracao operacional basica |

### Epico 8 - Qualidade e entrega

Objetivo: garantir repetibilidade, confianca e evolucao continua.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E8-01 | Configurar lint e testes automatizados | concluido | E1-03 | Validacao automatica no repositorio |
| E8-02 | Configurar pipeline CI/CD inicial | concluido | E8-01 | Build, testes e empacotamento automatizados |
| E8-03 | Definir estrategia de deploy | concluido | E8-02 | Fluxo de entrega em dev e homologacao |
| E8-04 | Executar smoke tests do MVP | concluido | E3-06, E6-03, E8-02 | Validacao minima ponta a ponta |

## Ordem recomendada de execucao

1. Epico 1 - Fundacao do projeto
2. Epico 2 - Contratos e dominio
3. Epico 4 - Backend core
4. Epico 3 - Front web inicial
5. Epico 5 - Workflow e analises assincronas
6. Epico 6 - Notificacao e acompanhamento
7. Epico 7 - Observabilidade, seguranca e operacao
8. Epico 8 - Qualidade e entrega

## Proposta de ciclos

### Ciclo 1

- E1-01 Definir escopo fechado do MVP
- E1-02 Definir estrutura inicial do repositorio
- E2-01 Modelar entidades principais
- E2-02 Definir maquina de estados da proposta

### Ciclo 2

- E2-03 Definir contratos de API
- E2-04 Definir eventos assincronos
- E4-01 Criar BFF inicial
- E4-02 Implementar Proposal Service

### Ciclo 3

- E4-03 Implementar Customer Service
- E4-04 Implementar Document Service
- E3-01 Criar aplicacao web base
- E3-03 Criar jornada de abertura de proposta

### Ciclo 4

- E3-04 Criar jornada de cadastro do cliente
- E3-05 Criar jornada de upload de documentos
- E3-06 Criar consulta de status da proposta
- E5-01 Desenhar workflow inicial da proposta

### Ciclo 5

- E5-02 Implementar analise documental simulada
- E5-03 Implementar analise de credito simulada
- E5-04 Implementar analise de fraude simulada
- E5-05 Consolidar decisao da proposta

### Ciclo 6

- E6-01 Implementar Notification Service basico
- E6-02 Registrar historico de comunicacoes
- E7-01 Adicionar logs estruturados
- E8-01 Configurar lint e testes automatizados

## Registro de ciclos

### Ciclo atual

- Identificador: ciclo-17
- Status: aberto
- Objetivo: endurecer governanca operacional dos callbacks e sua rastreabilidade
- Itens priorizados:
  - preparar dashboards e operacao de fila para cenarios de erro mais agressivos
  - avaliar rate limit e politicas de replay por parceiro e por rota
  - adicionar enriquecimento e correlacao cruzada entre auditoria de callback e timeline da proposta
  - expor expiracao e limpeza operacional dos registros de auditoria
- Observacao:
  - ciclo-16 fechou auditoria persistida dos callbacks e replay-release manual no BFF

## Entregas materializadas no ciclo-16

- `BFF` com auditoria persistida dos callbacks em Redis quando configurado e fallback em memoria para testes;
- endpoints internos `GET /internal/webhooks/audit` e `POST /internal/webhooks/audit/{eventId}/replay-release`;
- preservacao do historico de liberacao manual mesmo apos novo processamento do mesmo evento;
- novas configuracoes de prefixo e retencao para a trilha de auditoria de callbacks;
- backlog reaberto para `ciclo-17` com foco em governanca e correlacao operacional.

## Entregas materializadas no ciclo-15

- `BFF` com deduplicacao compartilhada de webhooks via Redis quando `BFF_REDIS_URL` estiver configurado;
- fallback em memoria preservado para testes e execucao simplificada;
- `/metrics` do `BFF` ampliado com contadores por tipo de callback, parceiro e replay status;
- stack Docker e variaveis de ambiente atualizadas para a nova persistencia de replay;
- backlog reaberto para `ciclo-16` com foco em auditoria e operacao distribuida.

## Entregas materializadas no ciclo-14

- `BFF` com webhooks de parceiros em `/api/v1/webhooks/partners/credit-analysis` e `/api/v1/webhooks/partners/fraud-analysis`;
- protecao anti-replay no `BFF` com `X-Webhook-Event-Id`, `X-Webhook-Timestamp` e janela configuravel por `BFF_WEBHOOK_MAX_AGE_SECONDS`;
- `workflow service` com callbacks internos para aplicar resultados externos de credito e fraude;
- `workflow service` capaz de pausar em `credit_analysis_in_progress` ou `fraud_analysis_in_progress` quando os flags externos estiverem ativos;
- contrato OpenAPI, documentacao operacional e variaveis de ambiente atualizados para o novo fluxo.

## Entregas materializadas no ciclo-13

- `BFF` com validacao HMAC de `X-Webhook-Signature` para o webhook de storage;
- `workflow service` com endpoints internos para listar DLQ e reprocessar jobs manualmente;
- `workflow service` exposto localmente em `localhost:18084` para inspecao operacional;
- novo `mock-oidc service` com `authorize`, `token`, `userinfo`, `jwks` e `discovery`;
- stack Docker local configurada por padrao para login OIDC real contra o issuer mockado;
- smoke da stack Docker endurecido para validar `bff`, `workflow` e `mock-oidc`.

## Entregas materializadas no ciclo-3

- `Proposal Service` em Go com criacao, consulta e atualizacao de status;
- `Customer Service` em Go com cadastro, consulta por proposta e validacoes basicas;
- `Document Service` em Go com geracao de upload URL, listagem e confirmacao de recebimento;
- persistencia real em PostgreSQL para propostas;
- persistencia real em PostgreSQL para clientes;
- persistencia real em PostgreSQL para documentos;
- ambiente local minimo com `docker-compose` para banco;
- testes HTTP basicos do `Proposal Service`;
- testes HTTP basicos do `Customer Service`;
- testes HTTP basicos do `Document Service`;
- validacao de compilacao com `go test ./...` no modulo de proposta.
- validacao de compilacao com `go test ./...` no modulo de cliente.
- validacao de compilacao com `go test ./...` no modulo de documentos.

## Entregas materializadas no ciclo-4

- `BFF` integrado aos servicos reais de proposta, cliente e documento;
- `BFF` com agregacao da proposta consolidada e suporte a CORS para o front;
- app web em Next.js com jornada inicial de criacao de proposta, cadastro de cliente, upload e consulta de status;
- validacao do `BFF` com `go test ./...`;
- validacao do app web com `npm run typecheck` e `npm run build`.

## Entregas materializadas no ciclo-5

- `workflow service` para orquestracao da analise documental, de credito e de fraude;
- `credit-analysis service` e `fraud-analysis service` com regras simuladas e deterministicas;
- analise documental simulada adicionada ao `document service`;
- persistencia de `analysis_results` no `proposal service`;
- `BFF` disparando o workflow apos confirmacao do documento;
- front exibindo os resultados das analises na proposta consolidada;
- documentacao do fluxo em `docs/workflow_inicial.md`;
- validacao com `go test ./...` nos modulos `proposal`, `document`, `credit-analysis`, `fraud-analysis`, `workflow` e `bff`.

## Entregas materializadas no ciclo-6

- `notification service` com persistencia em PostgreSQL e entrega simulada;
- historico de status persistido no `proposal service`;
- `BFF` agregando status, analises, historico e notificacoes na proposta consolidada;
- timeline da proposta exibida no front com eventos de status e comunicacoes;
- validacao automatizada basica com `scripts/verify.ps1`;
- workflow GitHub Actions em `.github/workflows/validate.yml`.

## Entregas materializadas no ciclo-7

- logs estruturados em JSON nos servicos centrais;
- endpoint `/metrics` em `bff`, `proposal`, `workflow` e `notification`;
- `Dockerfile` para servicos e front;
- pipeline de build de imagens em `.github/workflows/build-images.yml`;
- smoke test automatizado do MVP em `scripts/smoke_mvp.ps1`;
- execucao bem-sucedida do smoke test local contra o fluxo fim a fim.

## Entregas materializadas no ciclo-8

- `BFF` mascarando `cpf`, `email`, `phone` e `recipient` nas respostas externas;
- `notification service` com destinatario mascarado para leitura e copia criptografada em repouso;
- suporte a `*_FILE` para secrets de banco nos servicos `proposal`, `customer`, `document` e `notification`;
- documentacao operacional em `docs/seguranca_operacional.md`, `docs/alarmes_dashboards.md` e `docs/estrategia_deploy.md`;
- atualizacao do contrato do BFF para refletir mascaramento de campos sensiveis;
- smoke test ajustado para exercitar a chave de criptografia da notificacao.

## Entregas materializadas no ciclo-9

- autenticacao mock OIDC-ready no `apps/web` com rota protegida, login, logout e sessao em cookie;
- pagina `/login` com configuracao-base para futura troca para issuer OIDC real;
- separacao da jornada principal em componente autenticado no front;
- ampliacao do `docker-compose` com `redis`, `mailpit` e `minio`;
- documentacao do ambiente local em `docs/ambiente_local.md`.

## Entregas materializadas no ciclo-10

- `document service` com upload real em MinIO e persistencia de `storage_url`;
- `BFF` com endpoint multipart para envio do arquivo e disparo automatico do workflow;
- front web atualizado para selecionar arquivo, registrar documento e enviar conteudo real;
- `notification service` com entrega SMTP local via Mailpit e status `sent` ou `failed`;
- callback inicial OIDC no front com inicio de login, validacao de `state` e criacao de sessao local;
- `docker-compose` e `.env.example` atualizados para MinIO e SMTP locais;
- contrato do BFF e documentacao do workflow e ambiente local atualizados;
- validacao com `go test ./...`, `npm run typecheck`, `npm run build` e `scripts/verify.ps1`.

## Entregas materializadas no ciclo-11

- `workflow service` com enfileiramento assincrono, worker dedicado e retry configuravel;
- suporte a Redis no workflow por `WORKFLOW_REDIS_URL`, com fallback para fila em memoria;
- callback OIDC no front com troca real de token e leitura de perfil via `userinfo` ou `id_token`;
- novas variaveis de ambiente para token endpoint, userinfo e segredo do cliente OIDC;
- `docker-compose` ligado ao Redis para o workflow e smoke dedicado da stack em `scripts/smoke_docker_stack.ps1`;
- documentacao atualizada para fluxo enfileirado, ambiente local e validacao ponta a ponta;
- validacao com `go test ./...` no workflow, `npm run typecheck`, `npm run build` e smoke da stack Docker.

## Entregas materializadas no ciclo-12

- validacao criptografica de `id_token` no front usando discovery/JWKS e biblioteca `jose`;
- novas variaveis de ambiente para discovery e JWKS do provedor OIDC;
- `workflow service` com metricas de fila, retry, profundidade e DLQ em `/metrics`;
- DLQ no workflow com suporte em Redis e fallback local em memoria;
- webhook inicial no `BFF` para callback externo de upload concluido em `/api/v1/webhooks/storage/document-uploaded`;
- contrato OpenAPI e documentacao do fluxo atualizados para o webhook e para o comportamento da DLQ;
- validacao com `go test ./...` no `workflow` e `bff`, `npm run typecheck` e `npm run build`.

### Historico

| Ciclo | Status | Resumo |
| --- | --- | --- |
| ciclo-0 | concluido | Backlog inicial criado a partir do plano macro |
| ciclo-1 | concluido | Escopo do MVP, estrutura inicial do repositorio, padroes de desenvolvimento, entidades e maquina de estados definidos |
| ciclo-2 | concluido | Contratos de API, eventos assincronos, padroes transversais e BFF inicial definidos e validados |
| ciclo-3 | concluido | Backend core inicial entregue com Proposal Service, Customer Service, Document Service e persistencia em PostgreSQL |
| ciclo-4 | concluido | BFF integrado ao backend core e front web base entregue com a jornada principal do MVP |
| ciclo-5 | concluido | Workflow inicial entregue com analises simuladas, persistencia dos resultados e consolidacao automatica da proposta |
| ciclo-6 | concluido | Notification Service, historico da proposta, timeline no front e validacao automatizada basica entregues |
| ciclo-7 | concluido | Observabilidade inicial, build de imagens e smoke test do MVP entregues |
| ciclo-8 | concluido | Mascaramento de PII no BFF, secrets por arquivo, criptografia de destinatario em notificacoes e documentacao operacional entregues |
| ciclo-9 | concluido | Autenticacao inicial mock OIDC-ready no front e ambiente local ampliado com Redis, Mailpit e MinIO |
| ciclo-10 | concluido | Upload real via MinIO, notificacoes SMTP locais via Mailpit e callback OIDC inicial no front |
| ciclo-11 | concluido | Workflow com fila e retry, callback OIDC com token exchange e smoke da stack Docker |
| ciclo-12 | concluido | Validacao criptografica de id_token, DLQ e metricas da fila no workflow e webhook inicial de storage no BFF |
| ciclo-13 | concluido | Webhook autenticado no BFF, inspecao e reprocessamento de DLQ e issuer OIDC mockado na stack local |
| ciclo-14 | concluido | Callbacks externos de credito e fraude, pausa opcional do workflow e anti-replay no BFF |
| ciclo-15 | concluido | Deduplicacao compartilhada em Redis no BFF e metricas de callbacks por parceiro |
| ciclo-16 | concluido | Auditoria persistida dos callbacks e replay-release manual no BFF |
