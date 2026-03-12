# Obeya Cloud SaaS — Design Spec

## Overview

Obeya Cloud is a web-based SaaS version of the Obeya task tracking CLI. It enables multi-user team collaboration and provides a full web UI alongside the existing CLI, with both as equal first-class clients of the same API.

**Core value prop:** Humans and AI agents collaborating on the same board, from web or terminal, with real-time updates.

## Goals

- Full SaaS platform — teams sign up, create orgs, manage boards via web UI
- CLI remains a first-class citizen — agents and developers use `ob` commands against the cloud API
- Real-time collaboration — changes appear instantly for all viewers
- Personal cloud sync as a secondary use case — boards accessible from any machine
- Free tier + paid plans

## Non-Goals (MVP)

- Billing/payment flow (free tier limits enforced, but no Stripe integration yet)
- Self-hosted deployment option
- CI/CD integrations / webhooks
- Email/push notifications
- File attachments on items

---

## System Architecture

### Stack

- **Frontend + API:** Next.js (React for web UI, API routes for backend logic)
- **Persistence:** Appwrite (Database, Auth, Realtime, Storage)
- **CLI:** Go (`ob` binary with cloud backend support)
- **Deployment:** Vercel (or Railway/Fly.io) for Next.js, Appwrite Cloud for managed Appwrite

### Architecture Pattern: CQRS Hybrid

```
WRITES (create, move, edit, delete)
  CLI / Web UI → Next.js API Routes → Appwrite Database
                 (business logic, validation, history tracking)

REALTIME (live updates)
  Appwrite Database change → Appwrite Realtime Engine
    → WebSocket direct to CLI (gorilla/websocket)
    → WebSocket direct to Web UI (Appwrite JS SDK)

REST READS (list, show, export)
  CLI / Web UI → Next.js API Routes → Appwrite Database
```

Both CLI and Web UI use the same two paths — writes go through Next.js API routes (which own all business logic), realtime subscriptions go directly to Appwrite's WebSocket endpoint.

### Why CQRS Hybrid

- **No proxy complexity:** Appwrite handles realtime natively — no need to build SSE/WebSocket relay in Next.js
- **Serverless compatible:** Works on Vercel — no long-lived server connections required for the API layer
- **True client parity:** Both CLI and Web use identical write and realtime paths
- **Native WebSocket performance:** No added latency from proxying events

### Component Overview

```
┌─────────────┐  ┌─────────────┐
│  ob CLI (Go) │  │  Web UI     │
│  Local/Cloud │  │  (Next.js)  │
└──────┬───────┘  └──────┬──────┘
       │ REST            │ REST + SSR
       ▼                 ▼
┌─────────────────────────────────┐
│  Next.js API Routes             │
│  /api/auth  /api/boards         │
│  /api/items /api/orgs           │
│  /api/plans                     │
│  Middleware: auth, rate limit,  │
│  usage metering, validation     │
└──────────────┬──────────────────┘
               │ Appwrite Server SDK
               ▼
┌─────────────────────────────────┐
│  Appwrite                       │
│  Database │ Auth │ Realtime     │
│  Storage                        │
└──────┬──────────────┬───────────┘
       │ WebSocket     │ WebSocket
       ▼               ▼
   ob CLI TUI      Web UI (live)
```

---

## Data Model

### Appwrite Collections

Collections mirror the local `board.json` schema exactly — cloud is a different storage backend for the same data structure.

#### boards

| Field           | Type     | Description                    |
|-----------------|----------|--------------------------------|
| $id             | auto     | Appwrite document ID           |
| name            | string   | Board name                     |
| owner_id        | string   | User who created it            |
| org_id          | string?  | Null = personal board          |
| display_counter | integer  | Next #N number                 |
| columns         | string   | JSON array: [{name, limit}]    |
| created_at      | datetime |                                |
| updated_at      | datetime |                                |

#### items

| Field       | Type     | Description                    |
|-------------|----------|--------------------------------|
| $id         | auto     | Appwrite document ID           |
| board_id    | string   | FK → boards                    |
| display_num | integer  | Human-readable #N              |
| type        | enum     | epic \| story \| task          |
| title       | string   |                                |
| description | string   |                                |
| status      | string   | Column name                    |
| priority    | enum     | low \| medium \| high \| urgent |
| parent_id   | string?  | FK → items (self-ref)          |
| assignee_id | string?  | FK → users                     |
| blocked_by  | string[] | IDs of blocking items          |
| project     | string?  | Project name tag               |
| created_at  | datetime |                                |
| updated_at  | datetime |                                |

#### item_history

| Field      | Type     | Description                    |
|------------|----------|--------------------------------|
| $id        | auto     |                                |
| item_id    | string   | FK → items                     |
| board_id   | string   | FK → boards                    |
| user_id    | string   | Who did it                     |
| session_id | string   | CLI session / web              |
| action     | string   | created \| moved \| edited     |
| detail     | string   | "status: todo → done"          |
| timestamp  | datetime |                                |

Separate collection (not embedded in items) for efficient activity feed queries.

#### plans

| Field        | Type     | Description                    |
|--------------|----------|--------------------------------|
| $id          | auto     |                                |
| board_id     | string   | FK → boards                    |
| display_num  | integer  | Shares counter with items      |
| title        | string   |                                |
| source_path  | string   | Original file path             |
| content      | string   | Plan markdown body             |
| linked_items | string[] | IDs of linked items            |
| created_at   | datetime |                                |

#### orgs

| Field      | Type     | Description                    |
|------------|----------|--------------------------------|
| $id        | auto     |                                |
| name       | string   | Org display name               |
| slug       | string   | URL-safe identifier            |
| owner_id   | string   | Creator                        |
| plan       | enum     | free \| pro \| enterprise      |
| created_at | datetime |                                |

#### org_members

| Field       | Type      | Description                   |
|-------------|-----------|-------------------------------|
| $id         | auto      |                               |
| org_id      | string    | FK → orgs                     |
| user_id     | string    | Appwrite user ID              |
| role        | enum      | owner \| admin \| member      |
| invited_at  | datetime  |                               |
| accepted_at | datetime? |                               |

#### board_members

| Field      | Type     | Description                    |
|------------|----------|--------------------------------|
| $id        | auto     |                                |
| board_id   | string   | FK → boards                    |
| user_id    | string   | Appwrite user ID               |
| role       | enum     | owner \| editor \| viewer      |
| invited_at | datetime |                                |

#### api_tokens

| Field        | Type      | Description                   |
|--------------|-----------|-------------------------------|
| $id          | auto      |                               |
| user_id      | string    | Token owner                   |
| name         | string    | "niladri's macbook"           |
| token_hash   | string    | bcrypt hash                   |
| scopes       | string[]  | boards:read, items:write, etc.|
| last_used_at | datetime? |                               |
| expires_at   | datetime? |                               |

CLI stores the raw token in `~/.obeya/credentials.json` (never committed). Server stores only the bcrypt hash.

### Local → Cloud Migration

When `ob init --cloud` migrates a local board:

```
board.json → POST /api/boards/import
  ├── Creates board document (preserving columns, display_counter)
  ├── Creates all items (preserving display_num, parent relationships)
  ├── Creates all item_history entries
  └── Creates all plans with linked item mappings
```

ID mapping is handled server-side — local IDs are mapped to new Appwrite document IDs, and all references (parent_id, blocked_by, linked_items) are updated accordingly.

---

## API Design

### Response Envelope

All endpoints return a consistent shape:

```json
{
  "ok": true,
  "data": { ... },
  "meta": { "total": 42, "page": 1 }
}
```

Error responses:

```json
{
  "ok": false,
  "error": { "code": "BOARD_NOT_FOUND", "message": "Board xyz does not exist" }
}
```

No silent failures — errors throw with codes and messages, following the fast-fail principle.

### Endpoints

#### Auth

```
POST   /api/auth/signup           Email/password signup
POST   /api/auth/login            Email/password login
POST   /api/auth/oauth/:provider  OAuth flow (github, google)
POST   /api/auth/token            Create API token (for CLI)
DELETE /api/auth/token/:id        Revoke API token
GET    /api/auth/me               Current user profile
```

#### Boards

```
GET    /api/boards                List boards (personal + org)
POST   /api/boards                Create board
GET    /api/boards/:id            Get board (with columns)
PATCH  /api/boards/:id            Update board (name, columns, WIP limits)
DELETE /api/boards/:id            Delete board
POST   /api/boards/import         Import from local board.json
GET    /api/boards/:id/export     Export as board.json (for local use)
```

#### Board Members

```
GET    /api/boards/:id/members          List members
POST   /api/boards/:id/members          Invite member (email or user_id)
PATCH  /api/boards/:id/members/:uid     Change role
DELETE /api/boards/:id/members/:uid     Remove member
```

#### Items

```
GET    /api/boards/:id/items      List items (filterable: status, type, assignee)
POST   /api/boards/:id/items      Create item (task/story/epic)
GET    /api/items/:id             Get item with history
PATCH  /api/items/:id             Edit item (title, description, priority)
DELETE /api/items/:id             Delete item
POST   /api/items/:id/move        Move to column (status change)
POST   /api/items/:id/assign      Assign to user
POST   /api/items/:id/block       Add blocker
DELETE /api/items/:id/block/:bid  Unblock
```

#### Item History

```
GET    /api/items/:id/history     Full history for an item
GET    /api/boards/:id/activity   Board-wide activity feed
```

#### Plans

```
GET    /api/boards/:id/plans      List plans
POST   /api/boards/:id/plans      Import plan (markdown body)
GET    /api/plans/:id             Get plan with linked items
POST   /api/plans/:id/link        Link items to plan
DELETE /api/plans/:id/link/:iid   Unlink item
```

#### Orgs

```
GET    /api/orgs                  List user's orgs
POST   /api/orgs                  Create org
GET    /api/orgs/:id              Get org details
PATCH  /api/orgs/:id              Update org (name, settings)
DELETE /api/orgs/:id              Delete org
```

#### Org Members

```
GET    /api/orgs/:id/members          List members
POST   /api/orgs/:id/members          Invite member
PATCH  /api/orgs/:id/members/:uid     Change role
DELETE /api/orgs/:id/members/:uid     Remove member
```

---

## Auth & Multi-tenancy

### Auth Flows

| Client | Flow |
|--------|------|
| Web UI | Appwrite OAuth (GitHub/Google) → session cookie → API routes verify session via Appwrite Server SDK |
| CLI    | `ob login` → opens browser to OAuth flow → on success, API creates api_token → CLI stores token in `~/.obeya/credentials.json` → all requests include `Authorization: Bearer <token>` |

### API Route Auth Middleware

```
Request arrives at /api/*
  1. Appwrite session cookie present? → validate via Appwrite Server SDK → allow
  2. Authorization: Bearer <token>? → hash token, lookup in api_tokens collection → allow
  3. Neither? → 401 Unauthorized
```

### OAuth Provider Setup (Appwrite)

#### GitHub OAuth

1. Go to [GitHub Developer Settings](https://github.com/settings/developers) → OAuth Apps → New OAuth App
2. Set:
   - Application name: `Obeya`
   - Homepage URL: `https://obeya.app`
   - Authorization callback URL: `https://<APPWRITE_ENDPOINT>/v1/account/sessions/oauth2/callback/github/[PROJECT_ID]`
3. Copy Client ID and Client Secret
4. In Appwrite Console → Auth → OAuth2 Providers → GitHub:
   - Enable GitHub provider
   - Paste Client ID and Client Secret
   - Set success redirect: `https://obeya.app/auth/callback`
   - Set failure redirect: `https://obeya.app/auth/error`

#### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials → Create OAuth Client ID
2. Set:
   - Application type: Web application
   - Name: `Obeya`
   - Authorized redirect URIs: `https://<APPWRITE_ENDPOINT>/v1/account/sessions/oauth2/callback/google/[PROJECT_ID]`
3. Copy Client ID and Client Secret
4. In Appwrite Console → Auth → OAuth2 Providers → Google:
   - Enable Google provider
   - Paste Client ID and Client Secret
   - Set success redirect: `https://obeya.app/auth/callback`
   - Set failure redirect: `https://obeya.app/auth/error`

#### Appwrite Project Setup

1. Create project in Appwrite Console (or via CLI: `appwrite projects create`)
2. Note the Project ID and API Endpoint
3. Create a Server API Key with scopes: `databases.read`, `databases.write`, `users.read`, `users.write`, `teams.read`, `teams.write`
4. Store in Next.js environment variables:
   ```
   APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
   APPWRITE_PROJECT_ID=<your-project-id>
   APPWRITE_API_KEY=<your-server-api-key>
   ```

### CLI Auth Flow Detail

```
$ ob login
  → CLI starts local HTTP server on localhost:9876
  → Opens browser to: https://obeya.app/auth/cli?callback=http://localhost:9876/callback
  → User authenticates via GitHub or Google OAuth
  → Server creates api_token document (stores bcrypt hash)
  → Server redirects to: http://localhost:9876/callback?token=ob_tok_...
  → CLI receives token, writes ~/.obeya/credentials.json
  → CLI stops local server
  → "Logged in as niladribose. Token stored."
```

### Permission Model

#### Org-level access

| Role   | Permissions                                         |
|--------|-----------------------------------------------------|
| owner  | Full control — delete org, manage billing, all boards |
| admin  | Manage members, create/delete boards                 |
| member | Access all org boards, create/edit items              |

#### Board-level access (personal board sharing)

| Role   | Permissions                                 |
|--------|---------------------------------------------|
| owner  | Full control — delete board, manage members  |
| editor | Create, edit, move, delete items             |
| viewer | Read-only                                    |

#### Resolution order

```
1. Is user an org member? → gets org role on ALL org boards
2. Is user a board member? → gets board-specific role
3. Neither? → 403 Forbidden
```

### Free Tier Limits

| Constraint          | Free        | Pro (future)  |
|---------------------|-------------|---------------|
| Personal boards     | 3           | Unlimited     |
| Orgs                | 1           | Unlimited     |
| Members per org     | 3           | Unlimited     |
| Items per board     | 100         | Unlimited     |
| History retention   | 30 days     | Unlimited     |
| Realtime            | Standard    | Priority      |

Limits enforced in API middleware on create operations. Returns error code `PLAN_LIMIT_REACHED` with upgrade guidance.

---

## CLI Cloud Mode

### Mode Detection

```
ob <command>
  ├── .obeya/cloud.json exists? → Cloud mode (read API URL + board_id)
  ├── .obeya/board.json exists? → Local mode (current behavior, unchanged)
  └── Neither → "Not initialized. Run ob init or ob init --cloud"
```

### New CLI Commands

| Command          | Description                          |
|------------------|--------------------------------------|
| `ob init --cloud` | Create cloud board or migrate local  |
| `ob login`        | OAuth via browser → store API token  |
| `ob logout`       | Clear stored credentials             |

All existing commands (`ob list`, `ob move`, `ob show`, etc.) work unchanged — they use the cloud backend transparently.

### Cloud Config: `.obeya/cloud.json`

```json
{
  "api_url": "https://obeya.app/api",
  "board_id": "abc123",
  "org_id": "org_456",
  "user": "niladribose"
}
```

This file is committed to the repo (no secrets). Credentials are stored separately.

### Credentials: `~/.obeya/credentials.json`

```json
{
  "token": "ob_tok_...",
  "user_id": "usr_...",
  "created_at": "2026-03-12T10:00:00Z"
}
```

This file is NOT committed — lives in the user's home directory.

### `ob init --cloud` Flow

```
Scenario 1: Fresh project (no .obeya/ directory)
  → ob login (if not already authenticated)
  → Prompt: "Create board in: (a) personal, (b) org <name>?"
  → POST /api/boards → creates cloud board
  → Write .obeya/cloud.json
  → Done

Scenario 2: Existing local board (.obeya/board.json exists)
  → ob login (if not already authenticated)
  → Prompt: "Local board found with 42 items.
             (a) Migrate to cloud  (b) Keep local"
  → If migrate:
      POST /api/boards/import with board.json body
      → Backup .obeya/board.json → .obeya/board.json.local-backup
      → Write .obeya/cloud.json
      → "Migrated 42 items to cloud board #abc123"
  → If keep local:
      → No change, exit

Scenario 3: Already cloud (.obeya/cloud.json exists)
  → "Already connected to cloud board #abc123.
     Run ob init --local to switch to local mode."
```

### Go CLI Architecture Change

```go
// store/backend.go — new interface
type Backend interface {
    GetBoard() (*Board, error)
    ListItems(filter ItemFilter) ([]Item, error)
    GetItem(id string) (*Item, error)
    CreateItem(item *Item) (*Item, error)
    UpdateItem(id string, updates ItemUpdates) (*Item, error)
    MoveItem(id string, status string) (*Item, error)
    DeleteItem(id string) error
    // ... mirrors current store functions
}

// store/local.go — existing behavior, unchanged
type LocalBackend struct { boardPath string }

// store/cloud.go — new HTTP client backend
type CloudBackend struct { apiURL string; token string; boardID string }
```

Commands in `cmd/` call `store.GetBackend()` which returns `LocalBackend` or `CloudBackend` based on `.obeya/cloud.json` presence. Zero changes to command layer.

---

## Realtime

### Subscription Model

Clients subscribe to board-specific document changes via Appwrite's WebSocket:

```
Web UI:  Appwrite JS SDK → client.subscribe(
           'databases.<db>.collections.items.documents',
           filterByBoardId
         )

CLI TUI: gorilla/websocket → Appwrite WebSocket endpoint
         (same subscription, different client library)
```

### Events That Trigger UI Updates

| Event            | UI Response                      |
|------------------|----------------------------------|
| Item created     | New card appears on board        |
| Item moved       | Card animates to new column      |
| Item edited      | Title/description updates in-place |
| Item deleted     | Card removed from board          |
| Member joined    | Member list updates              |
| History added    | Activity feed updates            |

### Scope

- **TUI (interactive `ob` board view):** Subscribes to WebSocket for live updates — replaces current `fsnotify` file-watching, but over the network
- **Non-interactive CLI commands** (`ob list`, `ob move`): Fire-and-forget REST calls — no subscription needed

### Appwrite Realtime Permissions

Appwrite realtime respects document-level permissions. Users only receive events for boards they have access to (via org membership or board membership).

---

## Web UI Pages

### MVP Pages

| Route                    | Page                                |
|--------------------------|-------------------------------------|
| `/`                      | Landing page / marketing            |
| `/auth/login`            | Login (email + OAuth buttons)       |
| `/auth/signup`           | Signup                              |
| `/auth/callback`         | OAuth callback handler              |
| `/auth/cli`              | CLI auth flow (redirect after OAuth)|
| `/dashboard`             | Board list (personal + org boards)  |
| `/boards/:id`            | Kanban board view (main workspace)  |
| `/boards/:id/activity`   | Activity feed for a board           |
| `/boards/:id/settings`   | Board settings, members, columns    |
| `/orgs/new`              | Create org                          |
| `/orgs/:id`              | Org dashboard — boards list         |
| `/orgs/:id/members`      | Org member management               |
| `/orgs/:id/settings`     | Org settings                        |
| `/settings`              | User profile, API tokens            |

### Kanban Board View (`/boards/:id`)

The main workspace — mirrors the TUI board view:

- Columns rendered left-to-right with WIP limit indicators
- Cards show: type icon, #N, title, priority badge, assignee avatar
- Drag-and-drop to move between columns
- Click card → detail panel (description, history, subtasks, blocking)
- Real-time: cards animate in/out/between columns as changes arrive via WebSocket

---

## MVP Scope

### Included

| Area            | Features                                                    |
|-----------------|-------------------------------------------------------------|
| Auth            | Signup/login, GitHub + Google OAuth, API tokens, `ob login` |
| Boards          | CRUD, columns with WIP limits, import/export board.json     |
| Items           | Full task/story/epic CRUD, move, assign, block, history     |
| Orgs            | Create, invite members, roles (owner/admin/member)          |
| Board sharing   | Invite by email, roles (owner/editor/viewer)                |
| Realtime        | Live board updates via Appwrite WebSocket (web + TUI)       |
| CLI cloud mode  | `ob init --cloud`, migration, Backend interface             |
| Web UI          | Kanban board, item detail, org/board management, activity   |
| API             | All REST endpoints with consistent error envelope           |

### Excluded (Post-MVP)

| Area                | Notes                                              |
|---------------------|----------------------------------------------------|
| Billing/payments    | Stripe integration, plan upgrade flow              |
| Dashboard/metrics   | Velocity charts, burndown, cycle time (port from TUI) |
| Plan documents      | Import/link plans via web UI                       |
| Attachments/storage | File uploads on items                              |
| Notifications       | Email/push for assignments, mentions               |
| Audit log UI        | Dedicated audit log page (history is tracked)      |
| CI/CD integration   | Webhooks, GitHub Actions bot                       |
| Self-hosted option  | Docker compose for on-premise deployment           |

---

## Post-MVP Roadmap

### Phase 2: Monetization & Metrics

- **Stripe integration** — subscribe to Pro plan, manage billing
- **Plan limit enforcement with upgrade flow** — soft limits → upgrade prompt in UI
- **Dashboard page** — port TUI dashboard metrics (velocity, burndown, cycle time) to web
- **Usage metering** — track API calls, items created, storage used per org

### Phase 3: Collaboration Features

- **Plan documents** — import, view, and link plans from the web UI
- **Attachments** — file uploads on items via Appwrite Storage
- **Comments** — threaded comments on items (beyond history log)
- **Notifications** — email digests, in-app notification bell, @mentions
- **Audit log page** — filterable audit log for org admins

### Phase 4: Integrations & Extensibility

- **Webhooks** — notify external services on board events
- **GitHub integration** — link PRs/issues to board items, auto-move on merge
- **CI/CD bot** — GitHub Action that creates/updates board items from pipeline
- **Slack integration** — board updates posted to Slack channels

### Phase 5: Enterprise & Self-Hosted

- **Docker Compose** — self-hosted Appwrite + Next.js deployment
- **SSO/SAML** — enterprise identity providers
- **Advanced permissions** — custom roles, field-level access control
- **Data export** — bulk export for compliance and backup

---

## Project Structure

```
obeya-cloud/
├── app/                      # Next.js App Router
│   ├── (auth)/               # Auth pages (login, signup, callback)
│   ├── (dashboard)/          # Authenticated pages
│   │   ├── dashboard/        # Board list
│   │   ├── boards/[id]/      # Kanban board view
│   │   ├── orgs/[id]/        # Org pages
│   │   └── settings/         # User settings
│   ├── api/                  # API routes
│   │   ├── auth/             # Auth endpoints
│   │   ├── boards/           # Board CRUD + import/export
│   │   ├── items/            # Item CRUD + move/block
│   │   ├── orgs/             # Org management
│   │   └── plans/            # Plan management
│   ├── layout.tsx            # Root layout
│   └── page.tsx              # Landing page
├── lib/                      # Shared server-side code
│   ├── appwrite.ts           # Appwrite Server SDK client
│   ├── auth.ts               # Auth middleware helpers
│   ├── errors.ts             # Error codes and response helpers
│   └── validation.ts         # Request validation
├── components/               # React components
│   ├── board/                # Kanban board components
│   ├── items/                # Item cards, detail view
│   ├── layout/               # Shell, nav, sidebar
│   └── ui/                   # Shared UI primitives
├── hooks/                    # React hooks (realtime subscriptions, etc.)
├── public/                   # Static assets
├── .env.local                # Environment variables (not committed)
└── package.json
```

---

## Environment Variables

```bash
# Appwrite
APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
APPWRITE_PROJECT_ID=<project-id>
APPWRITE_API_KEY=<server-api-key>
APPWRITE_DATABASE_ID=obeya

# OAuth (configured in Appwrite Console, referenced here for docs)
# GitHub: Client ID + Secret set in Appwrite Console → Auth → GitHub
# Google: Client ID + Secret set in Appwrite Console → Auth → Google

# App
NEXT_PUBLIC_APP_URL=https://obeya.app
NEXT_PUBLIC_APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
NEXT_PUBLIC_APPWRITE_PROJECT_ID=<project-id>
```
