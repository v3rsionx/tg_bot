# tgbot-versionx

Production-ready Go foundation for a high-performance Telegram bot on Windows VPS.

The bot uses SQLite for operational data and LMDB for large-scale search indexes. The repository currently ships a Clean Architecture scaffold with a validated configuration system. Domain logic, Telegram handlers, and storage adapters are intentionally not implemented yet.

## Architecture

Dependencies point inward:

| Layer | Responsibility |
| --- | --- |
| `cmd/` | Application entry points and composition |
| `internal/service` | Application use cases |
| `internal/repository` | Data-access ports |
| `internal/models` | Domain models |
| `internal/database`, `internal/telegram`, `internal/logger` | Infrastructure adapters |

Transport concerns (routing, middleware, keyboards, handlers) stay inside the Telegram adapter. Storage implementations stay inside SQLite and LMDB adapters.

## Folder Structure

```text
.
├── cmd/
│   ├── bot/                 # Telegram bot executable
│   └── importer/            # Search-data import executable
├── configs/                 # Deployment configuration templates
├── data/                    # Local SQLite and LMDB data (ignored)
├── docs/                    # Technical documentation
├── dumps/                   # Export artifacts (ignored)
├── internal/
│   ├── config/              # Environment loading and validation
│   ├── database/
│   │   ├── sqlite/          # SQLite adapter
│   │   └── lmdb/            # LMDB adapter
│   ├── importer/            # Ingestion boundary
│   ├── logger/              # Logging boundary
│   ├── models/              # Domain models
│   ├── repository/          # Data-access ports
│   ├── search/              # Search boundary
│   ├── service/             # Application use cases
│   ├── telegram/            # Telegram transport
│   │   ├── handlers/
│   │   ├── keyboard/
│   │   ├── middleware/
│   │   └── router/
│   └── utils/               # Shared helpers
├── logs/                    # Runtime logs (ignored)
├── migrations/              # Schema migrations
├── scripts/                 # Operational scripts
├── .env.example
├── go.mod
├── Makefile
└── README.md
```

## Tech Stack

- Go 1.26
- Telegram Bot API
- SQLite for bot and operational data
- LMDB for high-throughput search storage
- Windows VPS deployment target

## Requirements

- Go 1.26 or later
- Telegram bot token from [BotFather](https://t.me/BotFather)
- Disk capacity for SQLite, LMDB indexes, logs, and imports
- Write access to `data/`, `logs/`, and `dumps/`

## Installation

```powershell
git clone https://github.com/<owner>/tgbot-versionx.git
Set-Location tgbot-versionx
Copy-Item .env.example .env
go mod download
```

Fill in `.env` with production values. Never commit `.env`, database files, LMDB environments, or logs.

## Configuration

All settings are required and validated at startup. Invalid configuration terminates the process immediately.

| Variable | Description |
| --- | --- |
| `BOT_TOKEN` | Telegram bot API token |
| `BOT_OWNER_IDS` | Comma-separated unique positive Telegram user IDs |
| `SQLITE_PATH` | SQLite database file path |
| `LMDB_ID_PATH` | LMDB directory for ID index |
| `LMDB_PHONE_PATH` | LMDB directory for phone index |
| `LMDB_USERNAME_PATH` | LMDB directory for username index |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` |
| `POINTS_PER_SEARCH` | Positive integer cost per search |
| `MAX_SEARCH_RESULT` | Positive integer result limit |

The three LMDB paths must be distinct. Prefer absolute paths on Windows services where the working directory is not guaranteed.

## Running Bot

```powershell
go run ./cmd/bot
```

Or:

```powershell
go build -o bin/bot.exe ./cmd/bot
.\bin\bot.exe
```

The bot currently validates configuration and exits. Telegram integration is not implemented yet.

## Running Importer

```powershell
go run ./cmd/importer
```

Or:

```powershell
go build -o bin/importer.exe ./cmd/importer
.\bin\importer.exe
```

The importer currently validates configuration and exits. Ingestion is not implemented yet.

## Development

```powershell
make fmt
make vet
make test
make build
```

## Security

- Keep `BOT_TOKEN` and owner IDs only in `.env` or a secret store.
- Do not commit `.env`, `*.db`, LMDB files, logs, dumps, or build artifacts.
- Restrict filesystem permissions on `.env` and `data/` to the service account.

## Future Features

- Telegram routing, middleware, and handlers
- SQLite repositories and migrations
- LMDB indexing and high-throughput search
- Streaming, resumable imports with progress reporting
- Structured logging, health checks, and metrics
- Windows service installation and deployment automation
- Backups, retention policies, and operational runbooks

## License

All rights reserved. No open-source license has been granted for this repository.
