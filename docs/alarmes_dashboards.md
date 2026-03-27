# Alarmes e Dashboards Minimos

## Fontes

- logs estruturados JSON com `correlation_id`;
- endpoint `/metrics` nos servicos `bff`, `proposal`, `workflow` e `notification`;
- smoke test ponta a ponta em `scripts/smoke_mvp.ps1`.

## Dashboard operacional minimo

- throughput por servico: `total_requests`;
- erros por servico: `total_errors`;
- latencia media por servico: `average_ms`;
- requisicoes em voo: `inflight_requests`;
- distribuicao por rota a partir do mapa `paths`.

## Alarmes minimos

- `bff` indisponivel por mais de 2 minutos;
- `workflow` com `total_errors` crescente por 5 minutos;
- `proposal` com latencia media acima de 500 ms por 10 minutos;
- `notification` com erro em mais de 5% das requisicoes no periodo de 10 minutos;
- falha do smoke test apos deploy em `dev`.

## Acao operacional recomendada

- correlacionar erro pelo `X-Correlation-Id` do front ate os servicos internos;
- validar saude em `/healthz` antes de reciclar instancia;
- usar o smoke test como gate de aceite apos rollout;
- abrir incidente apenas se houver impacto confirmado no fluxo de criacao, envio de documento ou consolidacao final.
