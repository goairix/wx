# Health Card Client Design

## Goal

Add a standalone `health_card` package for the Tencent Electronic Health Card Open Platform APIs used by the WeChat mini-program binding-card plugin (`serviceId=138`). The package will let a business backend exchange plugin one-time codes for health-card data without coupling the feature to `mini_program/auth` or WeChat callback handling.

## Scope

The first version supports the three backend APIs referenced by the plugin documentation:

1. `registerHealthCard` — exchange `wechatCode` plus registration data for a newly registered card.
2. `getHealthCardByHealthCode` — exchange `healthCode` for an existing card.
3. `getRegInfoByCode` — exchange `regInfoCode` for form data after a card-management failure.

It does not implement old-patient matching, scan-code field submission, plugin UI components, or WeChat server callbacks.

## Package boundary and API

Create `health_card` as an independent package. Its client owns the Tencent Health Card base URL, credentials, signing, JSON transport, and platform error decoding. It does not use the existing `support/http` package because that package is fixed to `api.weixin.qq.com`.

The client is configured with:

- `AppSecret`: used only for request signing;
- `AppToken`: platform-issued request credential;
- `HospitalID`: hospital binding identifier;
- `ChannelNum`: optional channel number, defaulting to the platform default;
- injectable HTTP client/base URL for deterministic tests.

Public methods:

```go
RegisterHealthCard(req RegisterHealthCardRequest) (RegisterHealthCardResponse, error)
GetHealthCardByHealthCode(healthCode string) (HealthCardResponse, error)
GetRegInfoByCode(code string) (RegistrationInfoResponse, error)
```

The package should also expose request/response DTOs so callers can pass registration fields and consume card/registration data without raw maps.

## Request construction and signing

Every request is a JSON object with `commonIn` and `req`. `commonIn` contains `appToken`, a generated unique `requestId`, `hospitalId`, Unix-second `timestamp`, `channelNum`, and `sign`; optional relation fields remain available on the public common-input type.

The signature follows the platform rule: flatten non-empty values from `commonIn` (excluding `sign`) and `req`, sort by parameter name, join as `key=value` pairs separated by `&`, append `AppSecret`, compute SHA-256, and Base64-encode the digest. Nested objects are serialized deterministically before being included in the flattened value.

The client must treat non-zero `commonOut.resultCode` as an error and retain the platform request ID in the typed error when available. HTTP status failures and malformed JSON are also returned as errors.

## Data flow

The mini-program plugin produces `wechatCode`, `healthCode`, or `regInfoCode`. The business backend receives one code, calls the corresponding `health_card.Client` method, and returns the result to the mini-program. The package never receives plugin callbacks and never exposes `AppSecret` or `AppToken` to client code.

## Testing

Tests use `httptest.Server` or an injected `http.Client` and cover:

- exact signature generation, including sorted fields and omission of empty values;
- request paths and JSON envelopes for all three methods;
- successful response decoding;
- non-zero `commonOut.resultCode` conversion to an error;
- transport, non-2xx, and malformed-response errors.

No existing package behavior should change.
