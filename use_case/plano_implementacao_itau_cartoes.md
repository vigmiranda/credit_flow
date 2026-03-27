# Plano de Implementação — Plataforma de Captura e Análise de Propostas de Cartões

## 1. Contexto

Este documento apresenta um plano completo de implementação para uma plataforma de **captura e análise de propostas de novos clientes para cartões**, contemplando:

- cadastro de clientes, conta e cartão;
- análise de documentos;
- análise de crédito;
- análise de fraude;
- notificação do cliente sobre os passos da proposta.

O case sugere explicitamente o uso de **microserviços, cloud, OAuth e princípios de 12 fatores**. fileciteturn1file0L1-L12

A proposta abaixo foi pensada com uma visão de arquitetura especializada em **Go**, priorizando:

- **escalabilidade**;
- **modularidade**;
- **simplicidade operacional (KISS)**;
- **segurança**;
- **valor real ao usuário e ao negócio**.

---

## 2. Objetivos da solução

A arquitetura não deve ser guiada apenas por tecnologia, mas pelo problema de negócio. O principal valor da solução é permitir que o cliente:

- envie sua proposta com baixa fricção;
- acompanhe o status com clareza;
- tenha respostas rápidas nas etapas críticas;
- não enfrente indisponibilidades causadas por integrações externas;
- receba uma experiência fluida, segura e confiável.

Do ponto de vista do negócio, a solução deve:

- aumentar conversão no onboarding;
- reduzir abandono da jornada;
- suportar picos de volume com estabilidade;
- reduzir fraude sem degradar a experiência do cliente;
- permitir evolução contínua de regras de crédito, fraude e comunicação.

---

## 3. Princípios arquiteturais

### 3.1 KISS — Keep It Simple
A solução deve começar simples, com poucos serviços bem definidos e limites claros de domínio, evitando complexidade desnecessária no início.

### 3.2 Modularidade orientada a domínio
Os principais domínios de negócio devem ser isolados para facilitar evolução, escalabilidade independente e manutenção.

### 3.3 Escalabilidade seletiva
Nem todos os componentes precisam escalar igualmente. Serviços com carga variável, integrações externas ou processamento pesado devem escalar de forma independente.

### 3.4 Event-driven onde faz sentido
Etapas longas ou sujeitas a falhas, como crédito, fraude, documentos e notificações, devem ser assíncronas.

### 3.5 Segurança by design
Dados sensíveis, autenticação, trilhas de auditoria, criptografia, segregação de acesso e aderência a LGPD devem fazer parte do desenho desde o início.

### 3.6 12 fatores
Os serviços devem seguir boas práticas cloud-native:

- configuração externa ao código;
- serviços stateless;
- logs como fluxo de eventos;
- deploys reprodutíveis;
- isolamento entre ambientes;
- escalabilidade horizontal.

---

## 4. Visão funcional da plataforma

A plataforma terá três camadas principais:

1. **Front web** para jornada operacional e/ou captura digital;
2. **Backend** orientado a domínios e APIs;
3. **Infraestrutura cloud na AWS** para execução, segurança, mensageria, persistência e observabilidade.

---

## 5. Arquitetura proposta — visão macro

```text
[ Front Web / Portal ]
        |
        v
[ CloudFront + WAF ]
        |
        v
[ API Gateway / BFF ]
        |
        +------------------------------+
        |                              |
        v                              v
[ Proposal Service ]            [ Customer Service ]
        |                              |
        +---------------+--------------+
                        |
                        v
                 [ PostgreSQL ]
                        |
                        v
                [ Step Functions ]
                        |
        +---------------+----------------+----------------+
        |                                |                |
        v                                v                v
[ Document Service ]            [ Credit Service ] [ Fraud Service ]
        |                                |                |
        v                                v                v
      [ S3 ]                   [ Integrações ]     [ Integrações ]
                        
                        v
               [ Notification Service ]
                        |
                        v
              [ E-mail / SMS / Push ]

Suporte transversal:
- Cognito / OIDC
- SNS / SQS / DLQ
- Redis
- CloudWatch / OpenTelemetry / X-Ray / Grafana
- Secrets Manager / KMS / IAM
```

---

## 6. Front web

### 6.1 Objetivo do front
O front web será a camada de interação com o usuário final e/ou usuários operacionais, cobrindo:

- abertura da proposta;
- preenchimento de dados cadastrais;
- envio de documentos;
- acompanhamento do status;
- exibição de pendências;
- retorno da decisão da proposta.

### 6.2 Stack recomendada

- **React**
- **TypeScript**
- **Next.js**
- **Tailwind CSS**
- **React Query / TanStack Query**
- **Zod** para validação
- **OpenTelemetry Web**
- **Jest + Testing Library + Playwright**

### 6.3 Por que essas escolhas

#### React
Escolha madura para aplicações web corporativas, com amplo ecossistema, componentização forte e facilidade para times grandes.

#### TypeScript
Traz segurança de tipos, reduz erros em tempo de desenvolvimento e melhora a manutenção em fluxos de cadastro complexos.

#### Next.js
Permite excelente estruturação do projeto, otimização de build, rotas organizadas e flexibilidade para páginas públicas, autenticadas e renderização híbrida quando necessário.

#### Tailwind CSS
Acelera a construção da interface, mantém consistência visual e reduz custo de manutenção de CSS em projetos grandes.

#### React Query
Ideal para controle de cache, sincronização com backend, estados de loading, retry e invalidação inteligente.

#### Zod
Facilita validação consistente de formulários e schemas, reduzindo inconsistências entre front e backend.

#### Playwright
Essencial para validar jornadas críticas ponta a ponta, especialmente onboarding, upload de documentos e acompanhamento de status.

### 6.4 Estrutura sugerida do front

```text
/apps/web
  /src
    /app
    /components
    /features
      /proposal
      /customer
      /documents
      /status
    /services
    /hooks
    /lib
    /styles
```

### 6.5 Responsabilidades do front

- autenticar usuário via OAuth/OIDC;
- iniciar nova proposta;
- preencher dados e validar entrada;
- solicitar upload de documentos com URL assinada;
- consultar status da proposta;
- exibir mensagens claras de pendência, aprovação ou reprovação;
- registrar telemetria da jornada.

---

## 7. Backend

## 7.1 Visão geral
O backend será composto por serviços em **Go**, desenhados como serviços stateless, orientados a domínio, com separação clara entre fluxos síncronos e assíncronos.

### Serviços principais

- **BFF / API Composition**
- **Proposal Service**
- **Customer Service**
- **Document Service**
- **Credit Analysis Service**
- **Fraud Analysis Service**
- **Notification Service**
- **Workflow / Orchestration** com Step Functions

### 7.2 Linguagem e padrões

- **Go** como linguagem principal;
- **REST** para APIs síncronas;
- **Eventos via SNS/SQS** para integração interna;
- **Clean Architecture / Hexagonal leve**;
- **OpenAPI** para contratos;
- **Idempotência e rastreabilidade por correlation-id**.

### 7.3 Por que Go

- excelente desempenho com baixo consumo de recursos;
- ótima concorrência para I/O intensivo;
- boa aderência a containers e ambientes cloud;
- simplicidade de build e deploy;
- manutenção previsível em times de backend.

---

## 8. Serviços de backend e responsabilidades

### 8.1 BFF / API Composition

#### Responsabilidades
- servir como ponto de entrada para o front;
- consolidar respostas de múltiplos serviços;
- esconder complexidade do backend da interface;
- aplicar políticas de autenticação, autorização e rate limiting;
- otimizar payloads para a experiência do front.

#### Tecnologia
- Go
- API Gateway na borda
- possível uso de gRPC interno no futuro, se necessário

#### Por que essa escolha
Evita acoplamento excessivo do front com vários domínios internos e melhora a evolução da experiência digital sem expor detalhes da arquitetura.

---

### 8.2 Proposal Service

#### Responsabilidades
- criar e atualizar propostas;
- controlar o ciclo de vida da proposta;
- persistir status e metadados de processo;
- disponibilizar consulta consolidada do andamento.

#### Tecnologia
- Go
- PostgreSQL
- publicação de eventos para SNS

#### Por que essa escolha
A proposta é o núcleo do processo. Um serviço dedicado facilita auditoria, rastreabilidade e evolução do fluxo principal.

---

### 8.3 Customer Service

#### Responsabilidades
- cadastro de dados do cliente;
- validações cadastrais;
- normalização de informações;
- relacionamento cliente-conta-cartão.

#### Tecnologia
- Go
- PostgreSQL

#### Por que essa escolha
O domínio de cliente tende a evoluir de forma independente e possui regras próprias de validação, o que justifica separação lógica clara.

---

### 8.4 Document Service

#### Responsabilidades
- gerar URLs assinadas para upload;
- persistir metadados dos documentos;
- versionar arquivos;
- publicar eventos após recebimento;
- apoiar integração com análise documental.

#### Tecnologia
- Go
- Amazon S3
- PostgreSQL para metadados
- SNS/SQS para evento de documento recebido

#### Por que essa escolha
Documentos têm grande volume binário e não devem ficar em banco relacional. O S3 é a escolha natural para armazenamento seguro, escalável e econômico.

---

### 8.5 Credit Analysis Service

#### Responsabilidades
- consultar bureaus e provedores externos;
- enriquecer dados de crédito;
- executar score e parecer;
- devolver resultado para o fluxo de decisão.

#### Tecnologia
- Go
- SQS para processamento assíncrono
- integração com APIs externas
- Redis para cache pontual e controle de idempotência

#### Por que essa escolha
Crédito envolve latência externa e risco de indisponibilidade. O processamento assíncrono aumenta resiliência e evita travar a experiência do usuário.

---

### 8.6 Fraud Analysis Service

#### Responsabilidades
- validar inconsistências cadastrais;
- integrar com motores antifraude;
- executar regras e score de risco;
- retornar decisão de fraude para o fluxo principal.

#### Tecnologia
- Go
- SQS
- integrações externas
- PostgreSQL/Redis conforme necessidade de persistência temporária

#### Por que essa escolha
Fraude é um domínio que muda rapidamente, integra terceiros e exige escalabilidade própria. Ter um serviço isolado reduz acoplamento e permite evolução contínua.

---

### 8.7 Notification Service

#### Responsabilidades
- notificar o cliente sobre cada etapa da proposta;
- enviar e-mail, SMS, push e futuros canais;
- gerenciar templates;
- manter trilha de entrega e falha.

#### Tecnologia
- Go
- SNS/SQS
- Amazon SES para e-mail
- integração com provedor SMS/push

#### Por que essa escolha
Notificações não devem bloquear a jornada principal. O desacoplamento via eventos melhora confiabilidade e escalabilidade.

---

### 8.8 Workflow / Orquestração

#### Responsabilidades
- coordenar as etapas da proposta;
- controlar status e transições;
- consolidar os resultados de documentos, crédito e fraude;
- acionar decisão final e comunicação.

#### Tecnologia
- **AWS Step Functions**

#### Por que essa escolha
Step Functions oferece visibilidade do workflow, retries controlados, tratamento de falhas e clareza operacional sem necessidade de construir um orquestrador customizado.

---

## 9. Fluxo de negócio ponta a ponta

### 9.1 Jornada principal

1. Usuário acessa o front web e autentica.
2. Front chama o BFF/API Gateway.
3. Proposal Service cria proposta e retorna protocolo.
4. Customer Service registra dados do cliente.
5. Document Service gera URL assinada para upload no S3.
6. Após upload, evento de documento recebido é publicado.
7. Step Functions dispara:
   - análise documental;
   - análise de crédito;
   - análise de fraude.
8. Cada domínio processa de forma independente e publica o resultado.
9. O workflow consolida o resultado.
10. Notification Service comunica o cliente.
11. Front exibe status final ou pendências.

### 9.2 Status possíveis

- proposta criada;
- documentação pendente;
- documentação em análise;
- crédito em análise;
- fraude em análise;
- aprovada;
- reprovada;
- revisão manual;
- aguardando complementação.

---

## 10. Estratégia de integração

### 10.1 Síncrono
Usar para operações que exigem resposta imediata:

- login;
- criação de proposta;
- consulta de status;
- leitura de dados cadastrais;
- geração de upload URL.

### 10.2 Assíncrono
Usar para operações sujeitas a latência, falhas externas ou processamento pesado:

- análise documental;
- análise de crédito;
- análise de fraude;
- notificações;
- auditoria;
- tarefas de pós-processamento.

### 10.3 Tecnologias e justificativas

#### SNS
Permite fan-out de eventos para múltiplos consumidores sem acoplamento direto.

#### SQS
Oferece buffer, desacoplamento, controle de retry e escalabilidade por consumo.

#### DLQ
Garante tratamento seguro de mensagens com falha e evita perda silenciosa.

---

## 11. Persistência e dados

### 11.1 Banco transacional

#### Tecnologia
- **Amazon RDS PostgreSQL** ou **Aurora PostgreSQL**

#### Recomendação
- iniciar com **RDS PostgreSQL** se o objetivo for simplicidade e menor custo inicial;
- evoluir para **Aurora PostgreSQL** caso o volume e requisitos de disponibilidade/redução de failover justifiquem.

#### Por que PostgreSQL
- maturidade;
- consistência transacional;
- ótima aderência a domínios relacionais;
- suporte forte no mercado;
- facilidade de operação e observabilidade.

### 11.2 Object storage

#### Tecnologia
- **Amazon S3**

#### Uso
- documentos enviados pelo cliente;
- arquivos de evidência;
- exportações e anexos de suporte.

#### Por que S3
- armazenamento escalável e durável;
- integração nativa com eventos e segurança AWS;
- excelente custo-benefício para conteúdo binário.

### 11.3 Cache e idempotência

#### Tecnologia
- **Amazon ElastiCache for Redis**

#### Uso
- cache de consultas frequentes;
- deduplicação;
- controle de idempotência;
- dados temporários de processamento.

#### Por que Redis
Baixa latência, simplicidade de uso e eficiência para cenários transitórios.

---

## 12. Infraestrutura AWS

### 12.1 Camada de entrega

#### CloudFront
Distribuição do front web com baixa latência e integração simples com WAF.

#### WAF
Proteção contra ataques comuns na camada web, como bots, exploração automática e tráfego malicioso.

#### S3 para assets estáticos do front
Hospedagem estável, barata e simples para artefatos do front quando aplicável.

---

### 12.2 Entrada de APIs

#### Amazon API Gateway
Responsável por expor APIs externas com segurança, controle de autenticação, throttling, versionamento e observabilidade.

#### Por que API Gateway
Reduz esforço operacional na borda, centraliza políticas e simplifica governança.

---

### 12.3 Execução de serviços

#### Opção recomendada
- **Amazon EKS** para os serviços Go

#### Por que EKS
- padronização enterprise;
- ótima aderência a microsserviços em container;
- escalabilidade horizontal;
- controle avançado de rede, segurança e observabilidade;
- flexibilidade para workloads heterogêneos.

#### Alternativa viável
- **ECS Fargate** se o foco for reduzir complexidade operacional de Kubernetes.

#### Posição recomendada para este case
Apresentar **EKS como principal** e citar **ECS Fargate como alternativa mais simples** demonstra maturidade de trade-off.

---

### 12.4 Orquestração de processos

#### AWS Step Functions
Responsável pelo fluxo da proposta.

#### Por que Step Functions
- observabilidade do fluxo;
- retries e timeout declarativos;
- baixo esforço de manutenção;
- clareza na explicação para banca técnica.

---

### 12.5 Segurança

#### Amazon Cognito ou IdP corporativo federado via OIDC/OAuth2
Autenticação e autorização dos usuários.

#### AWS Secrets Manager
Armazenamento seguro de segredos e credenciais.

#### AWS KMS
Gestão de chaves de criptografia.

#### IAM
Controle fino de permissões entre componentes.

#### Por que essas escolhas
São serviços nativos, maduros, auditáveis e aderentes a requisitos corporativos e regulatórios.

---

### 12.6 Observabilidade

#### CloudWatch
Logs, métricas e alarmes básicos da plataforma.

#### X-Ray / OpenTelemetry
Tracing distribuído para acompanhamento da jornada ponta a ponta.

#### Prometheus + Grafana
Métricas técnicas e dashboards mais ricos em ambiente Kubernetes.

#### Por que essa composição
Une monitoração nativa AWS com padrões modernos de observabilidade distribuída.

---

## 13. Segurança, conformidade e LGPD

### Requisitos essenciais

- autenticação via OAuth2/OIDC;
- comunicação TLS ponta a ponta;
- criptografia em repouso com KMS;
- segredos fora do código;
- mascaramento de dados sensíveis em logs;
- trilhas de auditoria por proposta;
- segregação por ambiente;
- menor privilégio;
- retenção e descarte de dados por política.

### Boas práticas adicionais

- tokenização/máscara de CPF e dados sensíveis onde possível;
- correlation-id por requisição;
- auditoria de decisões automáticas;
- trilha de consentimento e acesso.

---

## 14. Escalabilidade e resiliência

### Estratégias principais

- serviços stateless;
- auto scaling horizontal;
- filas entre domínios críticos;
- retry com backoff exponencial;
- circuit breaker para integrações externas;
- timeout bem definido;
- idempotência em operações críticas;
- DLQ para mensagens inválidas ou falhadas.

### Valor prático dessas escolhas
Esses padrões reduzem impacto de indisponibilidades externas, evitam duplicidade de processamento e mantêm a plataforma funcional em cenários de pico.

---

## 15. Organização do código em Go

### Estrutura sugerida por serviço

```text
/cmd/api
/internal
  /domain
  /application
  /adapters
    /http
    /repository
    /events
  /platform
/pkg
```

### Padrões recomendados

- handlers finos;
- casos de uso explícitos;
- domínio isolado;
- interfaces nas bordas;
- observabilidade embutida desde o início;
- configuração por variáveis de ambiente.

### Por que essa abordagem
Mantém clareza arquitetural sem exagerar na complexidade, favorecendo testes, evolução e reuso.

---

## 16. Estratégia de APIs

### Externa
- REST + JSON
- OpenAPI
- versionamento de contratos
- autenticação por JWT/OIDC

### Interna
- REST inicialmente
- eventos para workflows e integrações internas
- evolução futura para gRPC apenas se houver ganho real

### Por que essa abordagem
REST é suficiente, simples e compatível com a maioria dos cenários do case. Introduzir gRPC cedo demais agregaria complexidade sem necessidade comprovada.

---

## 17. DevOps, CI/CD e qualidade

### Ferramentas sugeridas

- **Terraform** para infraestrutura como código;
- **GitHub Actions / GitLab CI / Jenkins** para pipeline;
- **Docker** para empacotamento;
- **SonarQube** ou equivalente para qualidade;
- **Trivy / Snyk** para segurança de imagens e dependências.

### Pipeline recomendado

1. lint;
2. testes unitários;
3. testes de integração;
4. build;
5. análise de segurança;
6. build de imagem;
7. publicação em registry;
8. deploy em ambiente alvo;
9. smoke tests.

### Estratégias de deploy

- rolling update para serviços simples;
- blue/green ou canary para componentes críticos.

---

## 18. Estratégia de testes

### Front
- unitários;
- integração de componentes;
- E2E com Playwright.

### Backend
- unitários;
- contract tests;
- integração com banco e filas;
- testes com mocks de integrações externas.

### Plataforma
- testes de carga;
- testes de resiliência;
- validação de alarmes e observabilidade.

---

## 19. MVP e evolução

### 19.1 MVP
Entregar primeiro:

- autenticação;
- criação de proposta;
- cadastro do cliente;
- upload de documentos;
- análise assíncrona de crédito e fraude;
- consulta de status;
- notificações básicas.

### 19.2 Evoluções futuras

- revisão manual operacional;
- motor de regras configurável;
- recomendação personalizada de produto;
- analytics avançado da jornada;
- antifraude com modelos mais sofisticados;
- omnicanal de notificações.

---

## 20. Trade-offs assumidos

### Por que não criar dezenas de microserviços no início
Porque aumentaria complexidade operacional, custo de observabilidade, número de deploys e dificuldade de troubleshooting sem ganho proporcional.

### Por que não deixar tudo síncrono
Porque análises e integrações externas aumentariam latência, risco de timeout e fragilidade da jornada.

### Por que usar AWS Step Functions
Porque acelera a construção do workflow e torna a orquestração visível e auditável.

### Por que EKS em vez de Lambda como padrão
Porque a solução possui múltiplos serviços backend em Go, integração contínua, observabilidade compartilhada e necessidade de padronização enterprise. Lambda pode ser usada pontualmente, mas não como base principal.

---

## 21. Métricas de sucesso

### Negócio
- taxa de conversão de propostas;
- taxa de abandono por etapa;
- tempo médio até decisão;
- taxa de aprovação;
- taxa de fraude detectada;
- tempo de notificação ao cliente.

### Técnica
- latência p95 e p99;
- taxa de erro por serviço;
- profundidade das filas;
- tempo médio de processamento por etapa;
- disponibilidade por domínio;
- tempo de recuperação de falhas.

---

## 22. Conclusão executiva

A melhor abordagem para este case é uma arquitetura **cloud-native, modular e pragmática**, com forte uso de **Go no backend**, **React/Next.js no front** e **AWS como base de execução, segurança e observabilidade**.

A proposta equilibra:

- **simplicidade no início**;
- **separação clara de responsabilidades**;
- **escalabilidade independente dos domínios críticos**;
- **resiliência para integrações externas**;
- **segurança e governança adequadas ao contexto financeiro**.

A combinação de **front web moderno**, **serviços Go stateless**, **workflow orquestrado com Step Functions**, **eventos com SNS/SQS**, **dados transacionais em PostgreSQL**, **documentos em S3** e **observabilidade completa** cria uma solução robusta, evolutiva e aderente ao objetivo do case.

Em resumo, trata-se de uma arquitetura pensada para gerar valor real ao usuário final, sustentando crescimento com previsibilidade, clareza operacional e baixo acoplamento.
