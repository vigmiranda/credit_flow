# Workflow Inicial do MVP

## Objetivo

Descrever o fluxo implementado para a simulacao das analises de documento, credito e fraude no MVP.

## Sequencia atual

1. o usuario cria a proposta no front;
2. o usuario salva os dados do cliente;
3. o usuario registra o metadado do documento;
4. o front envia o arquivo para o BFF em `multipart/form-data`;
5. o BFF encaminha o arquivo para o `document service`;
6. o `document service` grava o objeto no MinIO e marca o documento como `uploaded`;
7. o BFF atualiza a proposta para `documents_received`;
8. o BFF dispara o `workflow service` em modo assincrono;
9. o `workflow service` enfileira o job em Redis ou em fila local de fallback;
10. um worker do `workflow service` consome o job e atualiza a proposta para `document_analysis_in_progress`;
11. o `document service` executa a analise documental simulada;
12. o resultado e persistido no `proposal service`;
13. se aprovado, o workflow avanca para `credit_analysis_in_progress`;
14. o `credit-analysis service` executa a analise simulada;
15. o resultado e persistido no `proposal service`;
16. se aprovado, o workflow avanca para `fraud_analysis_in_progress`;
17. o `fraud-analysis service` executa a analise simulada;
18. o resultado e persistido no `proposal service`;
19. o workflow consolida a decisao final da proposta;
20. se houver falha tecnica, o worker reaplica o job ate o limite configurado e depois direciona a proposta para `manual_review`;
21. notificacoes sao registradas ao longo do fluxo sem bloquear a proposta e enviadas por SMTP local.
22. callbacks externos podem marcar documentos via webhook autenticado no BFF quando houver upload concluido fora da jornada sincrona do front.
23. quando `WORKFLOW_EXTERNAL_CREDIT_CALLBACKS` ou `WORKFLOW_EXTERNAL_FRAUD_CALLBACKS` estiverem ativos, o workflow pausa na etapa correspondente e aguarda o callback do parceiro.

## Regras de consolidacao

- `awaiting_additional_documents` encerra o fluxo pedindo complementacao;
- `manual_review` encerra o fluxo em revisao manual;
- `rejected` encerra o fluxo como reprovado;
- se documento, credito e fraude forem aprovados, a proposta vai para `approved`.

## Comportamento de simulacao

- analise documental:
  - arquivo com `ilegivel` ou `pendente` no nome gera `awaiting_additional_documents`;
  - arquivo com `manual` no nome gera `manual_review`;
  - demais casos geram `approved`.
- analise de credito:
  - renda abaixo de `2000` gera `rejected`;
  - e-mail com `manual` ou CPF terminando em `00` gera `manual_review`;
  - demais casos geram `approved`.
- analise de fraude:
  - e-mail com `fraude` ou CPF com digitos repetidos gera `rejected`;
  - telefone terminando em `0000` ou endereco com `manual` gera `manual_review`;
  - demais casos geram `approved`.

## Observacoes

- o disparo do workflow e assincrono a partir do BFF;
- o processamento do workflow agora passa por fila com retry configuravel;
- jobs esgotados vao para DLQ e ficam refletidos nas metricas do workflow;
- a DLQ pode ser inspecionada e reprocessada por endpoints internos do `workflow service`;
- o upload de documento passa por `BFF -> Document Service -> MinIO`;
- o BFF expoe um webhook inicial em `/api/v1/webhooks/storage/document-uploaded` com assinatura HMAC em `X-Webhook-Signature`;
- o BFF tambem expoe `/api/v1/webhooks/partners/credit-analysis` e `/api/v1/webhooks/partners/fraud-analysis` com `X-Webhook-Event-Id` e `X-Webhook-Timestamp` para anti-replay;
- cada callback recebido passa a gerar um registro de auditoria consultavel, correlacionado com a proposta consolidada e com suporte a `replay-release` manual;
- a auditoria expirada pode ser limpa por `POST /internal/webhooks/audit/cleanup`, enquanto as politicas por parceiro e janela sao controladas por `BFF_ALLOWED_*_PROVIDERS` e `BFF_*_WEBHOOK_MAX_AGE_SECONDS`;
- o BFF tambem pode limitar bursts por rota e parceiro com `BFF_*_RATE_LIMIT` e `BFF_*_RATE_WINDOW_SECONDS`, devolvendo `429` com `Retry-After` quando necessario;
- os resultados ficam persistidos no `proposal service`;
- as notificacoes ficam persistidas no `notification service` e sao enviadas para o Mailpit no ambiente local;
- o front le a proposta consolidada com cliente, documentos e resultados das analises;
- a stack local inclui um issuer `mock-oidc` para exercitar o login OIDC completo.
