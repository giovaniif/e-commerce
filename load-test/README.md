# Teste de carga e performance (k6)

Scripts k6 para testar o checkout sob carga e com injeção de erros. As métricas podem ser enviadas ao InfluxDB e visualizadas no Grafana.

## Pré-requisitos

- [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) instalado localmente
- Stack rodando: `docker-compose up -d` (Order, Stock, Payment, Nginx, InfluxDB, Grafana)

**Estoque para carga:** no Docker Compose o Stock já sobe com `STOCK_INITIAL_QUANTITY=100000` (item 1), para o teste de carga não esgotar o estoque e gerar 409/500. Se rodar o Stock localmente, defina `STOCK_INITIAL_QUANTITY` alto (ex.: `100000`) antes do teste.

**Se o smoke falhar 100%:** o script loga no console o status e o corpo (ex.: `[smoke] 500 ... -> reserve stock request failed`). Veja a seção **Troubleshooting** abaixo.

**Erro de DNS ("lookup stock... server misbehaving") ou connection refused:** o compose usa uma rede explícita `app` para todos os serviços. Faça um restart completo para recriar a rede: `docker-compose down && docker-compose up -d --build`.

## Cenários do script

O script `checkout.js` define quatro cenários:

| Cenário     | Descrição |
|------------|-----------|
| **smoke**  | 2 VUs por 30s — valida que a API responde (checkout OK). |
| **error_mix** | 20 VUs por 2 min — ~70% checkouts válidos (200), ~10% sem Idempotency-Key (400), ~10% JSON inválido (400), ~10% item inexistente (500), ~10% estoque insuficiente (500). Verifica se a aplicação lida bem com erros. |
| **load_ramp** | Rampa 0→50 VUs em 1 min, mantém 50 VUs por 5 min, rampa 50→0 em 1 min. |
| **load_high** | 150 VUs por 10 min — centenas de milhares de requests para testar performance sustentada. |

## Executar

**Sem enviar métricas (só terminal):**
```bash
k6 run load-test/checkout.js
```

**Com base URL explícita (ex.: Order direto na porta 3131):**
```bash
k6 run --env BASE_URL=http://localhost:3131 load-test/checkout.js
```

**Enviando métricas para o Grafana (InfluxDB):**
```bash
k6 run --out influxdb=http://localhost:8086/k6 load-test/checkout.js
```

Com o stack no Docker, use `BASE_URL=http://localhost/order` (Nginx) ou `http://localhost:3131` (Order direto). O InfluxDB está em `localhost:8086`; o banco `k6` é criado automaticamente pelo k6 na primeira execução.

## Executar apenas um cenário

Exemplo: só carga alta por 5 minutos com 100 VUs:
```bash
k6 run --scenario load_high --duration 5m --vus 100 --out influxdb=http://localhost:8086/k6 load-test/checkout.js
```

Exemplo: só injeção de erros por 1 minuto:
```bash
k6 run --scenario error_mix --duration 1m --vus 20 --out influxdb=http://localhost:8086/k6 load-test/checkout.js
```

## Ver resultados no Grafana

1. Suba o stack com InfluxDB: `docker-compose up -d` (o `docker-compose.yml` já inclui o serviço `influxdb`).
2. Execute o k6 com `--out influxdb=http://localhost:8086/k6`.
3. Abra o Grafana em http://localhost:3000 (admin/admin).
4. Vá em **Dashboards** → pasta **E-commerce** → **k6 Load Test** (ou **Dashboards** → **k6 Load Test**).
5. Ajuste o intervalo de tempo (canto superior direito) para cobrir o período do teste.

No mesmo Grafana você pode correlacionar com métricas dos serviços (Prometheus), logs (Loki) e traces (Tempo) para analisar comportamento sob carga e em cenários de erro.

---

## Troubleshooting

### 500 ao chamar stock/payment (DNS "server misbehaving" ou "connection refused")

Todos os serviços estão na rede explícita **app** para o DNS do Docker resolver `stock` e `payment` de forma estável a partir do Order.

**Sempre que alterar o compose (rede ou env):**

```bash
docker-compose down
docker-compose up -d --build
```

Depois rode o k6 de novo. Não use `host.docker.internal` — o Order usa os hostnames `stock:3133` e `payment:3132` na rede `app`.

### Estoque esgotando (muitos 409/500)

Recrie o Stock para aplicar `STOCK_INITIAL_QUANTITY=100000`:
```bash
docker-compose up -d --force-recreate stock
```
