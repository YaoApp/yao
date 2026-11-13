# Tao Cloud Mock

In-process Tao Computers API v1 mock for Yao purchase/provisioning tests and UAT integration.

## Test API Key

Default fixed test key:

```text
sk-tao-e2e
```

Override:

```bash
TAO_MOCK_API_KEY=sk-tao-uat go run ./e2e/taocloud/cmd/taocloud
```

Multiple keys are supported for in-process tests only:

```go
server.SetAPIKeys("sk-tao-a", "sk-tao-b")
```

## Standalone Server

```bash
cd yao
TAO_MOCK_ADDR=127.0.0.1:18080 \
TAO_MOCK_API_KEY=sk-tao-e2e \
go run ./e2e/taocloud/cmd/taocloud
```

## UAT Wiring

```text
Tao endpoint: http://127.0.0.1:18080
Tao key:      sk-tao-e2e (or TAO_MOCK_API_KEY)

Yao account:  PUT /v1/tao/account {"api_key":"sk-tao-e2e"}

Resources:
  starter-2c2g  MONTH
  flex-2c2g     HOUR + MONTH
```

## Implemented Contract

- `GET /v1/health`
- `GET /v1/catalog`, `GET /v1/services`
- `GET /v1/computers/resources`
- `GET /v1/computers/resources/{id}/capacity?region=&billing_unit=`
- `POST /v1/computers` with `billing_unit`, `period_months`, `ssh.public_key`, `Idempotency-Key`
- `GET /v1/computers/{id}` (`404` after delete)
- `GET /v1/computers?status=&region=&page=&size=`
- `POST /v1/computers/{id}/start|stop|reboot`
- `DELETE /v1/computers/{id}` (`202` first, `204` repeat)
