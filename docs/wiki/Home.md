# FaizApp Wiki

FaizApp is an internal field-operations platform currently implemented as a Telegram bot and developed as a modular Go monolith.

The project is being built incrementally. Documentation must distinguish between functionality that is already implemented and functionality that is only planned.

## Current Status

**Stage:** early pilot / MVP

Completed foundation:

- Go application entry point;
- centralized `.env` configuration loading;
- typed Telegram and PostgreSQL configuration;
- configuration validation;
- Telegram Bot API initialization;
- Long Polling runtime;
- basic Telegram update handling;
- minimal `/start` flow;
- architecture documentation foundation.

Next technical milestone:

- PostgreSQL connection pool;
- connection health check;
- lifecycle ownership;
- graceful shutdown;
- migrations after the connection foundation is stable.

## Architecture

FaizApp follows a modular-monolith direction.

```text
Telegram
   │
   ▼
Transport
   │
   ▼
Services
   │
   ▼
Repositories
   │
   ▼
PostgreSQL
```

Current runtime path:

```text
cmd/bot/main.go
      │
      ▼
config.Load()
      │
      ▼
app.Run(cfg)
      │
      ▼
Telegram Bot API
      │
      ▼
Long Polling
      │
      ▼
Telegram Handler
```

Detailed architecture documentation:

- [Architecture Overview](../architecture/overview.md)

## Main Business Areas

The planned business scope includes:

- employee registration;
- TP / territory assignment;
- photo-report processing;
- duplicate-photo control;
- merch accounting;
- dispatcher verification;
- comments and corrections;
- plans and progress tracking;
- statistics;
- scheduled notifications and reports.

Business rules should be documented separately from infrastructure and transport details.

## Development Workflow

```text
implement
   ↓
make check
   ↓
commit
   ↓
push
   ↓
GitHub Actions CI
   ↓
review / merge
```

## Local Quality Gate

Run all fast local checks:

```bash
make check
```

Enable the repository pre-push hook once:

```bash
make hooks
```

After that, Git automatically runs `make check` before every push.

## CI Quality Gate

GitHub Actions validates every push and every pull request targeting `main`.

The baseline checks are:

- Go formatting with `gofmt`;
- whitespace errors with `git diff --check`;
- Go module integrity with `go mod verify`;
- accidental tracking of `.env`;
- static analysis with `go vet`;
- tests with the race detector;
- test coverage execution;
- full project build;
- known Go vulnerability scanning with `govulncheck`.

Once branch protection is enabled, required CI jobs should block merges to `main`.

## Security Rules

- Never commit `.env`.
- Never commit Telegram tokens or database credentials.
- Keep `.env.example` placeholder-only.
- Do not log secrets.
- Authorization must be enforced server-side.
- Persistent business state must move to PostgreSQL when the corresponding feature is implemented.

## Documentation Map

```text
docs/
├── architecture/
│   └── overview.md
├── wiki/
│   └── Home.md
└── privacy-policy.md
```

Planned:

```text
docs/
├── architecture/
│   ├── project-structure.md
│   └── data-flow.md
├── database/
│   ├── schema.md
│   └── entities.md
├── business/
│   ├── roles.md
│   ├── registration.md
│   ├── photo-lifecycle.md
│   ├── merch-accounting.md
│   └── verification.md
└── decisions/
    └── README.md
```

## Roadmap

### Foundation

- [x] Configuration
- [x] Telegram startup
- [x] Long Polling
- [ ] PostgreSQL foundation
- [ ] Migrations

### Core Business Flow

- [ ] Registration
- [ ] Roles and TP assignment
- [ ] Photo ingestion
- [ ] Duplicate detection
- [ ] Merch accounting
- [ ] Dispatcher verification
- [ ] Corrections

### Operational Features

- [ ] Plans
- [ ] Statistics
- [ ] Scheduled notifications
- [ ] Reports

### Later Stages

- [ ] Advanced analytics
- [ ] Optional AI-assisted photo analysis
- [ ] Additional interfaces only when justified by real requirements

## Documentation Rule

The repository code and the latest approved project decisions are the source of truth.

Documentation should be updated after architecture-relevant milestones so that planned functionality is never presented as already implemented.
