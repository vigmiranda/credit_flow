# Estrategia de Deploy

## Objetivo

Definir um fluxo simples de entrega continua, coerente com o MVP atual baseado em imagens Docker.

## Estrategia recomendada

1. `push` ou `pull request` dispara validacao e build de imagens.
2. imagens versionadas por `commit SHA` sao publicadas no registry do ambiente.
3. ambiente `dev` recebe deploy automatico apos merge em `main`.
4. ambiente `hml` recebe promocao manual da mesma imagem validada em `dev`.
5. ambiente `prod` usa a mesma imagem promovida, nunca rebuild local.

## Passos do deploy em cada ambiente

- aplicar secrets por arquivo ou secret manager;
- atualizar `Deployment` ou equivalente com nova tag;
- aguardar `healthz` verde de todos os servicos;
- executar `scripts/smoke_mvp.ps1` ou variante remota contra o `BFF`;
- liberar trafego somente apos smoke test e metricas basicas estaveis.

## Regras de rollout

- usar rollout gradual no `BFF` primeiro, depois servicos internos stateful;
- manter compatibilidade retroativa de contrato entre `BFF` e servicos por pelo menos um ciclo;
- evitar mudanca simultanea de schema e codigo sem fallback;
- rollback deve reapontar para a imagem anterior, sem rebuild.

## Pendencias para evolucao

- publicacao automatica em registry;
- manifestos versionados de ambiente;
- estrategia de migracao de banco com passos forward-only;
- smoke remoto executado pelo pipeline apos deploy.
