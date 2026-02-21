# Como executar

## Pré-requisitos

- Docker e Docker Compose
- Go 1.21+ (para desenvolvimento local)

---

## URLs nos gateways (Order → Stock, Order → Payment)

O **Order Service** lê as URLs de Stock e Payment pelas variáveis de ambiente **`STOCK_BASE_URL`** e **`PAYMENT_BASE_URL`**.

- **Docker Compose:** o `docker-compose.yml` já define `STOCK_BASE_URL=http://stock:3133` e `PAYMENT_BASE_URL=http://payment:3132` no serviço `order`. Nada a alterar.
- **Local (um processo por terminal):** se não definir essas variáveis, o código usa por padrão `http://localhost:3133` (Stock) e `http://localhost:3132` (Payment). Para sobrescrever, crie um `.env` a partir de `.env.example` ou exporte as variáveis no shell.

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
- **Grafana:** http://localhost:3000 (usuário `admin`, senha `admin`) — métricas (Prometheus) e logs (Loki)
- **Prometheus:** http://localhost:9090 (scrape de `/metrics` em Order, Payment, Stock)
- **Loki:** porta 3100 (armazenamento de logs; Promtail envia os logs dos containers)

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

Para o Order conseguir falar com Stock e Payment, use as variáveis `STOCK_BASE_URL` e `PAYMENT_BASE_URL`; sem elas o código usa por padrão `http://localhost:3133` e `http://localhost:3132`. Opcional: copie `.env.example` para `.env` e ajuste (o Order não carrega `.env` automaticamente; use `export` ou uma ferramenta que leia `.env`).

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

---

## Métricas e logs (Grafana)

Com `docker-compose up`, sobem também **Prometheus**, **Loki**, **Promtail** e **Grafana**.

**Logs no Loki:** os serviços **order**, **payment** e **stock** enviam logs **diretamente** ao Loki por HTTP (variável `LOKI_URL=http://loki:3100` no Docker). Não é necessário plugin nem Promtail. No Grafana Explore (Loki), use a query `{job="order"}`, `{job="payment"}` ou `{job="stock"}` (ou `{job=~"order|payment|stock"}` para todos).

- **Grafana:** http://localhost:3000 — login `admin` / `admin`. Já vêm provisionados três datasources:
  - **Prometheus** (padrão): métricas dos três serviços (latência, contagem de requests por método/path/status, etc.).
  - **Loki:** logs enviados pelas aplicações; filtro por `job` (order, payment, stock).
  - **Tempo:** traces distribuídos (OpenTelemetry); correlação com Loki via **trace to logs**.

Há um **dashboard provisionado** na pasta **E-commerce** (menu lateral): **E-commerce services**. Use o filtro **Application** (order, payment, stock) para ver por serviço:

- **Requests by status** — taxa de requests por status (2xx, 4xx, 5xx)
- **Throughput** — req/s
- **p99 / p95 latency** — percentis de latência por path
- **Logs** — logs do Loki filtrados pela aplicação

Para importar outro dashboard manualmente: **Dashboards → New → Import** e cole o JSON de `monitoring/grafana/dashboards/e-commerce-services.json`.

Para ver **logs** no Explore: **Explore → Loki**, query `{job="order"}`, `{job="payment"}` ou `{job="stock"}` (ou `{}` para todos). Selecione **Last 15 minutes**. No dashboard **E-commerce services**, use o filtro **Application** para filtrar os logs por serviço.

**Se não aparecer nenhum log:** confira se o stack subiu com `docker compose up -d --build` (para aplicar `LOKI_URL`). Os serviços só enviam logs ao Loki quando `LOKI_URL` está definido (já definido no compose). Verifique se o Loki está acessível: `docker compose logs loki`.

**Se o Grafana não abrir (erro "database is locked"):** os dados do Grafana ficam no volume `grafana_data`. Para começar do zero: `docker compose down` e depois `docker volume rm payments_grafana_data` (ou o nome do volume que aparecer em `docker volume ls`). Em seguida `docker compose up -d`. O provisionamento recria os datasources e o dashboard da pasta E-commerce.

---

## Tracing distribuído (Tempo)

Com `docker-compose up`, o **Tempo** recebe traces OTLP (HTTP na porta 4318) dos serviços **order**, **payment** e **stock**. O **trace_id** da request é o mesmo que o **X-Request-ID** (quando enviado com 32 caracteres hexadecimais), permitindo correlacionar um único request em todos os serviços.

**Como ver um trace no Grafana:**

1. Abra **Explore** (ícone de bússola) e selecione o datasource **Tempo**.
2. Em **Query**, escolha **Search** e use:
   - **Trace ID:** o valor do header `X-Request-ID` da request (ex.: o que você passou no `curl` ou o que a API retornou no header de resposta).
3. Clique em **Run query** para ver a árvore de spans (Order → Stock reserve, Order → Payment charge, etc.).

**Trace → Logs:** no datasource Tempo está configurado **trace to logs** apontando para o Loki. Ao abrir um span no Tempo, use o link **Logs for this span** (ou equivalente) para ver no Loki os logs do mesmo `request_id`, permitindo inspecionar a mesma request em traces e logs.
