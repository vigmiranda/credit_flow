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
| E1-04 | Preparar ambiente local minimo | pendente | E1-02 | Stack local com banco, cache e servicos auxiliares |

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
| E3-01 | Criar aplicacao web base | pendente | E1-02 | App Next.js com estrutura inicial |
| E3-02 | Implementar autenticacao inicial | pendente | E3-01 | Fluxo mockado ou OIDC-ready para evolucao |
| E3-03 | Criar jornada de abertura de proposta | pendente | E3-01, E2-03 | Tela e fluxo para iniciar proposta |
| E3-04 | Criar jornada de cadastro do cliente | pendente | E3-03, E2-03 | Formularios com validacao |
| E3-05 | Criar jornada de upload de documentos | pendente | E3-03, E2-03 | Upload via URL assinada |
| E3-06 | Criar consulta de status da proposta | pendente | E3-03, E2-03 | Tela com status, pendencias e decisao |

### Epico 4 - Backend core

Objetivo: entregar o slice funcional principal da proposta.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E4-01 | Criar BFF inicial | concluido | E2-03 | API unica para o front |
| E4-02 | Implementar Proposal Service | concluido | E2-01, E2-02 | Criacao, consulta e atualizacao da proposta |
| E4-03 | Implementar Customer Service | concluido | E2-01, E2-03 | Cadastro e validacoes basicas de cliente |
| E4-04 | Implementar Document Service | pendente | E2-01, E2-04 | URL assinada, metadados e evento de upload |
| E4-05 | Persistencia inicial | em_andamento | E4-02, E4-03, E4-04 | Migrations e armazenamento transacional |

### Epico 5 - Workflow e analises assincronas

Objetivo: suportar o processamento desacoplado do fluxo de decisao.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E5-01 | Desenhar workflow inicial da proposta | pendente | E2-02, E2-04 | Fluxo orquestrado do recebimento ate decisao |
| E5-02 | Implementar analise documental simulada | pendente | E5-01, E4-04 | Resultado assincrono inicial para destravar fluxo |
| E5-03 | Implementar analise de credito simulada | pendente | E5-01, E4-02 | Worker/processo com retorno de status |
| E5-04 | Implementar analise de fraude simulada | pendente | E5-01, E4-02 | Worker/processo com retorno de status |
| E5-05 | Consolidar decisao da proposta | pendente | E5-02, E5-03, E5-04 | Atualizacao final ou pendencia da proposta |

### Epico 6 - Notificacao e acompanhamento

Objetivo: informar o cliente e registrar o historico do processo.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E6-01 | Implementar Notification Service basico | pendente | E5-05 | Envio inicial de notificacoes por mudanca de status |
| E6-02 | Registrar historico de comunicacoes | pendente | E6-01 | Trilha de tentativas, sucesso e falha |
| E6-03 | Expor timeline da proposta no front | pendente | E6-02, E3-06 | Visualizacao do andamento para o usuario |

### Epico 7 - Observabilidade, seguranca e operacao

Objetivo: endurecer a solucao para operacao real.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E7-01 | Adicionar logs estruturados | pendente | E4-01, E4-02 | Logs consistentes com correlation-id |
| E7-02 | Adicionar metricas e tracing | pendente | E7-01 | Visibilidade ponta a ponta |
| E7-03 | Implementar mascaramento de dados sensiveis | pendente | E2-05 | Protecao de logs e payloads |
| E7-04 | Configurar secrets e criptografia | pendente | E1-04 | Segregacao segura de credenciais e chaves |
| E7-05 | Definir alarmes e dashboards minimos | pendente | E7-02 | Monitoracao operacional basica |

### Epico 8 - Qualidade e entrega

Objetivo: garantir repetibilidade, confianca e evolucao continua.

| ID | Item | Status | Dependencias | Saida esperada |
| --- | --- | --- | --- | --- |
| E8-01 | Configurar lint e testes automatizados | pendente | E1-03 | Validacao automatica no repositorio |
| E8-02 | Configurar pipeline CI/CD inicial | pendente | E8-01 | Build, testes e empacotamento automatizados |
| E8-03 | Definir estrategia de deploy | pendente | E8-02 | Fluxo de entrega em dev e homologacao |
| E8-04 | Executar smoke tests do MVP | pendente | E3-06, E6-03, E8-02 | Validacao minima ponta a ponta |

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

- Identificador: ciclo-3
- Status: em_andamento
- Objetivo: iniciar os servicos core e a persistencia do fluxo de proposta
- Itens priorizados:
  - E4-03 Implementar Customer Service
  - E4-04 Implementar Document Service
  - E4-05 Persistencia inicial
- Observacao:
  - os servicos devem nascer aderentes aos contratos e padroes definidos no ciclo-2
  - a persistencia inicial ja foi aberta com PostgreSQL local e implementada para Proposal Service

## Entregas materializadas no ciclo-3

- `Proposal Service` em Go com criacao, consulta e atualizacao de status;
- `Customer Service` em Go com cadastro, consulta por proposta e validacoes basicas;
- persistencia real em PostgreSQL para propostas;
- persistencia real em PostgreSQL para clientes;
- ambiente local minimo com `docker-compose` para banco;
- testes HTTP basicos do `Proposal Service`;
- testes HTTP basicos do `Customer Service`;
- validacao de compilacao com `go test ./...` no modulo de proposta.
- validacao de compilacao com `go test ./...` no modulo de cliente.

### Historico

| Ciclo | Status | Resumo |
| --- | --- | --- |
| ciclo-0 | concluido | Backlog inicial criado a partir do plano macro |
| ciclo-1 | concluido | Escopo do MVP, estrutura inicial do repositorio, padroes de desenvolvimento, entidades e maquina de estados definidos |
| ciclo-2 | concluido | Contratos de API, eventos assincronos, padroes transversais e BFF inicial definidos e validados |
| ciclo-3 | em_andamento | Proposal Service e persistencia inicial abertos, com foco restante em Customer Service e Document Service |
