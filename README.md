# ClickHouse Alerting System

A lightweight alerting platform that monitors ClickHouse databases by evaluating periodic queries against configurable thresholds and sending notifications when conditions are met.

## Features

- **Rule-based alerting** — define custom ClickHouse queries with threshold conditions (gt, gte, lt, lte, eq, neq)
- **State machine** — alerts transition through inactive, pending, and firing states with configurable pending durations
- **Notifications** — send alerts via Slack webhooks or generic HTTP webhooks, with configurable repeat intervals
- **Silences** — suppress notifications using label matchers (exact or regex) during maintenance windows
- **Web dashboard** — built-in single-page UI for managing rules, viewing alerts, and configuring channels
- **Single binary** — UI assets and migrations are embedded into the Go binary

## Architecture

```
internal/
├── api/         HTTP server, REST endpoints, and web dashboard
├── evaluator/   Rule evaluation engine and alert state machine
├── notifier/    Notification dispatcher (Slack, webhook senders)
├── store/       Data persistence (SQLite)
└── model/       Data structures (rules, alerts, silences, channels)
```

The system uses **ClickHouse** as the monitored data source and **SQLite** for storing rules, alert states, silences, and notification channels.

## Getting Started

### Prerequisites

- Go 1.25+
- Access to a ClickHouse instance

### Configuration

Copy the example config and edit it:

```bash
cp config.example.yaml config.yaml
```

Key settings in `config.yaml`:

```yaml
listen_addr: ":8080"

clickhouse:
  dsn: "clickhouse://default:@localhost:9000/default"
  max_open_conns: 5

sqlite:
  path: "./alerting.db"

evaluation:
  default_interval: 60s   # how often rules are evaluated
  query_timeout: 30s       # max time per ClickHouse query
  max_concurrent: 10       # parallel rule evaluations

notifications:
  repeat_interval: 4h      # re-notify interval for firing alerts

log:
  level: info              # debug, info, warn, error
  format: json             # json or text
```

### Run directly

```bash
go build -o alerting-system .
./alerting-system --config config.yaml
```

### Run with Docker

```bash
docker build -t clickhouse-alerting-system .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/etc/alerting-system/config.yaml clickhouse-alerting-system
```

## API

All endpoints return JSON. Base path: `/api`

| Resource | Method | Path | Description |
|----------|--------|------|-------------|
| Rules | `GET` | `/api/rules` | List all rules |
| | `POST` | `/api/rules` | Create a rule |
| | `GET` | `/api/rules/{id}` | Get a rule |
| | `PUT` | `/api/rules/{id}` | Update a rule |
| | `DELETE` | `/api/rules/{id}` | Delete a rule |
| Channels | `GET` | `/api/channels` | List channels |
| | `POST` | `/api/channels` | Create a channel |
| | `GET` | `/api/channels/{id}` | Get a channel |
| | `PUT` | `/api/channels/{id}` | Update a channel |
| | `DELETE` | `/api/channels/{id}` | Delete a channel |
| | `POST` | `/api/channels/{id}/test` | Send test notification |
| Silences | `GET` | `/api/silences` | List silences |
| | `POST` | `/api/silences` | Create a silence |
| | `DELETE` | `/api/silences/{id}` | Delete a silence |
| Alerts | `GET` | `/api/alerts` | List current alerts |
| | `GET` | `/api/alerts/history` | List alert history |

## Web UI

The built-in dashboard is served at the root path (`/`) and provides tabs for managing alerts, rules, history, silences, and notification channels.

## Project Structure

```
├── main.go                 Entry point and server setup
├── config.go               Configuration loader
├── config.example.yaml     Config template
├── Dockerfile              Multi-stage Docker build
├── internal/
│   ├── api/                HTTP handlers and routing
│   ├── evaluator/          Rule evaluation and state transitions
│   ├── notifier/           Slack and webhook notification senders
│   ├── store/              SQLite store implementation
│   └── model/              Data models
├── ui/                     Web dashboard (HTML, JS, CSS)
├── migrations/             SQLite schema migrations
└── cmd/chquery/            ClickHouse query utility
```
