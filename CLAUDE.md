# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build main API binary
go build -o bin/api cmd/main/main.go

# Build all binaries (API + all daemon services)
bash scripts/bin.sh

# Build individual daemon services
go build -o bin/monitor cmd/admin/monitor.go
go build -o bin/auth_suspicios cmd/daemon/auth/suspicios.go
go build -o bin/healthcheck cmd/daemon/healthcheck/healthcheck.go
go build -o bin/server_priority cmd/daemon/server/priority.go
go build -o bin/vpn_refresh cmd/daemon/vpn/refresh.go
go build -o bin/payment cmd/daemon/payment/payment.go
```

## Tests

Only one test file exists: `pkg/ovpn/client_test.go`.

```bash
go test ./pkg/ovpn/...
```

## Service Management

```bash
bash scripts/start_services.sh    # start all systemd services
bash scripts/stop_services.sh     # stop all
bash scripts/restart_services.sh  # restart all
```

## Architecture Overview

MiracleLVPN is a VPN service backend with a Gin REST API and several independent daemon binaries.

**Entry point:** `cmd/main/main.go` — sets up Gin routes under `/api/v1/`, applies JWT middleware, registers all controllers, and starts the server on `$PORT`.

**Layer structure:**
- `internal/controller/http/controller/` — HTTP handlers (Auth, User, Server, Info, Payment, AdminMonitor)
- `internal/controller/http/middleware/` — JWT auth, device proof validation, Telegram error recovery
- `internal/services/` — Business logic (auth, user, servers, payment, crypt, info)
- `internal/repo/` — Repository pattern over GORM: User, Server, UserServer, Payment, PaymentPlan, AuthData, News, Info, KeyValue
- `internal/config/db/` — PostgreSQL connection via GORM; SQLite supported as fallback
- `pkg/` — External integrations: `ovpn/` (SSH-based OpenVPN management), `yookassa/` (Russian payment gateway), `tg/` (Telegram notifications)

**Daemon binaries** (each runs independently as a systemd service):
- `vpn_refresh` — regenerates VPN configs for active users
- `payment` — processes expired subscriptions and auto-renewals via YooKassa webhooks
- `healthcheck` — monitors VPN server health and alerts via Telegram
- `server_priority` — updates server load-balancing weights
- `kick_highload` — disconnects users on overloaded servers
- `monitor` — admin monitoring endpoint

**Authentication flow:** device-based JWT. Client POSTs device info → `AuthController` validates device proof (`$MII_VPN_PROOF`) → returns short-lived JWT → client sends JWT in `Authorization` header for all subsequent requests.

**Payment flow:** `PaymentController.Create` → YooKassa API → webhook POST to `/api/v1/payment/hook` → `payment` daemon activates subscription.

**VPN provisioning:** `pkg/ovpn` SSHes into VPN servers to run shell scripts (`$SSH_CREATE_USER_FILE`, `$SSH_REVOKE_USER_FILE`). WireGuard/Amnezia variant is handled by `scripts/amneziawg.sh`.

## Adding a New Server

To add a new server to the system, insert a row into the `servers` table with the appropriate `type` field:
- `type = "ovpn"` (default, backwards-compatible) — OpenVPN server managed via custom shell scripts
- `type = "amneziawg"` — AmneziaWG server managed via `wg-manage.sh`

The `VpnRouter` (`internal/services/vpn/router.go`) dispatches all VPN operations to the correct client based on `server.type`. No application code changes are needed to add new servers.

## Required Environment Variables

All config is loaded from `.env` via godotenv. Key variables:

| Group | Variables |
|---|---|
| Database | `DB_USER`, `DB_HOST`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `DB_SSLMODE`, `DB_TIMEZONE` |
| JWT | `JWT_SECRET_AUTH`, `JWT_SECRET_PAYMENT`, `JWT_DURATION_MIN` |
| SSH/VPN | `SSH_USER`, `SSH_STATUS_PATH`, `SSH_CREATE_USER_FILE`, `SSH_REVOKE_USER_FILE`, `SSH_CONFIGS_DIR` |
| Payment | `PAYMENT_SHOP_ID`, `PAYMENT_SECRET`, `PAYMENT_RETURN_URL`, `PAYMENT_EXPIRATION_SEC`, `PAYMENT_REMOVE_EXPIRED_INTERVAL_SEC`, `PAYMENT_AUTO_INTERVAL_SEC` |
| VPN config | `VPN_CONFIG_DIRATION_SEC` (note: typo in codebase), `FREE_TRIAL_SEC` |
| AmneziaWG | `AWG_SSH_USER` (defaults to `SSH_USER`), `AWG_MANAGE_SCRIPT` (defaults to `/usr/local/bin/wg-manage.sh`), `AWG_CLIENTS_DIR` (defaults to `/etc/wireguard/clients`) |
| Anti-fraud | `MII_VPN_PROOF` (format: `device1:key1::device2:key2`), `PROOF_BAN_IF_FAIL` |
| Telegram | `TG_HEALTHCHECK_TOKEN`, `TG_HEALTHCHECK_CHAT_ID` |
| Server | `PORT`, `PPROF_PORT`, `LOG_DIR`, `LOG_RETAIN`, `DEBUG` |
