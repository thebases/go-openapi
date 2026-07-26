# Fiber Example API

Sample API built with [`go-openapi`](https://github.com/thebases/go-openapi) and Fiber, demonstrating typed route
registration and every OpenAPI 3.2.0 Security Scheme Object case.

## Running this example

```bash
cd examples/fiber
go run .
```

The server listens on `:3000`. Docs UI: `http://localhost:3000/docs/`. Raw document: `http://localhost:3000/openapi.json`.

## Trying the secured endpoints

Every `/auth/*` route below requires its matching credential. Use the "Authorize" button in the docs UI, or `curl`
directly with the fixed sample values in the table below — they are demo-only constants, not real secrets, and are
checked as-is by the example handlers.

| Endpoint | Security scheme | Type | How to authenticate |
| --- | --- | --- | --- |
| `GET /auth/apikey/profile` | `merchantApiKey` | apiKey (header) | `X-API-Key: sample-api-key-123` |
| `GET /auth/basic/profile` | `merchantBasicAuth` | http (basic) | `Authorization: Basic <base64(sample-user:sample-pass)>` |
| `GET /auth/bearer/profile` | `merchantBearerAuth` | http (bearer, JWT) | `Authorization: Bearer sample-bearer-token-123` |
| `GET /auth/oauth/session` | `merchantOAuth` | oauth2 (all flows) | `Authorization: Bearer sample-oauth-token-456` |
| `GET /auth/mtls/profile` | `merchantMutualTLS` | mutualTLS | Docs-only — declared for spec coverage, not enforced |
| `GET /auth/oidc/profile` | `merchantOpenIdConnect` | openIdConnect | Docs-only — declared for spec coverage, not enforced |

`GET /merchants/{id}` also requires `X-API-Key: sample-api-key-123` (unenforced in this route, credential is
documented for the OpenAPI schema only). `POST /merchants` requires the `merchantOAuth` scheme with the
`merchant:write` scope.

Example calls:

```bash
curl -H "X-API-Key: sample-api-key-123" http://localhost:3000/auth/apikey/profile
curl -u sample-user:sample-pass http://localhost:3000/auth/basic/profile
curl -H "Authorization: Bearer sample-bearer-token-123" http://localhost:3000/auth/bearer/profile
curl -H "Authorization: Bearer sample-oauth-token-456" http://localhost:3000/auth/oauth/session
```

Requests without the matching credential return `401 Unauthorized` for the enforced routes above.
