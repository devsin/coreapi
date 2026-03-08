# Core API

Go REST API starter template for social platforms. Users follow each other, discover profiles, and track engagement analytics.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25.5 |
| Router | chi/v5 |
| Database | PostgreSQL 18 |
| DB Driver | pgx/v5 (pool) |
| SQL Gen | sqlc v1.30.0 |
| Migrations | golang-migrate |
| Auth | Supabase JWT (ES256 via JWKS) |
| Config | cleanenv (.env) |
| Logging | zap |

## Quick Start

```bash
# Prerequisites: Go 1.25+, PostgreSQL, Make

# Copy env and edit
cp .env.example .env

# Run migrations
make migrate-up

# Generate sqlc code
make sqlc

# Run development server
make run
```

## Environment Variables

```env
PORT=8080
ENV=development
DB_HOST=localhost
DB_PORT=5432
DB_NAME=core
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_SSL=disable
JWT_JWKS_URL=https://your-project.supabase.co/auth/v1/.well-known/jwks.json
CORS_ALLOWED_ORIGINS=http://localhost:5173
LOG_LEVEL=debug
LOG_FORMAT=console
GEOIP_DB_PATH=db/geocity/GeoLite2-City.mmdb
DISCORD_CONTACT_WEBHOOK_URL=              # Optional: Discord webhook for contact form notifications
```

## Make Commands

| Command | Description |
|---------|-------------|
| `make run` | Run the server |
| `make lint` | Run golangci-lint |
| `make sqlc` | Generate sqlc code |
| `make migrate-up` | Run all migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-new MIGRATION_NAME=<name>` | Create new migration pair |

## Project Structure

```
cmd/coreapi/main.go            Entry point
common/
├── config/                    Generic config loader
├── db/                        pgxpool connection factory
├── httpx/                     JSON helpers, middleware, HTTP server
├── logger/                    Zap logger factory
└── reserved/                  Reserved usernames
internal/
├── app/routes.go              Router + all route definitions
├── auth/                      JWKS-based JWT verification + middleware
├── users/                     User domain (CRUD, follow/unfollow)
├── contact/                   Contact form (submit, Discord webhook notification)
├── insights/                  Tracking (profile views), dashboard, geo, events
├── media/                     Media upload handling
├── notification/              Notification system (SSE + polling)
├── search/                    Full-text search (users via tsvector)
├── session/                   Active session tracking & management (multi-device)
├── settings/                  User notification/privacy settings
├── storage/                   Object storage (Cloudflare R2)
db/
├── migrations/                SQL migration files
└── queries/                   sqlc query files
gen/db/                        Generated sqlc code (DO NOT EDIT)
```

## Architecture

**Handler → Service → Repository** pattern for each domain.

- **Handler**: HTTP concerns — parse request, extract auth, call service, write JSON
- **Service**: Business logic, validation, access control
- **Repository**: Database via sqlc, model conversion

## Database Tables

| Table | Description |
|-------|-------------|
| users | Profiles (id from Supabase, username, display_name, bio, avatar_url) |
| user_settings | Notification & privacy prefs (email_follow, profile_public, allow_nsfw_content, etc.) |
| follows | User ↔ User follows (composite PK, no self-follow) |
| profile_views | Profile view events (IP, UA, geo, referrer, dedup indexes) |
| daily_insights | Pre-aggregated daily counters (views, followers) |
| user_sessions | Active sessions per device (session_id, IP, UA, browser/OS/device, last_active_at) |
| contact_messages | Contact form submissions (name, email, subject, message, status) |

## API Endpoints

### Public
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| GET | /api/users/username/{username} | User profile with stats |
| GET | /api/users/username/{username}/followers | Paginated followers |
| GET | /api/users/username/{username}/following | Paginated following |
| GET | /api/users/check-username/{username} | Check username availability |
| GET | /api/search | Full-text search (users) |
| POST | /api/contact | Submit contact form (+ Discord notification) |
| POST | /api/insights/profile/{userId}/view | Track profile view |

### Protected (JWT required)
| Method | Path | Description |
|--------|------|-------------|
| GET | /api/me | Get/create auth user (upsert) |
| PUT | /api/me | Update profile |
| POST/DELETE | /api/users/{id}/follow | Follow/unfollow |
| GET/PUT | /api/settings | Get/update settings |
| GET | /api/insights/overview | Insights dashboard (metrics, time series, breakdowns) |
| GET | /api/insights/events | Paginated event log |
| GET | /api/insights/geo | Geographic data (map points) |
| GET | /api/sessions | List active sessions for current user |
| DELETE | /api/sessions | Delete all sessions except current |
| DELETE | /api/sessions/{id} | Delete a specific session by DB ID |
| POST | /api/media/upload | Upload media (avatar) |
| GET | /api/notifications | Get notifications (SSE stream) |

## Docker

```bash
# Build and run with docker-compose
docker compose up --build
```

Multi-stage Dockerfile: builds Go binary, runs on Alpine with ca-certificates.

## License

Private — all rights reserved.
