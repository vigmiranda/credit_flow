# Seguranca Operacional

## Objetivo

Registrar os controles minimos de seguranca adicionados ao MVP e a forma esperada de operar os servicos em ambientes reais.

## Controles implementados no ciclo-8

- o `BFF` mascara `cpf`, `email` e `phone` ao expor a proposta consolidada para o front;
- o `BFF` mascara o `recipient` das notificacoes retornadas ao front;
- o `notification service` grava o destinatario mascarado para leitura operacional e uma copia criptografada para armazenamento em repouso;
- `proposal`, `customer`, `document` e `notification` aceitam segredos via variaveis `*_FILE`, permitindo consumo de secret montado em arquivo.

## Variaveis de secret suportadas

- `PROPOSAL_SERVICE_DATABASE_URL_FILE`
- `CUSTOMER_SERVICE_DATABASE_URL_FILE`
- `DOCUMENT_SERVICE_DATABASE_URL_FILE`
- `NOTIFICATION_SERVICE_DATABASE_URL_FILE`
- `NOTIFICATION_SERVICE_ENCRYPTION_KEY_FILE`
- fallback global opcional: `DATABASE_URL_FILE`

## Politica minima de mascaramento

- `cpf`: expor apenas os 4 ultimos digitos;
- `phone`: expor apenas os 4 ultimos digitos;
- `email`: manter somente o primeiro caractere antes de `@`;
- `recipient` de notificacao: aplicar a mesma regra de `email` para canal `email`.

## Politica minima para ambientes reais

- nunca commitar `DATABASE_URL` ou chaves de criptografia em arquivos versionados;
- preferir secret manager ou volume de secret montado como arquivo;
- rotacionar `NOTIFICATION_SERVICE_ENCRYPTION_KEY` por ambiente;
- usar chaves distintas para `dev`, `hml` e `prod`;
- limitar acesso ao banco e ao namespace dos servicos apenas aos workloads necessarios.
