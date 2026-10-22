# FaizApp

<p align="center">
  <img src="docs/assets/faizapp-logo-dark.png" alt="FaizApp logo" width="180">
</p>

<p align="center"><strong>Telegram-based field operations platform</strong></p>
<p align="center">Registration · Photo reports · Team control · Operational statistics</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" alt="Go">
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
| SHA-256 photo hashing | ✅ Implemented |
| Photo counting | ✅ Implemented |
| Album buffering and worker pool | 🟡 Planned |
| Duplicate-review workflow | 🟡 Planned |
| Dispatcher verification | ⬜ Planned |
| Plans, reminders, reports | ⬜ Planned |

## Current workflow

```text
Telegram user
     │
     ▼
Telegram Bot API
     │
     ▼
Long polling
     │
     ▼
Telegram transport
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

The first photo-reporting foundation is implemented. For every received photo the system can:

1. identify the Telegram sender;
2. read the Telegram message ID and caption;
3. download the highest-resolution photo variant;
4. calculate a SHA-256 hash;
5. save metadata to PostgreSQL;
6. return the user's current photo count.

```text
Telegram photo → Download → SHA-256 → PostgreSQL → User photo count
```

The caption is stored for future analysis. It is not interpreted yet.

## Photo verification roadmap

The current implementation stores and counts photos. The next photo capabilities are planned in this order:

1. collect photos from Telegram albums through `MediaGroupID`;
2. add a bounded queue and worker pool for concurrent downloads;
3. detect exact duplicates using stored SHA-256 hashes;
4. later evaluate visually similar images with perceptual hashing;
5. send possible duplicates to the dispatcher for a decision;
6. store the verification result and notify the employee.

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
│   ├── models/                      # Domain models and Telegram menus
│   ├── polling/                     # Telegram update delivery
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

POSTGRES_HOST=
POSTGRES_PORT=
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=
```

Use `.env.example` as a template. The real `.env` file and all credentials must remain local and ignored by Git.

## Local development

### Requirements

- Go 1.24 or newer;
- PostgreSQL;
- a Telegram bot token;
- applied database migrations.

### Run the project

```bash
go mod download
go test ./...
go run ./cmd/bot
```

### Check changes

```bash
gofmt -w .
git diff --check
go test ./...
```

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
- [x] Count photos by user.
- [ ] Process albums as a single submission.
- [ ] Add bounded concurrent processing.
- [ ] Detect exact and potential duplicates.
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
Photo intake
    │
    ├── Albums
    ├── Queue and workers
    ├── Duplicate detection
    ├── Dispatcher review
    └── Daily reporting
```

After this block is complete, the project can move to employee plans, scheduled reminders, and full operational statistics.

## Project status

**Active development**

The application foundation, user-management workflows, group assignments, and the first photo-reporting layer are implemented. The next major milestone is a reliable photo workflow with album handling, duplicate review, dispatcher verification, and daily reporting.

## License

Internal company project. No public license has been defined.
