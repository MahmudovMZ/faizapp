# FaizApp

<p align="center">
  <img src="docs/assets/faizapp-logo-dark.png" alt="FaizApp logo" width="180">
</p>

<p align="center"><strong>Telegram-based field operations platform</strong></p>
<p align="center">Registration · Photo reports · Team control · Operational statistics</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/PostgreSQL-ready-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Telegram-bot-26A5E4?logo=telegram&logoColor=white" alt="Telegram">
  <img src="https://img.shields.io/badge/status-active%20development-F2C94C" alt="Active development">
</p>

---

## What is FaizApp?

FaizApp is an internal platform for managing the daily work of sales representatives, merchandisers, supervisors, dispatchers, and management teams.

The first version is intentionally focused: Telegram is the user interface, PostgreSQL is the source of truth, and human dispatchers remain responsible for visual decisions that require business judgment.

> **Automate counting and routine work. Keep important decisions auditable and human-controlled.**

## Product at a glance

| Area | Current state |
|---|---|
| Telegram polling and commands | ✅ Implemented |
| Environment configuration | ✅ Implemented |
| PostgreSQL and migrations | ✅ Implemented |
| User registration and approval | ✅ Implemented |
| Roles, work groups, territories | ✅ Implemented |
| Supervisor assignment | ✅ Implemented |
| Telegram photo intake | ✅ Implemented |
| SHA-256 photo hashing and exact-duplicate rejection | ✅ Implemented |
| Photos from unregistered senders | ✅ Implemented |
| Photo counting for sales representatives and supervisors | ✅ Implemented |
| Parallel photo workers and photo timeout | ✅ Implemented |
| Telegram album id stored (metadata only) | ✅ Implemented |
| Durable photo queue and retries | ✅ Implemented |
| Dispatcher verification | ⬜ Planned |
| Plans, reminders, reports | ⬜ Planned |
| VPS deployment and backups | ⬜ Planned |

## Current workflow

```text
Telegram user
     │
     ▼
Telegram Bot API
     │
     ▼
Long polling (main loop)
     │
     ├── text, commands, buttons ──► handled in order
     │
     └── photos ──► photo_jobs table ──► PHOTO_WORKERS goroutines
                                              │
Telegram transport ◄─────────────────────────┘
     │
     ├── Registration and approval
     ├── Role and group assignment
     └── Photo reception
             │
             ▼
        Service layer
             │
             ▼
       PostgreSQL repository
```

## What works today

### Registration and approval

Users register through Telegram by providing their full name and sharing their own Telegram contact. The application stores the request as pending and sends an approval workflow to the administrator.

```text
/start → Registration → Full name → Phone number → Admin decision
                                                    │
                                    ┌───────────────┴───────────────┐
                                    ▼                               ▼
                                  Approve                         Reject
```

### Roles and operational structure

The current role catalog contains:

- Sales Representative / Merchandiser;
- Supervisor;
- Dispatcher;
- Commercial Director;
- Senior Supervisor.

The **Senior Supervisor** role is already present as a structural role. Its special permissions will be implemented later together with the access-control layer.

Role names are defined once as constants in `internal/models` (`RoleSalesRep`, `RoleSupervisor`, …) and used everywhere instead of string literals.

### Work groups

```text
Horeca 1
├── Horeca 1
└── Horeca 2

Horeca 2
├── Horeca 3
└── Horeca 4

SM 1
├── SM 2
├── SM 3
├── SM 5
└── SM 6

SM 2
├── SM 1
└── SM 4
```

Supervisors can be assigned to a work group. Sales representatives follow the configured assignment flow for their work group, territory, and SR code.

### Photo intake

For every received photo the system:

1. stores a job in the `photo_jobs` table (sender, message ID, caption, Telegram `file_id`); the photo itself is not downloaded yet;
2. a free worker takes the job from the table;
3. downloads the highest-resolution photo variant (within `PHOTO_TIMEOUT`);
4. calculates a SHA-256 hash;
5. saves the photo record to PostgreSQL;
6. rejects exact byte duplicates through a unique index on the hash;
7. marks the job as done, or schedules a retry on error;
8. stores the Telegram album id (`media_group_id`, `NULL` for a single photo) as metadata.

```text
Telegram photo → photo_jobs → worker → Download → SHA-256 → PostgreSQL
```

**Unregistered senders.** A photo from a Telegram user who is not in the database is not dropped. The sender is created as a `pending` user (name taken from Telegram), the photo is saved, and the bot replies to the photo with a mention and a link `t.me/<bot>?start=registration`. Opening the link starts registration right away; the entered name and phone overwrite the pending record, and the administrator receives the usual approval request.

**Which photos are counted.** Every photo is stored, but only users with the **Sales Representative** or **Supervisor** role are counted. Other roles and pending users without a role just send photos. Photos sent before a role was assigned are counted once the role is set, because counting uses the user's current role.

**Replies.** A normal photo gets no reply; the result is written to the log. The bot replies to the photo only on a duplicate, a new sender (registration link) or when the job could not be stored.

**Durable queue.** Every photo becomes a row in `photo_jobs` before any processing, so it survives a bot restart. Statuses: `pending` → `processing` → `done`, or `dead` after 5 failed attempts. A worker locks the job for `PHOTO_TIMEOUT × 2`; if the bot stops in the middle, the lock expires and another worker takes the job again after the restart. A failed attempt is retried after 15 seconds, and the error text is kept in `last_error`. A duplicate photo finishes the job as `done`, because retrying will not change the answer.

**Concurrency.** Photos are processed by `PHOTO_WORKERS` goroutines (default 4), so a slow download does not block buttons, commands or registration. Two workers never take the same job (`FOR UPDATE SKIP LOCKED`). Replies to photos can therefore arrive out of order.

The caption is stored for future analysis. It is not interpreted yet.

## Photo verification roadmap

Already done: parallel workers, photo timeout, exact duplicates by SHA-256, album id stored, durable photo queue with retries. The next photo capabilities are planned in this order:

1. reply only on anomalies instead of confirming every photo;
2. later evaluate visually similar images with perceptual hashing;
3. send possible duplicates to the dispatcher for a decision;
4. store the verification result and notify the employee.

**Merch counting:** merch = number of counted photos ÷ a coefficient. Albums are **not** used to form before/after pairs: on a weak mobile connection one or more photos of an album may never finish uploading, so an album is not a reliable unit.

Note: Telegram strips EXIF metadata from compressed photos, so the shooting date is only available when a photo is sent as a file.

Planned dispatcher decisions:

| Result | Meaning |
|---|---|
| `ACCEPTED` | The submission is accepted and counted. |
| `ACCEPTED_WITH_COMMENT` | The submission is counted and a comment is recorded. |
| `REJECTED` | The submission is not counted and the reason is recorded. |

Automated duplicate detection will assist the dispatcher; it will not silently replace business judgment.

## Architecture

FaizApp is a modular monolith. The codebase is small by design and grows only when the next business capability requires it.

```text
faizapp/
├── cmd/bot/                         # Application entry point
├── internal/
│   ├── app/                         # Composition and startup
│   ├── config/                      # Environment loading and validation
│   ├── Database/                    # Connection pool and migrations
│   ├── models/                      # Domain models, roles and Telegram menus
│   ├── polling/                     # Update delivery and photo workers
│   ├── repository/postgres/         # PostgreSQL access
│   ├── service/                     # Business logic
│   ├── transport/telegram/          # Telegram handlers and state
│   └── middleware/                  # Cross-cutting checks
├── migrations/                      # Versioned database changes
├── docs/                            # Project documentation
├── .env.example                     # Safe configuration template
├── go.mod
└── README.md
```

### Dependency direction

```text
cmd → app → polling / transport
             │
             ▼
          services
             │
             ▼
        repositories
             │
             ▼
         PostgreSQL
```

Telegram-specific behavior stays in the transport layer. Reusable business rules belong in services. SQL and persistence details stay in repositories.

## Configuration

Runtime configuration is loaded from environment variables:

```env
TGBOTAPI_TOKEN=
BOT_MODE=polling
ADMIN_TELEGRAM_ID=

POSTGRES_HOST=
POSTGRES_PORT=
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=

# optional
PHOTO_TIMEOUT=60s
PHOTO_WORKERS=4
```

| Variable | Required | Meaning |
|---|---|---|
| `TGBOTAPI_TOKEN` | yes | Telegram bot token |
| `BOT_MODE` | yes | only `polling` is supported |
| `ADMIN_TELEGRAM_ID` | yes | Telegram ID that receives registration requests |
| `POSTGRES_*` | yes | PostgreSQL connection |
| `PHOTO_TIMEOUT` | no | time allowed for one photo, Go duration (`60s`, `2m`); default `60s` |
| `PHOTO_WORKERS` | no | number of parallel photo workers, `1`–`8`; default `4` |

Invalid values stop the bot at startup with a clear error. Other updates (text, buttons) have a fixed 5-second timeout. The PostgreSQL pool is fixed at 10 connections.

Use `.env.example` as a template. The real `.env` file and all credentials must remain local and ignored by Git.

## Local development

### Requirements

- Go 1.26 or newer;
- PostgreSQL;
- a Telegram bot token.

Migrations are applied automatically at startup.

### Run the project

```bash
go mod download
go test ./...
go run ./cmd/bot
```

### Check changes

```bash
make check
```

`make check` runs gofmt, `git diff --check`, `go mod verify`, `go vet`, `go test -race` and `go build`. Enable the same checks before every push with `make hooks`.

### Integration tests

Repository tests that need a real database are marked with the `integration` build tag and are skipped unless `TEST_DATABASE_URL` points to a **separate test database** (never production). The test database must have all migrations applied.

```bash
TEST_DATABASE_URL=postgres://user:pass@host:5432/faizapp_test?sslmode=disable \
  go test -tags integration ./internal/repository/postgres/
```

The tests create their own rows with random Telegram IDs and delete them afterwards.

## Database

PostgreSQL is the persistent source of truth. Schema changes are introduced only through versioned migrations.

Current and planned entities include users, roles, work groups, territories, SR codes, photos, plans, comments, verification results, and operational statistics.

Telegram handlers must not contain raw SQL.

## Roadmap

### Foundation — completed

- [x] Telegram bot initialization and long polling.
- [x] Environment configuration and validation.
- [x] PostgreSQL connection pool.
- [x] Versioned migrations.
- [x] User registration through Telegram contacts.
- [x] Administrator approval and rejection.
- [x] Role selection through inline keyboards.
- [x] Work groups and territories.
- [x] Sales representative and supervisor assignment workflows.

### Photo reporting — in progress

- [x] Receive Telegram photos.
- [x] Download photo files.
- [x] Store sender and message metadata.
- [x] Calculate SHA-256 hashes.
- [x] Reject exact duplicates.
- [x] Accept photos from unregistered senders and invite them to register.
- [x] Count photos for sales representatives and supervisors only.
- [x] Configurable photo timeout.
- [x] Bounded concurrent processing (photo workers).
- [x] Store the Telegram album id as metadata.
- [x] Durable queue with retries.
- [ ] Reply only on anomalies.
- [ ] Detect potential (non-identical) duplicates.
- [ ] Link photos to work/reporting events.

### Dispatcher verification — planned

- [ ] Accept a photo report.
- [ ] Accept it with a comment.
- [ ] Reject it with a reason.
- [ ] Notify the corresponding employee.
- [ ] Recalculate operational counters.

### Automation and reporting — planned

- [ ] Daily and monthly plans.
- [ ] Scheduled performance checks.
- [ ] Employee reminders.
- [ ] End-of-day summaries.
- [ ] Dispatcher statistics.
- [ ] Management reports and Excel-compatible exports.

### Operations — planned

- [ ] Docker Compose deployment on a VPS.
- [ ] Health check, structured logs, restart policy.
- [ ] Daily external PostgreSQL backups and restore check.
- [ ] Deploy, rollback and incident runbook.

### Future platform capabilities

- [ ] Web interface and personal dashboards.
- [ ] Trade-point auditing.
- [ ] Planogram management.
- [ ] Advanced analytics.
- [ ] AI-assisted photo analysis.

## Security rules

- Never commit `.env` files.
- Never commit Telegram tokens or database passwords.
- Never place production credentials in source code.
- Never log secrets.
- Enforce administrative authorization on the server side.
- Treat Telegram UI visibility as presentation, not authorization.

## Git workflow

Development changes are intended to flow through `develop` using focused feature branches and pull requests.

```text
feature branch → tests → pull request → CI → develop
```

Recommended checks before opening a pull request:

```bash
gofmt -w .
git diff --check
go test ./...
```

## Development principles

- Keep transport, business logic, persistence, and startup responsibilities separate.
- Prefer small, testable changes.
- Keep important operational actions auditable.
- Do not introduce infrastructure before it is needed.
- Preserve original records behind derived statistics.
- Use automation to assist people, not to hide business decisions.

## Current focus

```text
Reliable photo intake
    │
    ├── Workers and timeout   ✅
    ├── Album id stored       ✅
    ├── Durable queue and retries ✅
    └── Reply only on anomalies   ← next
```

After this block: dispatcher review, daily reporting, then VPS deployment for the pilot.

## Project status

**Active development**

The application foundation, user-management workflows, group assignments, the photo-reporting layer (including unregistered senders, role-based counting, parallel workers and a durable photo queue) are implemented. The next step in reliable photo intake is anomaly-only replies.

## License

Internal company project. No public license has been defined.
