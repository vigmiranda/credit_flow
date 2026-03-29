# Teste Manual Ponta a Ponta

## Objetivo

Executar o fluxo completo localmente, validando:

- subida da stack;
- login no front;
- criacao da proposta;
- cadastro do cliente;
- registro e upload do documento;
- processamento do workflow;
- notificacoes e timeline;
- sinais operacionais finais.

## Pre-requisitos

- Docker Desktop ativo e saudavel;
- portas locais livres: `3000`, `18080`, `18084`, `19090`, `15432`, `6379`, `9000`, `9001`, `8025`;
- projeto aberto em:

```powershell
cd C:\Users\vigmi\OneDrive\Documentos\develop\credit_flow
```

## Visao geral do fluxo

```text
1. Subir stack Docker
2. Acessar front web
3. Fazer login local
4. Criar proposta
5. Salvar cliente
6. Registrar documento
7. Enviar arquivo real
8. Aguardar analises
9. Validar timeline e notificacoes
10. Validar overview operacional
11. Derrubar stack
```

## Etapa 1. Validar Docker

Execute:

```powershell
docker version
docker info
```

Esperado:

- `docker version` com secao `Server`;
- `docker info` sem erro de engine indisponivel.

Se falhar:

- abra o Docker Desktop;
- aguarde o engine estabilizar;
- repita os comandos.

## Etapa 2. Subir a stack local

Execute:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\up_local_stack.ps1
```

Esperado:

- `docker compose` sobe `postgres`, `redis`, `mailpit`, `minio`, `mock-oidc`, servicos Go, `bff` e `web`;
- sem erro de porta ocupada;
- sem falha de build.

Cheque rapidamente:

```powershell
docker compose -f infra/docker/docker-compose.yml ps
```

Esperado:

- containers principais em `Up`.

## Etapa 3. Validar endpoints basicos

Abra no navegador ou teste por comando:

- Front: `http://localhost:3000`
- BFF health: `http://localhost:18080/healthz`
- Workflow health: `http://localhost:18084/healthz`
- Mailpit: `http://localhost:8025`
- MinIO Console: `http://localhost:9001`

Esperado:

- front carregando;
- `healthz` respondendo `{ "status": "ok" }`.

## Etapa 4. Abrir o front

No navegador:

```text
http://localhost:3000
```

Esperado:

- redirecionamento para login;
- tela de autenticacao local/OIDC mock.

## Etapa 5. Fazer login

Acione o login local exibido na interface.

Esperado:

- autenticacao bem-sucedida;
- redirecionamento para a home do fluxo;
- banner de sessao ativa.

Validacao visual:

- nome do operador logado;
- e-mail/role no topo;
- botao para encerrar sessao.

## Etapa 6. Criar proposta

Na home, clique em:

```text
Criar proposta
```

Esperado:

- geracao de `proposal_id`;
- geracao de protocolo;
- status inicial atualizado.

Validacao visual:

- card `Proposta` preenchido;
- mensagem de sucesso na faixa inferior.

## Etapa 7. Salvar cliente

Preencha os campos:

- Nome completo: `Maria Silva`
- CPF: `12345678901`
- Nascimento: `1990-01-01`
- E-mail: `maria@example.com`
- Telefone: `11999999999`
- Renda mensal: `5000`
- Endereco: `Rua Exemplo, 123`

Clique em:

```text
Salvar cliente
```

Esperado:

- status da proposta avancando para etapa documental;
- notificacao registrada;
- mensagem de sucesso no rodape.

Validacao visual:

- progresso da jornada atualizado;
- dados do cliente refletidos na proposta consolidada.

## Etapa 8. Registrar documento

Na secao de documento, mantenha ou preencha:

- Tipo: `id_front`
- Arquivo: `rg.jpg`
- Content-Type: `image/jpeg`

Clique em:

```text
Registrar documento
```

Esperado:

- geracao de `document_id`;
- documento aparecendo na lista;
- status ainda aguardando upload real.

## Etapa 9. Enviar arquivo real

Selecione um arquivo local qualquer, preferencialmente uma imagem pequena.

Clique em:

```text
Enviar arquivo
```

Esperado:

- upload via `BFF -> Document Service -> MinIO`;
- documento com status `uploaded`;
- workflow disparado automaticamente.

Validacao visual:

- documento aparece como enviado;
- status da proposta comeca a evoluir.

## Etapa 10. Aguardar processamento

Clique em:

```text
Atualizar proposta
```

se necessario, algumas vezes, com alguns segundos de intervalo.

Esperado:

- progresso por `document_analysis_in_progress`;
- progresso por `credit_analysis_in_progress`;
- progresso por `fraud_analysis_in_progress`;
- chegada a status terminal.

Status terminal esperado:

- `approved`

Tambem podem existir, dependendo do fluxo:

- `rejected`
- `manual_review`
- `awaiting_additional_documents`

## Etapa 11. Validar analises

Na area de resultados, valide:

- 3 analises registradas;
- `document`;
- `credit`;
- `fraud`.

Esperado:

- cada analise com `result`, `provider`, `score` e `reason`.

## Etapa 12. Validar timeline

Na timeline da proposta, valide a presenca de eventos de:

- criacao;
- mudancas de status;
- notificacoes;
- callbacks operacionais, quando existirem.

Esperado:

- ordem cronologica coerente;
- evolucao completa ate o status terminal.

## Etapa 13. Validar notificacoes

Abra:

```text
http://localhost:8025
```

Esperado:

- emails simulados entregues no Mailpit;
- mensagens correspondentes aos momentos do fluxo.

## Etapa 14. Validar storage

Abra:

```text
http://localhost:9001
```

Esperado:

- bucket `proposal-documents`;
- objeto enviado presente no storage.

## Etapa 15. Validar overview operacional

Execute:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\ops_overview.ps1
```

Esperado:

- `DLQ: 0`
- `Rate limited: 0`
- `Invalid provider: 0`
- `Sem alertas operacionais.`

Validacao ideal:

- `Workflow queue` com `Enqueued: 1` e `Processed: 1`
- `Retried: 0`
- `Dead letter: 0`

## Etapa 16. Validar prontidao automatizada

Se quiser repetir a validacao completa automatizada:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release_readiness.ps1
```

Esperado:

- verify completo;
- smoke passando;
- overview sem alertas criticos;
- stack derrubada ao final.

## Etapa 17. Derrubar a stack

Quando terminar:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\down_local_stack.ps1
```

Esperado:

- containers removidos;
- rede Docker removida;
- ambiente limpo.

## Resultado esperado do teste manual

O teste ponta a ponta deve ser considerado bem-sucedido quando:

- o front estiver acessivel;
- o login funcionar;
- a proposta puder ser criada;
- o cliente puder ser salvo;
- o documento puder ser registrado e enviado;
- o workflow chegar a status terminal;
- as analises aparecerem na tela;
- a timeline refletir o fluxo;
- o Mailpit receber notificacoes;
- o MinIO armazenar o arquivo;
- o overview operacional vier sem alerta critico.

## Troubleshooting rapido

### Front nao abre em `3000`

Execute:

```powershell
docker compose -f infra/docker/docker-compose.yml ps
docker compose -f infra/docker/docker-compose.yml logs web --tail 100
```

### BFF nao responde em `18080`

Execute:

```powershell
docker compose -f infra/docker/docker-compose.yml logs bff --tail 100
```

### Workflow nao conclui

Execute:

```powershell
Invoke-RestMethod http://localhost:18084/internal/dlq
powershell -ExecutionPolicy Bypass -File .\scripts\ops_overview.ps1
```

### Erro de porta ocupada

Execute:

```powershell
docker compose -f infra/docker/docker-compose.yml down --remove-orphans
```

e feche processos locais conflitantes.

## Referencias

- `scripts/up_local_stack.ps1`
- `scripts/down_local_stack.ps1`
- `scripts/smoke_docker_stack.ps1`
- `scripts/ops_overview.ps1`
- `scripts/release_readiness.ps1`
- `docs/checklist_readiness_mvp.md`
- `docs/handoff_operacional_mvp.md`
