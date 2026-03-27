# Padroes Transversais

## Correlation ID

- header padrao: `X-Correlation-Id`;
- se o cliente nao enviar, o BFF gera um valor e devolve na resposta;
- o mesmo valor deve seguir logs, eventos e chamadas internas.

## Idempotencia

- header padrao: `Idempotency-Key`;
- obrigatorio em operacoes `POST` do BFF no contexto produtivo;
- no MVP pode ser opcional na borda, mas o contrato ja deve aceita-lo;
- a deduplicacao final sera responsabilidade da camada de aplicacao.

## Padrao de erro

Formato de resposta:

```json
{
  "code": "invalid_request",
  "message": "payload invalido",
  "correlation_id": "corr_123",
  "details": {
    "field": "cpf"
  }
}
```

Codigos iniciais:

- `invalid_request`
- `not_found`
- `conflict`
- `unauthorized`
- `internal_error`

## Status HTTP iniciais

- `200` para leitura e geracao de URL;
- `201` para criacao de proposta;
- `202` para aceite de processamento assincrono;
- `400` para erro de validacao;
- `404` para recurso inexistente;
- `409` para conflito de estado ou duplicidade;
- `500` para erro interno.

## Convencoes de tempo e identificadores

- datas em `RFC3339`;
- ids tecnicos em formato string;
- mascarar dados sensiveis em logs e eventos;
- `protocol` e o identificador amigavel exibido ao cliente;
- `proposal_id` e o identificador tecnico interno.

## Logs

- logs em JSON;
- sem CPF completo, renda completa ou dados de documento em texto claro;
- incluir sempre `service`, `operation`, `correlation_id` e `proposal_id` quando disponivel.

