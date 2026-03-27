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
9. o `workflow service` atualiza a proposta para `document_analysis_in_progress`;
10. o `document service` executa a analise documental simulada;
11. o resultado e persistido no `proposal service`;
12. se aprovado, o workflow avanca para `credit_analysis_in_progress`;
13. o `credit-analysis service` executa a analise simulada;
14. o resultado e persistido no `proposal service`;
15. se aprovado, o workflow avanca para `fraud_analysis_in_progress`;
16. o `fraud-analysis service` executa a analise simulada;
17. o resultado e persistido no `proposal service`;
18. o workflow consolida a decisao final da proposta;
19. notificacoes sao registradas ao longo do fluxo sem bloquear a proposta e enviadas por SMTP local.

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
- o upload de documento passa por `BFF -> Document Service -> MinIO`;
- os resultados ficam persistidos no `proposal service`;
- as notificacoes ficam persistidas no `notification service` e sao enviadas para o Mailpit no ambiente local;
- o front le a proposta consolidada com cliente, documentos e resultados das analises.
