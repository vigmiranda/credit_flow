# Roteiro de Apresentacao de 30 Minutos

## Objetivo da apresentacao

Explicar de forma rapida e clara:

- qual problema o projeto resolve;
- o que foi entregue no MVP;
- quais tecnologias usamos;
- por que escolhemos essa arquitetura;
- como o fluxo funciona ponta a ponta;
- quais foram os principais trade-offs;
- o que fica para `pos-mvp`.

## Mensagem central

Construimos um `slice vertical` demonstravel para propostas de cartao, com jornada ponta a ponta, arquitetura preparada para evoluir com integracoes externas, operacao local reproduzivel e controles minimos de seguranca e observabilidade.

---

## Agenda sugerida

```text
1. Contexto e objetivo do MVP .................. 3 min
2. O que foi entregue .......................... 4 min
3. Tecnologias usadas e por que ................ 5 min
4. Arquitetura e decisoes-chave ................ 8 min
5. Como isso evolui para AWS real .............. 4 min
6. Fluxo ponta a ponta ......................... 4 min
7. Operacao, seguranca e observabilidade ....... 3 min
8. Encerramento, limites e proximos passos ..... 2 min
```

---

## 1. Contexto e objetivo do MVP

### Fala sugerida

`O objetivo deste projeto nao foi construir uma plataforma completa de cartoes em producao, mas sim um MVP tecnicamente consistente, capaz de demonstrar a jornada ponta a ponta de uma proposta. A prioridade foi entregar um fluxo realista, com separacao de responsabilidades, processamento assincrono, upload real de documentos, autenticacao, notificacao e operacao local reproduzivel.`

### Pontos para enfatizar

- queriamos um fluxo demonstravel de negocio, nao apenas prototipos isolados;
- o foco foi validar a espinha dorsal da plataforma;
- a arquitetura ja foi pensada para crescer para parceiros reais depois.

---

## 2. O que foi entregue

### Fala sugerida

`No MVP, entregamos a criacao da proposta, o cadastro do cliente, o upload real de documento, as analises assincronas simuladas, a consolidacao de status, as notificacoes, a timeline da proposta e um conjunto minimo de operacao local.`

### Escopo entregue

- abertura de proposta;
- cadastro do cliente;
- registro e upload real de documento em storage S3-compatible;
- workflow assincrono com fila, retry e DLQ;
- analise de documento, credito e fraude simuladas;
- consolidacao da proposta no BFF;
- front autenticado com OIDC local;
- notificacao por SMTP local;
- auditoria de callbacks e operacao assistida;
- stack Docker unica para rodar tudo localmente.

### Mensagem importante

`O MVP nao e um mock de tela. Ele executa um fluxo real entre componentes diferentes.`

---

## 3. Tecnologias usadas e por que

## Frontend

- `Next.js`

### Por que usamos

- entrega rapida de uma interface moderna;
- bom suporte para rotas, middleware e autenticacao;
- facilita demonstracao e evolucao futura para um portal operacional.

## Backend

- `Go`

### Por que usamos

- baixo overhead para servicos HTTP;
- simplicidade de deploy e build;
- codigo enxuto para APIs e workers;
- boa adequacao para componentes independentes e concorrencia simples.

## Persistencia

- `PostgreSQL`

### Por que usamos

- modelo transacional confiavel;
- adequado para entidades de negocio como proposta, cliente, documento e notificacao;
- bom ponto de partida para um dominio financeiro.

## Infra de apoio

- `Redis`

### Por que usamos

- fila do workflow;
- deduplicacao de callbacks;
- rate limit de webhook;
- operacao simples para um MVP distribuido.

- `MinIO`

### Por que usamos

- storage S3-compatible local;
- permite exercitar upload real sem depender de nuvem externa;
- aproxima o MVP de um desenho produtivo.

- `Mailpit`

### Por que usamos

- SMTP local simples;
- permite validar notificacao ponta a ponta sem dependencias externas.

- `Mock OIDC`

### Por que usamos

- login realista com issuer local;
- prepara o front para troca futura por provedor corporativo real.

- `Docker Compose`

### Por que usamos

- reproducao local com um unico comando;
- reduz friccao de setup;
- facilita demo, smoke e handoff tecnico.

---

## 4. Como definimos a arquitetura

## Principio principal

Separar a jornada em blocos com responsabilidade clara, sem exagerar na complexidade.

## Arquitetura escolhida

Referencia visual:

- `docs/diagrama_arquitetura_mvp.md`

```text
Web
  -> BFF
     -> Proposal Service
     -> Customer Service
     -> Document Service
     -> Workflow Service
     -> Notification Service
     -> parceiros simulados de credito e fraude
```

## Por que usamos BFF

### Fala sugerida

`Escolhemos um BFF porque queriamos proteger o front da complexidade interna. Em vez de o frontend conhecer cada servico, ele fala com uma camada de agregacao que monta a proposta consolidada, aplica mascaramento e concentra regras de borda como callbacks e politicas operacionais.`

### Beneficios

- simplifica o front;
- centraliza agregacao;
- reduz acoplamento com servicos internos;
- concentra preocupacoes de borda.

## Por que quebramos por dominio

### Servicos separados

- `proposal`
- `customer`
- `document`
- `workflow`
- `notification`

### Motivo

- cada um representa um contexto funcional claro;
- facilita evolucao incremental;
- torna mais natural integrar parceiros externos depois;
- melhora a clareza do fluxo para apresentacao e manutencao.

## Por que o workflow e assincrono

### Fala sugerida

`As analises de documento, credito e fraude nao deveriam bloquear a experiencia inteira nem ficar presas a uma unica chamada sincrona. Por isso introduzimos um workflow com fila, worker, retry e DLQ. Mesmo no MVP, isso ja aproxima o comportamento do que seria esperado em integracoes reais.`

### Beneficios

- desacoplamento temporal;
- tolerancia a falhas;
- facilidade de retry;
- caminho natural para parceiros externos.

## Por que nao fazer tudo como monolito simples

### Resposta sugerida

`Daria para fazer, mas perderiamos exatamente o aprendizado que o MVP queria validar: integracao entre contextos, agregacao, callbacks, processamento assincrono e operacao distribuida. Ainda assim, evitamos complexidade desnecessaria, porque os servicos sao pequenos e a stack local sobe toda com Docker Compose.`

---

## 5. Fluxo ponta a ponta

## 5. Como isso evolui para AWS real

Referencia de apoio:

- `docs/arquitetura_aws_referencia.md`

### Fala sugerida

`O ponto importante e que nao desenhamos uma arquitetura local descartavel. O que fizemos no MVP pode ser mapeado quase diretamente para uma implementacao real na AWS, trocando componentes locais por servicos gerenciados equivalentes.`

## Mapeamento principal

- `MinIO` vira `Amazon S3`
- `Mailpit` vira `Amazon SES`
- `mock-oidc` vira `Amazon Cognito` ou federacao com IdP corporativo
- `PostgreSQL` vira `Aurora PostgreSQL` ou `RDS PostgreSQL`
- `Redis` deixa de ser fila principal e passa a ficar mais focado em dedupe e rate limit via `ElastiCache Redis`
- fila principal do workflow vai para `Amazon SQS` com `DLQ`
- `web`, `bff` e servicos Go sobem em `ECS Fargate`

## Mensagem-chave

`Nao muda a responsabilidade de cada bloco. Muda apenas a implementacao da infraestrutura para um ambiente real e gerenciado.`

## Como responder por que ECS Fargate

`Porque ja temos servicos containerizados, separados por dominio, e queremos manter consistencia com o desenho atual sem trazer a complexidade de Kubernetes logo de inicio.`

## Como responder por que SQS e nao Redis como fila principal

`Porque em AWS o SQS oferece uma separacao mais limpa da responsabilidade de mensageria, com retry e DLQ mais naturais para o contexto do workflow.`

---

## 6. Fluxo ponta a ponta

### Narrativa sugerida

`O operador acessa o front, faz login, cria uma proposta, salva os dados do cliente, registra um documento e envia o arquivo. O documento vai para o storage, o workflow inicia, processa analise documental, credito e fraude, persiste os resultados, atualiza o status da proposta e registra notificacoes. O BFF consolida tudo e devolve para o front uma visao unica da proposta, incluindo timeline e eventos operacionais.`

## Fluxo resumido

```text
1. usuario acessa o web
2. web autentica via mock-oidc
3. web cria proposta no BFF
4. BFF cria proposta no proposal service
5. usuario salva cliente
6. BFF envia cliente ao customer service
7. usuario registra documento
8. BFF coordena document service
9. arquivo vai para MinIO
10. workflow e disparado
11. workflow executa analises
12. resultados voltam para proposal service
13. notification service registra e envia email
14. BFF agrega tudo para o front
```

## Ponto forte para apresentar

`O sistema nao retorna apenas um status final. Ele mostra historico, analises, notificacoes e sinais operacionais.`

---

## 7. Seguranca, operacao e observabilidade

## Seguranca minima implementada

- mascaramento de `cpf`, `email`, `phone` e `recipient`;
- secrets por arquivo para componentes stateful;
- criptografia de destinatario no servico de notificacao;
- autenticacao no front com fluxo OIDC local;
- assinatura HMAC para webhooks;
- anti-replay para callbacks;
- allowlist de providers e rate limit por rota/parceiro.

## Operacao implementada

- stack completa local com Docker;
- smoke ponta a ponta;
- readiness automatizada;
- overview operacional consolidado;
- auditoria de callbacks;
- replay-release manual;
- cleanup da auditoria;
- DLQ e reprocessamento no workflow.

## Observabilidade implementada

- `/metrics` em servicos chave;
- logs estruturados com `correlation_id`;
- contadores de callback por tipo, parceiro e outcome;
- metricas de fila, retry e DLQ;
- thresholds documentados para operacao local.

### Fala sugerida

`Mesmo sendo um MVP, nao paramos no fluxo feliz. Colocamos controles minimos para explicar como isso seria operado e evoluido.`

---

## 8. Como apresentar a demo em 5 minutos

## Sequencia recomendada

1. mostrar o diagrama simples da arquitetura;
2. abrir o front em `http://localhost:3000`;
3. fazer login;
4. criar proposta;
5. salvar cliente;
6. registrar e subir documento;
7. atualizar a proposta ate o status final;
8. mostrar timeline;
9. abrir Mailpit;
10. rodar `ops_overview.ps1`.

## Frases curtas para usar

- `Aqui o BFF agrega o fluxo e simplifica o front.`
- `Aqui o workflow entra para tirar a analise do caminho sincrono.`
- `Aqui validamos que a proposta nao e apenas criada, ela e realmente processada.`
- `Aqui mostramos que o MVP ja tem operacao local e rastreabilidade.`

---

## 9. Trade-offs assumidos

## O que simplificamos

- analises de credito e fraude ainda simuladas;
- thresholds e alertas ainda locais/documentais;
- parceiros externos ainda mockados;
- desenho otimizado para demonstração e readiness tecnica, nao para producao regulada.

## O que preservamos

- separacao de dominios;
- fluxo assincrono real;
- storage real;
- autenticacao realista local;
- rastreabilidade e operacao minima.

### Mensagem importante

`O MVP simplifica integrações, mas nao simplifica a espinha dorsal da arquitetura.`

---

## 10. Perguntas que provavelmente vao surgir

## Por que Go?

`Porque entrega servicos simples, performaticos e faceis de empacotar, com pouco overhead para APIs e workers.`

## Por que nao um monolito?

`Porque queriamos validar contextos separados, callbacks, agregacao e processamento assincrono, que sao centrais neste problema.`

## Por que Redis?

`Porque ele resolve bem fila leve, dedupe e rate limit no contexto do MVP.`

## Por que BFF?

`Porque o front precisa de uma visao consolidada da proposta sem conhecer a topologia interna.`

## O que falta para producao?

`Integracoes reais, observabilidade externa, endurecimento regulatorio, testes de carga, ambiente dedicado e operacao 24x7.`

## Como isso ficaria em AWS?

`Mantendo os mesmos dominios e a mesma separacao logica, mas substituindo os componentes locais por ECS Fargate, Aurora, S3, SQS, Cognito, SES, CloudWatch e ElastiCache Redis.`

---

## 11. Fechamento da apresentacao

### Fala sugerida

`Em resumo, o projeto entrega um MVP tecnicamente consistente, com jornada ponta a ponta, arquitetura organizada por dominios, processamento assincrono, operacao local simples e base preparada para evoluir para integrações reais. O backlog do MVP foi encerrado, e os proximos passos entram como pos-MVP.`

---

## Material de apoio

- `README.md`
- `docs/workflow_inicial.md`
- `docs/ambiente_local.md`
- `docs/playbook_callbacks_operacao.md`
- `docs/checklist_readiness_mvp.md`
- `docs/handoff_operacional_mvp.md`
- `docs/encerramento_mvp.md`
- `docs/teste_manual_ponta_a_ponta.md`
- `docs/arquitetura_aws_referencia.md`
