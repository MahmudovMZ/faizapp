# FaizApp Architecture Overview

> **Status:** Early pilot / MVP  
> **Last architecture checkpoint:** Configuration stage completed  
> **Primary runtime mode:** Telegram Long Polling

## 1. Purpose

FaizApp is an internal field-operations system currently implemented as a Telegram bot.

The first development stages focus on creating a reliable foundation for:

- Telegram-based user interaction;
- field employee registration;
- photo-report processing;
- merchandizing accounting;
- dispatcher verification;
- operational statistics and reporting.

The current codebase is still intentionally small. Only components that are required by the current development stage should be implemented.

---

## 2. System Context

At the current stage, Telegram is the external interface of the application.

```text
Telegram
   │
   ▼
Telegram Bot API
   │
   ▼
FaizApp
   │
   ├── Configuration
   ├── Application startup
   ├── Polling
   └── Telegram handlers
```

PostgreSQL is the planned persistent storage layer and is the next technical milestone. Database configuration is already part of the application configuration, but the database connection itself is not implemented yet.

---

## 3. Architectural Style

FaizApp is currently designed as a **modular monolith**.

The application is kept in a single Go codebase, while responsibilities are separated into internal packages.

Current and planned responsibility boundaries:

```text
cmd
 │
 ▼
app
 │
 ├── config
 ├── polling
 ├── transport
 ├── service        [planned / not implemented yet]
 ├── repository     [planned / not implemented yet]
 └── PostgreSQL     [next milestone]
```

The project deliberately avoids premature microservices or distributed infrastructure.

The architecture should grow incrementally as actual business features are implemented.

---

## 4. Core Components

### 4.1 Application Entry Point

Path:

```text
cmd/bot/main.go
```

Responsibilities:

1. load application configuration;
2. handle configuration-loading failure;
3. pass the completed configuration into the application package.

Current startup flow:

```text
main()
  │
  ▼
config.Load()
  │
  ▼
app.Run(cfg)
```

The entry point does not read environment variables directly.

---

### 4.2 Configuration

Path:

```text
internal/config/config.go
```

The configuration package owns environment configuration loading and validation.

The current configuration model is:

```text
Config
├── Bot BotConfig
└── DB  DBConfig
```

`BotConfig` contains:

```text
Token
BotMode
```

`DBConfig` contains:

```text
Username
Password
DBName
Host
Port
```

Current configuration flow:

```text
.env
  │
  ▼
godotenv.Load()
  │
  ├── NewBotConfig()
  │
  └── NewDBConfig()
          │
          ▼
       Config
```

The package validates required values before returning the final `Config`.

The currently supported bot mode is:

```text
polling
```

PostgreSQL port configuration is parsed as an integer and validated against the valid TCP port range.

### Required environment variables

```text
TGBOTAPI_TOKEN
BOT_MODE

POSTGRES_HOST
POSTGRES_PORT
POSTGRES_USER
POSTGRES_PASSWORD
POSTGRES_DB
```

A safe template is stored in:

```text
.env.example
```

Real credentials must remain in `.env` and must not be committed to the repository.

---

### 4.3 Application Composition

Path:

```text
internal/app/app.go
```

The application package currently:

1. receives `config.Config`;
2. creates the Telegram Bot API client;
3. configures Telegram debug mode;
4. selects the bot runtime mode;
5. starts polling.

Current flow:

```text
Config
  │
  ▼
app.Run()
  │
  ▼
Telegram Bot API client
  │
  ▼
polling.StartPolling()
```

A PostgreSQL connection is explicitly marked as the next application dependency.

Current code still contains:

```text
// TODO: Db connection
```

Database initialization should therefore be added to application composition rather than scattered through Telegram handlers.

---

### 4.4 Telegram Polling

Path:

```text
internal/polling/polling.go
```

Polling currently:

1. creates a Telegram update configuration;
2. uses a 30-second polling timeout;
3. receives updates through `GetUpdatesChan`;
4. forwards each update to the Telegram transport handler.

Current flow:

```text
Telegram API
    │
    ▼
GetUpdatesChan
    │
    ▼
for update := range updates
    │
    ▼
telegram.BotHandler(...)
```

The polling package is responsible for update delivery, not business logic.

---

### 4.5 Telegram Transport

Path:

```text
internal/transport/telegram/handlers.go
```

The Telegram handler currently:

- ignores updates without `Message`;
- reads the chat ID and message text;
- reacts to `/start`;
- sends simple response messages.

Current minimal `/start` flow:

```text
Telegram message
      │
      ▼
BotHandler
      │
      ▼
text == "/start"
      │
      ▼
send response
```

The current handler also contains temporary in-memory global state:

```text
botState
userState
userData
bot
```

These structures belong to the prototype stage and should not be treated as the final persistence/state architecture.

Persistent user identity and business data are planned to move to PostgreSQL.

---

### 4.6 Service Layer

Status:

```text
Planned
```

The service layer is intended to contain business logic such as:

- registration;
- photo processing;
- merch accounting;
- verification;
- statistics;
- notifications;
- scheduling.

Telegram-specific parsing should remain in the transport layer, while reusable business rules should move into services as they are implemented.

No unnecessary service abstraction should be created before the corresponding business feature exists.

---

### 4.7 Repository Layer

Status:

```text
Planned
```

The repository layer will isolate persistent-storage operations from business logic.

Expected direction:

```text
Service
  │
  ▼
Repository
  │
  ▼
PostgreSQL
```

Telegram handlers should not contain raw SQL.

Business services should not depend on PostgreSQL query details.

---

### 4.8 PostgreSQL

Status:

```text
Configuration ready
Connection not implemented
```

Database environment values are already loaded through `DBConfig`.

The next technical milestone is to establish:

```text
Config.DB
   │
   ▼
PostgreSQL connection pool
   │
   ▼
Ping / health verification
   │
   ▼
Application runtime
   │
   ▼
Graceful shutdown
```

The full business schema should not be implemented as part of the connection-foundation task unless explicitly required.

---

### 4.9 Scheduler

Status:

```text
Planned
```

Scheduling is a later application component for:

- plan-progress checks;
- reminders;
- midday reporting;
- end-of-day summaries.

It should be introduced only after the required persistent business data and statistics exist.

---

## 5. Dependency Direction

The intended dependency direction is:

```text
cmd
 │
 ▼
app
 │
 ├──────────────► config
 │
 ├──────────────► polling
 │                  │
 │                  ▼
 │               transport
 │
 ├──────────────► service          [future]
 │                  │
 │                  ▼
 └──────────────► repository       [future]
                    │
                    ▼
                PostgreSQL
```

General rule:

> Outer delivery/infrastructure components may call application/business components, but business logic should not depend on Telegram-specific behavior.

Configuration is constructed before application startup and passed inward explicitly.

---

## 6. Main Application Flows

### 6.1 Application Startup

Current implemented startup:

```text
Process starts
    │
    ▼
config.Load()
    │
    ├── load .env
    ├── validate Telegram config
    └── validate PostgreSQL config
    │
    ▼
Config
    │
    ▼
app.Run(cfg)
    │
    ▼
NewBotAPI(cfg.Bot.Token)
    │
    ▼
BOT_MODE == polling
    │
    ▼
StartPolling(bot)
```

If configuration loading fails, startup terminates in `main`.

---

### 6.2 User Registration

Status:

```text
Planned
```

Intended high-level flow:

```text
/start
  │
  ▼
identify Telegram user
  │
  ▼
lookup user in PostgreSQL
  │
  ├── not registered
  │      │
  │      ▼
  │   registration
  │
  └── registered
         │
         ▼
      main menu
```

The current `/start` implementation is only a temporary greeting and does not yet perform persistence-based registration checks.

---

### 6.3 Photo Processing

Status:

```text
Planned
```

High-level intended flow:

```text
Telegram photo
    │
    ▼
Telegram transport
    │
    ▼
identify sender
    │
    ▼
photo processing
    │
    ├── metadata
    ├── SHA-256
    └── EXIF when available
    │
    ▼
persistent photo record
    │
    ▼
work / merch accounting
```

Detailed photo-processing rules belong in dedicated business documentation rather than in this architecture overview.

---

### 6.4 Dispatcher Verification

Status:

```text
Planned
```

The dispatcher remains responsible for visual verification during the pilot.

Expected outcomes:

```text
Accepted
Accepted with comment
Rejected
```

FaizApp should persist the decision and update business statistics accordingly.

Visual quality assessment is intentionally outside the current automated architecture.

---

### 6.5 Statistics

Status:

```text
Planned
```

Statistics will be derived from persistent operational data.

Expected direction:

```text
photo/report records
       │
       ▼
verification
       │
       ▼
daily statistics
       │
       ▼
monthly statistics
       │
       ▼
reports
```

Business counters should not become an unrecoverable source of truth when the underlying records can be retained.

---

## 7. Data Flow

### Current implemented data flow

```text
.env
  │
  ▼
Config
  │
  ▼
Application
  │
  ▼
Telegram Bot API
  │
  ▼
Polling
  │
  ▼
Telegram Handler
  │
  ▼
Telegram response
```

### Planned persistent data flow

```text
Telegram Update
      │
      ▼
Transport
      │
      ▼
Service
      │
      ▼
Repository
      │
      ▼
PostgreSQL
```

---

## 8. Security Boundaries

Current security requirements:

- Telegram bot token must come from environment configuration;
- PostgreSQL credentials must come from environment configuration;
- `.env` must not be committed;
- `.env.example` must contain placeholders only;
- secrets must not be logged;
- administrative functionality must later be enforced by server-side authorization.

Telegram UI visibility alone must not be considered authorization.

---

## 9. Current Implementation Status

### Completed foundation

- [x] Go application entry point
- [x] `.env` loading
- [x] typed application configuration
- [x] Telegram configuration validation
- [x] PostgreSQL configuration validation
- [x] bot mode validation
- [x] Telegram Bot API client initialization
- [x] Long Polling startup
- [x] update forwarding to Telegram handler
- [x] minimal `/start` response
- [x] `.env.example`

### Not implemented yet

- [ ] PostgreSQL connection
- [ ] database health check
- [ ] graceful database shutdown
- [ ] migrations
- [ ] persistent users
- [ ] registration
- [ ] role authorization
- [ ] persistent conversational state
- [ ] photo processing
- [ ] duplicate detection
- [ ] merch accounting
- [ ] dispatcher verification
- [ ] plans
- [ ] statistics
- [ ] scheduler
- [ ] automated reports

---

## 10. Planned Architecture

The target direction is:

```text
Telegram
   │
   ▼
Transport
   │
   ▼
Services
   │
   ├── Registration
   ├── Photo Processing
   ├── Merch Accounting
   ├── Verification
   ├── Statistics
   └── Notifications
   │
   ▼
Repositories
   │
   ▼
PostgreSQL
```

Supporting application components may later include:

```text
Scheduler
Reporting
Excel export
```

These should be introduced only when their dependent business features exist.

---

## 11. Out of Scope for the Current Stage

The following are intentionally not part of the current configuration/startup stage:

- microservices;
- AI visual analysis;
- automated merch quality judgment;
- web frontend;
- Telegram Mini App;
- advanced analytics;
- complete reporting;
- object storage;
- distributed task processing;
- premature scaling infrastructure.

The next step is database foundation, not expansion into these areas.

---

## 12. Related Documentation

Current repository documentation:

```text
README.md
docs/privacy-policy.md
docs/architecture/overview.md
```

Recommended future documentation:

```text
docs/
├── architecture/
│   ├── overview.md
│   ├── project-structure.md
│   └── data-flow.md
│
├── database/
│   ├── schema.md
│   └── entities.md
│
├── business/
│   ├── roles.md
│   ├── registration.md
│   ├── photo-lifecycle.md
│   ├── merch-accounting.md
│   └── verification.md
│
└── decisions/
    └── README.md
```

---

## 13. Next Architecture Checkpoint

The next documentation update should happen after the PostgreSQL foundation is completed.

At that point this document should be updated with:

- selected PostgreSQL driver/pool;
- database initialization ownership;
- connection lifecycle;
- health-check behavior;
- shutdown behavior;
- updated application startup flow.
