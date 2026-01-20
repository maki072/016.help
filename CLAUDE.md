# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a multi-tenant helpdesk system built in Go with Telegram bot integration and Google Calendar support. The application is designed for low-load scenarios and uses a simple architecture without complex dependencies.

**Tech Stack:**
- Go 1.21+
- PostgreSQL 12+
- Chi router for HTTP handling
- Telegram Bot API for customer communication
- Google OAuth2 for Calendar integration

**Deployment Model:**
- Development on Windows
- Production deployment on Debian 12/13
- Code pushed to Git, pulled and executed on server

## Development Commands

### Running the Application

```bash
# Download dependencies
go mod download

# Run the application (development)
go run main.go

# Build for production
go build -o helpdesk main.go
```

The application will be available at `http://localhost:8080` by default.

### Database Setup

The application expects PostgreSQL to be running. Use Docker Compose for local development:

```bash
docker-compose up -d
```

Migrations run automatically on application startup from [migrations/001_init.sql](migrations/001_init.sql).

### Environment Configuration

Copy [env.example](env.example) to `.env` and configure:
- Database connection parameters
- `TELEGRAM_BOT_TOKEN` - required for Telegram bot functionality
- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` - optional for Calendar integration
- `SESSION_SECRET` - random string for session security

### Password Generation

Use the password generation utility when creating user accounts:

```bash
go run scripts/gen_password.go <password>
```

### Deployment

For automated deployment to Debian 12/13, use:

```bash
sudo bash scripts/deploy.sh
```

This script handles user creation, database setup, application build, and systemd service configuration.

## Architecture Overview

### Multi-tenancy Model

The system supports multiple organizations through the `organizations` table. All data entities (users, tickets, messages) are scoped to an organization via `organization_id` foreign keys.

- Default organization (ID=1) is created on first migration
- Each organization can have its own Telegram chat and Google Calendar
- Users belong to a single organization

### Authentication Flow

1. **Session Management**: In-memory session store in [internal/auth/auth.go](internal/auth/auth.go)
   - Sessions stored as map[string]*Session (not persisted to DB)
   - 24-hour session expiration
   - Session data passed via HTTP headers (X-User-ID, X-Organization-ID, X-User-Role)

2. **User Roles**:
   - `admin` - full access to all features
   - `agent` - can manage tickets, assign agents, update statuses
   - `customer` - can create tickets and add messages

3. **Default Admin**: Created by migration with credentials `admin@example.com` / `admin123`

### Telegram Bot Integration

The bot runs in a goroutine started from [main.go](main.go):

1. **User Creation**: When a Telegram user messages the bot, they're automatically created as a customer in organization 1
2. **Ticket Creation**: Non-command messages create new tickets
3. **Thread Management**: Replying to bot messages adds comments to the corresponding ticket
4. **Message Tracking**: Both ticket and message records store `telegram_message_id` and `telegram_chat_id` for bidirectional communication

Key behaviors in [internal/bot/bot.go](internal/bot/bot.go):
- Telegram users are linked via `telegram_id` field on users table
- Tickets track which Telegram message created them
- Callback queries handle inline keyboard actions (assign, resolve)

### Google Calendar Integration

OAuth2 flow managed in [internal/calendar/calendar.go](internal/calendar/calendar.go):

1. Tokens stored per-organization in `google_calendar_tokens` table
2. Automatic token refresh when expired
3. Events can be created with `CreateEvent()` function
4. Calendar ID per organization (defaults to "primary")

### Database Layer

All database operations are in [internal/db/](internal/db/):
- [db.go](internal/db/db.go) - connection management and migrations
- [users.go](internal/db/users.go) - user CRUD operations
- [tickets.go](internal/db/tickets.go) - ticket management
- [messages.go](internal/db/messages.go) - message operations
- [attachments.go](internal/db/attachments.go) - file attachments
- [calendar.go](internal/db/calendar.go) - Google Calendar token storage
- [organizations.go](internal/db/organizations.go) - organization operations

The global `db.DB` variable is used throughout (initialized in main.go).

### HTTP Request Flow

1. Request enters chi router in [main.go](main.go)
2. Public routes: `/login`, `/logout` (no auth required)
3. Protected routes pass through middleware (lines 76-97) that:
   - Validates session cookie
   - Stores user context in request headers
   - Redirects to login if unauthorized
4. Handlers in [internal/handlers/](internal/handlers/) process requests:
   - Extract user context from headers with helper functions (getUserID, getOrganizationID, getUserRole)
   - Perform authorization checks (org-scoped data access)
   - Render templates or redirect

### Template Rendering

HTML templates in [templates/](templates/) use Go's `html/template`:
- [base.html](templates/base.html) - base layout
- [login.html](templates/login.html) - authentication
- [dashboard.html](templates/dashboard.html) - ticket list with filters
- [ticket.html](templates/ticket.html) - ticket detail and conversation

Templates initialized once at startup via `handlers.InitTemplates()`.

## Important Implementation Details

### Session Context Passing

User context is NOT stored in request.Context(). Instead, it's passed via HTTP headers after middleware validation:
- `r.Header.Set("X-User-ID", ...)`
- `r.Header.Set("X-Organization-ID", ...)`
- `r.Header.Set("X-User-Role", ...)`

When modifying handlers, use the helper functions in handlers package (not shown in handlers.go but should exist).

### File Uploads

Upload directory created at startup (`uploads/` folder). The upload handler in [handlers.go](internal/handlers/handlers.go) is currently not implemented (returns 501).

### Ticket Status Flow

Automatic status transitions in [handlers.go](internal/handlers/handlers.go):
- When agent adds message to "open" ticket → changes to "in_progress"
- Manual status updates available to admin/agent roles only

### Error Handling

The application continues running if optional components fail:
- Telegram bot initialization failure is logged but doesn't stop startup
- Migration errors are logged as warnings (tables may already exist)
- Missing Google Calendar credentials prevents calendar features but allows app to run

## Common Patterns

### Adding a New Database Model

1. Define struct in [internal/models/models.go](internal/models/models.go)
2. Create migration SQL in new file in [migrations/](migrations/)
3. Add CRUD functions in appropriate file under [internal/db/](internal/db/)
4. Update `RunMigrations()` in [db.go](internal/db/db.go) to execute new migration

### Adding a New HTTP Handler

1. Define handler function in [internal/handlers/](internal/handlers/)
2. Extract user context from headers using helper functions
3. Validate organization-scoped access
4. Register route in [main.go](main.go) (public or protected group)

### Extending Telegram Bot Commands

Add new command cases in `handleCommand()` function in [bot.go](internal/bot/bot.go). Follow the pattern of existing commands (`/start`, `/help`, `/status`).

## Default Credentials

- **Admin Login**: `admin@example.com` / `admin123`
- **Database**: `helpdesk` / `helpdesk_password` (from env.example)

Change these immediately in production environments.
