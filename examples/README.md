# Examples

Runnable examples for testing gossm against real AWS Secrets Manager secrets.
Run all commands from the **repository root**.

## Setup

```bash
cp examples/.env.example examples/.env
# Edit examples/.env — fill in your AWS credentials and secret ARNs
export $(cat examples/.env | xargs)
```

> These credentials are for local development only.
> In production, use an IAM role — see the root [README](../README.md#authentication).

## basic

Fetches a single key from one secret and prints it.

Secret shape: `{"api_key":"..."}`

```bash
go run ./examples/basic
```

## variables

Fetches multiple keys from one secret into separate Go variables.

Secret shape: `{"db_user":"...","db_password":"...","api_key":"..."}`

```bash
go run ./examples/variables
```

## multisecrets

Fetches multiple keys from two different ARNs into a typed `Config` struct.

Secret shapes:
- `SECRET_ARN_DB`  → `{"db_user":"...","db_password":"..."}`
- `SECRET_ARN_API` → `{"api_key":"...","api_secret":"..."}`

```bash
go run ./examples/multisecrets
```

## server

The recommended production pattern: load all secrets once at startup into a
`Config` struct, then serve them from memory on every request via `GET /info`.

Same secrets as `multisecrets`.

```bash
go run ./examples/server
curl http://localhost:8080/info
```
