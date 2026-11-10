# FaizApp Architecture Overview

> **Status:** Early pilot / MVP
> **Last architecture checkpoint:** album id stored (PR #69)
> **Primary runtime mode:** Telegram Long Polling

## 1. Purpose

FaizApp is an internal field-operations system implemented as a Telegram bot backed by PostgreSQL.

It currently covers:

- Telegram-based registration with administrator approval;
- roles, work groups, territories and SR-code assignment;
- photo-report intake with SHA-256 duplicate protection;
- photos from unregistered senders and an invitation to register;
- photo counting for sales representatives and supervisors.

Planned next: anomaly-only replies, a durable photo queue, dispatcher verification, merch accounting, scheduled reports and VPS deployment.

Only components required by the current stage are implemented.

---

## 2. System Context

```text
Telegram users (sales reps, supervisors, dispatchers, admin)
        │
        ▼
Telegram Bot API
        │  long polling
        ▼
FaizApp (single Go process)
        │
        ▼
PostgreSQL
```

The bot downloads photos from Telegram servers, not from users' phones. Download speed depends on the network of the machine running the bot.

---

## 3. Architectural Style

FaizApp is a **modular monolith**: one Go codebase with responsibilities separated into internal packages.

```text
cmd/bot
  │
  ▼
config ──► Database (migrations, pool) ──► app
                                            │
                                            ▼
                                         polling
                                            │
                                            ▼
                                   transport/telegram
                                            │
                                            ▼
                                         service
                                            │
                                            ▼
                                   repository/postgres
                                            │
                                            ▼
                                        PostgreSQL
```

No microservices, message brokers or distributed infrastructure are used. The project grows only when a business feature needs it.

---

## 4. Core Components

### 4.1 Entry point — `cmd/bot/main.go`

1. `config.Load()` reads and validates `.env`.
2. `Database.RunMigrations()` applies pending migrations (golang-migrate).
3. `Database.NewPool()` creates the PostgreSQL pool and pings it.
4. `app.Run()` starts the bot.

Any failure in steps 1–3 stops the process with a clear log message.

### 4.2 Configuration — `internal/config`

```text
Config
├── Bot BotConfig
│   ├── Token          TGBOTAPI_TOKEN      required
│   ├── BotMode        BOT_MODE            required, only "polling"
│   ├── AdminTGID      ADMIN_TELEGRAM_ID   required, positive integer
│   ├── PhotoTimeout   PHOTO_TIMEOUT       optional, Go duration > 0, default 60s
│   └── PhotoWorkers   PHOTO_WORKERS       optional, 1–8, default 4
└── DB DBConfig
    └── POSTGRES_HOST / PORT / USER / PASSWORD / DB   required
```

Validation happens before the bot starts. Real values stay in `.env`; `.env.example` holds placeholders.

### 4.3 Database — `internal/Database`

- `BuildDSN` builds a plain PostgreSQL URL. It is shared with migrations, so it must not contain pgx-only parameters (golang-migrate rejects `pool_max_conns`).
- `NewPool` parses the DSN with `pgxpool.ParseConfig` and sets `MaxConns = 10`, so the pool size is the same on every machine (the pgx default depends on the CPU count).
- Migrations live in `migrations/` as numbered `up`/`down` pairs.

### 4.4 Application composition — `internal/app`

Creates the repository, `UserService`, `PhotoService` and the Telegram Bot API client, then starts polling with the admin ID, photo timeout and number of photo workers.

### 4.5 Polling and photo workers — `internal/polling`

```text
GetUpdatesChan
     │
     ▼
main loop ──── text, commands, callbacks ──► handler.BotHandler (in order)
     │
     └── photo ──► photos channel (buffer 100)
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
          worker 1    worker 2 …  worker N   (N = PHOTO_WORKERS)
              │
              ▼
       handler.BotHandler
```

- Workers start once at startup and wait on the channel.
- A slow photo download no longer blocks buttons, commands or registration.
- Replies to photos can arrive out of order.
- Each worker logs `[WORKER N] started` and every photo it takes.

### 4.6 Telegram transport — `internal/transport/telegram`

- `BotHandler` picks a timeout per update: `PhotoTimeout` for photos, 5 seconds for everything else.
- Registration conversation state (`botState`, `userState`, `userData`) is kept in memory and protected by a `sync.RWMutex`. It is lost on restart.
- Admin callbacks (`approve`, `reject`, `role`, `group`, `territory`, `code`) are parsed centrally and accepted only from `ADMIN_TELEGRAM_ID`.
- Photo flow: download the largest size, call `PhotoService.SavePhoto`, then:
  - new sender → reply to the photo with a mention and a `?start=registration` link;
  - counted role → `Received photos: N` (temporary confirmation);
  - other roles / no role → no reply.
- `/start registration` for a pending user without a phone starts registration immediately.

### 4.7 Services — `internal/service`

- `UserService`: registration, approval, role and assignment workflows. `CreateUser` first tries `UpdateUserContact` for an existing pending user and falls back to inserting a new one.
- `PhotoService.SavePhoto`: finds or creates the sender (`CreatePendingUser`), stores the photo, returns `SavePhotoResult{Count, Counted, NewSender}`. `Counted` is true only for `RoleSalesRep` and `RoleSupervisor`.

### 4.8 Repository — `internal/repository/postgres`

All SQL lives here behind the `Repository` interface. Notable methods:

| Method | Behavior |
|---|---|
| `CreatePendingUser` | single `INSERT … ON CONFLICT (tg_id) DO UPDATE … RETURNING`; never changes an existing user |
| `UpdateUserContact` | updates name and phone only for `status = 'pending'`; returns `pgx.ErrNoRows` otherwise |
| `CreatePhoto` | maps the unique-hash violation to `ErrDuplicatePhoto` |
| `CountPhotosByDate` | counts only photos of users with the sales-rep or supervisor role |

Multi-step changes (role + status + assignment) run in a transaction.

---

## 5. Data Model (current)

```text
users ─────────┬──< photos            (sha256_hash unique)
  │            └──< sr_assignments >── sr_codes ──> territories ──> work_groups
  │                                                                    │
  └──────────── supervisor of ────────────────────────────────────────┘
brand_categories
```

- `users.status`: `pending` / `approved` / `rejected`.
- `users.role`: one of the role constants, `NULL` until assigned.
- `photos` stores `user_id`, Telegram user and message IDs, caption, received time, SHA-256 and `media_group_id` (Telegram album id, `NULL` for a single photo, metadata only). The Telegram `file_id` is intentionally not stored.

---

## 6. Main Flows

### 6.1 Registration

```text
/start ─► not in DB ─► "Регистрация" ─► full name ─► own contact ─► pending ─► admin request
                                                                               │
                                                     approve + role (+ group/territory/code) / reject
```

### 6.2 Photo from an unregistered sender

```text
photo ─► sender not found ─► CreatePendingUser ─► photo saved ─► mention + ?start=registration
          │
          └─ link ─► /start registration ─► pending without phone ─► registration starts at once
                                                    │
                                          UpdateUserContact ─► admin request
```

### 6.3 Photo counting

```text
photo saved ─► role is sales rep or supervisor? ─ yes ─► counted
                                               └ no  ─► stored, not counted, no reply
```

---

## 7. Security Boundaries

- Token and database credentials come only from the environment; `.env` is never committed (CI checks this).
- Secrets must not be logged.
- Admin actions are authorized server-side by comparing the callback sender with `ADMIN_TELEGRAM_ID`.
- Only the user's own Telegram contact is accepted at registration.
- Integration tests run only against a separate database set in `TEST_DATABASE_URL`.

---

## 8. Quality Gates

- Local: `make check` (gofmt, `git diff --check`, `go mod verify`, `go vet`, `go test -race`, `go build`), also run by the `pre-push` hook (`make hooks`).
- CI (GitHub Actions): the same checks plus `govulncheck`.
- `main` accepts pull requests only from `develop`.

---

## 9. Known Limitations

- Conversation state is in memory: a restart in the middle of registration loses the step.
- Photos waiting in the channel are lost if the process stops; a durable queue is planned.
- A new sender who sends several photos at once may be mentioned more than once.
- If the very first photo of a new sender is a duplicate, the registration link is not sent.
- Telegram removes EXIF from compressed photos, so the shooting date is not available.

---

## 10. Next Steps

1. Reply only on anomalies.
2. Durable photo queue in PostgreSQL with leases and retries.
3. Photos sent as files and other media.
4. Dispatcher verification (accepted / accepted with comment / rejected).
5. Merch accounting (photo count ÷ coefficient; albums are not used), daily and monthly reports, reminders.
6. VPS deployment with Docker Compose, backups and monitoring.

---

## 11. Related Documentation

```text
README.md
docs/architecture/overview.md
docs/privacy-policy.md
```

The GitHub Wiki contains the current status, the roadmap and the configuration reference.
