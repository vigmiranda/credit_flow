# Padroes de Desenvolvimento

## Objetivo

Definir convencoes minimas para manter consistencia entre front, backend e infraestrutura desde o inicio do projeto.

## Organizacao do repositorio

- `apps`: aplicacoes de interface
- `services`: servicos de backend e workers
- `infra`: artefatos de infraestrutura e ambiente local
- `docs`: documentacao tecnica
- `planning`: backlog e acompanhamento de ciclos

## Convencoes gerais

- usar monorepo no primeiro corte;
- manter nomes de pastas e arquivos em minusculo com hifen quando necessario;
- isolar cada servico em seu proprio diretorio;
- evitar compartilhar codigo cedo demais; extrair bibliotecas so quando houver reuso real;
- manter configuracao fora do codigo com variaveis de ambiente.

## Backend

- linguagem principal: Go;
- APIs externas em REST no primeiro corte;
- handlers finos, regras em casos de uso e dominio isolado;
- logs estruturados com `correlation-id`;
- operacoes criticas devem ser idempotentes.

## Frontend

- stack base: Next.js com TypeScript;
- organizar por feature sempre que houver fluxo de negocio claro;
- validar formularios no cliente antes de enviar ao backend;
- separar componentes visuais de adaptadores de acesso a API.

## Infra e ambiente local

- usar `docker-compose` no primeiro momento para dependencias locais;
- manter `infra/docker` para ambiente local e `infra/terraform` para infraestrutura alvo;
- nao tentar reproduzir toda a AWS localmente no primeiro corte.

## Qualidade

- todo servico novo deve nascer com estrutura minima para teste;
- lint e testes automatizados entram antes do aumento de escopo;
- contratos de API e eventos devem ser versionados.

## Git e fluxo de trabalho

- trabalhar em ciclos curtos;
- atualizar `planning/backlog_implementacao.md` ao fechar cada ciclo;
- evitar mudancas grandes sem criterio de aceite claro;
- documentar decisoes que alterem escopo, arquitetura ou dominio.

## Definicao de pronto inicial

- estrutura do modulo criada no diretorio correto;
- convencoes de configuracao definidas;
- contrato ou interface minima documentada;
- impacto no backlog refletido no ciclo atual.

