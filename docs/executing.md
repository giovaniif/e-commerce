# Como executar

## Pré-requisitos

- Docker e Docker Compose
- Go 1.21+ (para desenvolvimento local)

---

## URLs nos gateways

Dependendo de como você roda os serviços (**Docker** ou **local**), é necessário usar URLs diferentes nos gateways do **Order Service** (que chama Stock e Payment).

| Onde | Arquivo | Docker | Local |
|------|---------|--------|--------|
| Order → Stock | `order/infra/gateways/stock.go` | `http://stock:3133/reserve`, `http://stock:3133/release`, `http://stock:3133/complete` | `http://localhost:3133/...` |
| Order → Payment | `order/infra/gateways/payment.go` | `http://payment:3132/charge` | `http://localhost:3132/charge` |

- **Docker Compose:** use os hostnames do serviço (`stock`, `payment`) — cada container resolve o nome do outro.
- **Local (um processo por terminal):** use `localhost` e a porta do serviço (3132 Payment, 3133 Stock).

No código, hoje há comentários com a URL para Docker e a linha ativa para local. Inverta conforme o modo em que estiver rodando.

---

## Executando com Docker Compose

```bash
# Na raiz do projeto
docker-compose up --build
```

Serviços disponíveis:

- Order: http://localhost/order (via Nginx :80)
- Payment: http://localhost/payment
- Stock: http://localhost/stock
- Redis: porta 6379 (idempotência do checkout)

O **Order** usa **Redis** para persistir idempotência do checkout quando `REDIS_ADDR` está definido (no Docker já vem `REDIS_ADDR=redis:6379`). TTL das chaves: **24 horas**. Sem Redis (ex.: local sem `REDIS_ADDR`), a idempotência fica em memória.

Garanta que os gateways do Order usem as URLs com hostname `stock` e `payment` (veja tabela acima).

---

## Executando localmente

Um terminal por serviço (na raiz do projeto):

```bash
# Terminal 1 - Stock
cd stock && go run main.go

# Terminal 2 - Payment
cd payment && go run main.go

# Terminal 3 - Order
cd order && go run main.go
```

Para o Order conseguir falar com Stock e Payment, os gateways devem usar `http://localhost:3133` (Stock) e `http://localhost:3132` (Payment). Veja a tabela acima.

**Redis (opcional):** para usar idempotência persistente localmente, suba um Redis (ex.: `docker run -p 6379:6379 redis:alpine`) e defina `REDIS_ADDR=localhost:6379` ao rodar o Order. Sem isso, a idempotência do checkout fica em memória.

**Nginx (opcional):** para acessar via porta 80 como no Docker:

```bash
docker run -p 80:80 -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf:ro nginx
```

---

## Testando o checkout

**Com Docker (via Nginx):**

```bash
curl -X POST http://localhost/order/checkout \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: abc-123" \
  -d '{"itemId": 1, "quantity": 2}'
```

**Local (Order na porta 3131):**

```bash
curl -X POST http://localhost:3131/checkout \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: abc-123" \
  -d '{"itemId": 1, "quantity": 2}'
```
