# FaizApp Architecture Overview

> **Status:** Early pilot / MVP
> **Last architecture checkpoint:** registration link sent once (PR #77)
> **Primary runtime mode:** Telegram Long Polling

## 1. Purpose

FaizApp is an internal field-operations system implemented as a Telegram bot backed by PostgreSQL.

It currently covers:

- Telegram-based registration with administrator approval;
- roles, work groups, territories and SR-code assignment;
- photo-report intake with SHA-256 duplicate protection;
- photos from unregistered senders and an invitation to register;
- photo counting for sales representatives and supervisors;
- a durable photo queue in PostgreSQL with retries.

Planned next: daily statistics, dispatcher verification, merch accounting, scheduled reports and VPS deployment.

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

Creates the repository, `UserService`, `PhotoService`, `PhotoJobService` (job lock = `PhotoTimeout × 2`) and the Telegram Bot API client, then starts polling with the admin ID, photo timeout and number of photo workers.

### 4.5 Polling and photo workers — `internal/polling`

```text
GetUpdatesChan
     │
     ▼
main loop ──── text, commands, callbacks ──► handler.BotHandler (in order)
     │
     └── photo ──► handler.EnqueuePhoto ──► photo_jobs (status pending)
                                                   │
                                   ┌───────────────┼───────────────┐
                                   ▼               ▼               ▼
                               worker 1        worker 2 …      worker N   (N = PHOTO_WORKERS)
                                   │
                                   ▼
             NextJob ──► processPhotoJob ──► FinishJob   (handler.RunPhotoWorker)
```

- The main loop only stores the job (5-second timeout) and moves on; it never waits for a download.
- Workers start once at startup. Each one loops: take a job, process it within `PHOTO_TIMEOUT`, record the result. On an empty queue or a database error it waits 3 seconds.
- Each step has its own context, so a slow download cannot leave `FinishJob` without time.
- A slow photo download does not block buttons, commands or registration.
- Replies to photos can arrive out of order.
- Each worker logs `[HANDLER] starting worker N`, every job it receives and every job it finishes.

### 4.6 Telegram transport — `internal/transport/telegram`

- `BotHandler` handles text, commands and callbacks with a 5-second timeout. Photos never reach it; they go through the queue.
- Registration conversation state (`botState`, `userState`, `userData`) is kept in memory and protected by a `sync.RWMutex`. It is lost on restart.
- Admin callbacks (`approve`, `reject`, `role`, `group`, `territory`, `code`) are parsed centrally and accepted only from `ADMIN_TELEGRAM_ID`.
- `EnqueuePhoto`: ignores a message without a sender or without a photo; otherwise builds a `PhotoJob` from the message (largest size `file_id`, sender name, caption, album id, date) and stores it; if storing fails, replies to the photo with «Не удалось обработать фотографию…».
- `processPhotoJob`: downloads by `file_id`, calls `PhotoService.SavePhoto`, then:
  - duplicate → reply to the photo with «Это фото уже было отправлено ранее…»;
  - after a saved photo or a duplicate → `sendRegistrationLinkOnce`: if `PhotoService.ClaimRegistrationLink` returns true, reply to the photo with a mention and a `?start=registration` link (a failure is only logged);
  - otherwise → no reply, only a log line.
- `/start registration` for a pending user without a phone starts registration immediately.

### 4.7 Services — `internal/service`

- `UserService`: registration, approval, role and assignment workflows. `CreateUser` first tries `UpdateUserContact` for an existing pending user and falls back to inserting a new one.
- `PhotoService.SavePhoto`: finds or creates the sender (`CreatePendingUser`), stores the photo, returns `SavePhotoResult{Count, Counted, NewSender}`. `Counted` is true only for `RoleSalesRep` and `RoleSupervisor`. On a unique-hash error it checks `PhotoExistsForMessage`: the same Telegram user and message means an earlier attempt already saved it, so it is a success, not a duplicate. `NewSender` is no longer used by the handler.
- `PhotoService.ClaimRegistrationLink`: wraps `MarkRegistrationLinkSent`; true means this call should send the link.
- `PhotoJobService`: `EnqueueJob`, `NextJob` (claims a job with the configured lock) and `FinishJob`. `FinishJob` marks a job `done` on success or on a duplicate, `dead` when `attempts` reached 5, otherwise schedules a retry in 15 seconds. The store interface `PhotoJobStore` is declared in the service.

### 4.8 Repository — `internal/repository/postgres`

All SQL lives here behind the `Repository` interface. Notable methods:

| Method | Behavior |
|---|---|
| `CreatePendingUser` | single `INSERT … ON CONFLICT (tg_id) DO UPDATE … RETURNING`; never changes an existing user |
| `UpdateUserContact` | updates name and phone only for `status = 'pending'`; returns `pgx.ErrNoRows` otherwise |
| `CreatePhoto` | maps the unique-hash violation to `ErrDuplicatePhoto` |
| `CountPhotosByDate` | counts only photos of users with the sales-rep or supervisor role |
| `PhotoExistsForMessage` | `SELECT EXISTS` by Telegram user and message ID |
| `MarkRegistrationLinkSent` | one `UPDATE` that sets `registration_link_sent_at` only for a pending user without a phone whose mark is empty; returns whether a row changed |
| `EnqueuePhotoJob` | `INSERT … ON CONFLICT DO NOTHING`, so a repeated Telegram update does not create a second job |
| `ClaimPhotoJob` | one `UPDATE` with a `FOR UPDATE SKIP LOCKED` subquery: takes a `pending` job whose `next_attempt_at` has come, or a `processing` job whose lock expired; sets `processing`, `locked_until`, `attempts + 1`; returns `nil, nil` on an empty queue |
| `MarkPhotoJobDone` / `MarkPhotoJobDead` / `RetryPhotoJob` | clear the lock; dead and retry also store `last_error`; retry returns the job to `pending` with a new `next_attempt_at` |

Multi-step changes (role + status + assignment) run in a transaction.

---

## 5. Data Model (current)

```text
users ─────────┬──< photos            (sha256_hash unique)
  │            └──< sr_assignments >── sr_codes ──> territories ──> work_groups
  │                                                                    │
  └──────────── supervisor of ────────────────────────────────────────┘
brand_categories
photo_jobs                            (queue, unique chat_id + telegram_message_id)
```

- `users.status`: `pending` / `approved` / `rejected`.
- `users.role`: one of the role constants, `NULL` until assigned.
- `users.registration_link_sent_at`: when the registration link was sent, `NULL` until then (migration 22).
- `photos` stores `user_id`, Telegram user and message IDs, caption, received time, SHA-256 and `media_group_id` (Telegram album id, `NULL` for a single photo, metadata only). The Telegram `file_id` is not stored in `photos`.
- `photo_jobs` keeps what a worker needs to process a photo: chat and message IDs, sender, `file_id`, caption, album id, received time, plus `status` (`pending` / `processing` / `done` / `dead`), `attempts`, `next_attempt_at`, `locked_until`, `last_error`. Finished jobs stay in the table; cleanup is not implemented yet.

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
photo ─► sender not found ─► CreatePendingUser ─► saved or duplicate ─► link not sent yet? ─► mention + ?start=registration
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
- If Telegram fails to deliver the one registration-link message, the link is not sent again.
- Bot replies in `processPhotoJob` are not covered by tests (needs a fake Telegram sender).
- Telegram removes EXIF from compressed photos, so the shooting date is not available.

---

## 10. Next Steps

1. Daily statistics with merch accounting (photo count ÷ coefficient; albums are not used).
2. VPS deployment with Docker Compose, backups and monitoring; end-to-end acceptance.
3. Dispatcher verification (accepted / accepted with comment / rejected); whether it is needed for the pilot is still open.
4. Photos sent as files and other media.
5. Monthly reports, reminders.

---

## 11. Related Documentation

```text
README.md
docs/architecture/overview.md
docs/privacy-policy.md
```

The GitHub Wiki contains the current status, the roadmap and the configuration reference.
