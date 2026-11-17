# Environment and API Endpoints

This directory is a public integration guide for Yao 2.0 frontend developers. It intentionally contains no internal test, staging, VPN, or LAN addresses.

## API base URL

Use the base URL supplied by your Yao operator:

```text
https://<your-yao-host>/v1
```

Replace `<your-yao-host>` with the deployment host. Production deployments should use HTTPS. The exact host, port, and path are deployment-specific.

## Reference assumptions

The guides assume:

- Yao 2.0 with the OpenAPI mounted under `/v1`.
- Browser clients that can use first-party cookies, directly or behind a same-origin proxy/BFF.
- Native desktop/mobile clients that can open a system browser and receive an OAuth authorization-code callback.
- A Web frontend that serves the Yao login route `/auth/entry` and honors its `return_to` query when continuing an OAuth authorization request. It should also serve `/auth/entry/success` and `/auth/entry/failure` or configure equivalent success/failure routes in the OpenAPI entry config. The Yao API redirects here; it does not ship these HTML pages itself.
- A Tao OAuth provider configured by the operator for Tao login.
- A Tai transport reachable by the host that runs a local Tai node; its port is published at `/.well-known/yao-transport`.
- HTTPS in production. Plain HTTP is only appropriate for controlled local development.

## Authentication guides

- [Web authentication](auth/web.md) — first-party browser session.
- [Desktop authentication](auth/desktop.md) — system browser + loopback OAuth callback.
- [Mobile authentication](auth/mobile.md) — iOS and Android system-browser OAuth callback.
- [Authentication API inventory](auth/api-endpoints.md) — all auth endpoints in one table.
- [Computer connections](features/computer-connections.md) — Computer lifecycle and Tai transport.
- [Computer API inventory](features/computer-api.md) — endpoints used by the Computer feature.

## Local development

If you run Yao locally, point your frontend dev proxy at your local Yao instance. Do not hardcode a shared test host in source code or documentation.

A same-origin proxy looks like:

```text
https://<your-dev-origin>/v1/*  ->  http://127.0.0.1:<yao-port>/v1/*
```

Ask your operator for deployment-specific endpoints. Do not put internal addresses, credentials, test hostnames, or test account details in a public document or repository.
