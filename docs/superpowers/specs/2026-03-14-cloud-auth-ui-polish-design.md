# Obeya Cloud: Auth Fixes, UI Polish & E2E Verification

**Date:** 2026-03-14
**Status:** Approved

## Problem

The Obeya Cloud web app has three categories of issues preventing production readiness:
1. **Auth gaps** — signup doesn't auto-login, no logout endpoint exists
2. **UI mismatch** — dashboard has a dark background bug and doesn't match the CLI's visual identity
3. **Deployment drift** — Vercel env vars point to wrong Appwrite project after database recreation

## Solution

Three phases executed sequentially: auth fixes, UI redesign to match CLI aesthetic, then end-to-end verification with redeployment.

---

## Phase 1: Auth Fixes

### Signup with Auto-Login

**Current:** `POST /api/auth/signup` creates user via `getUsers().create()` and returns user data. No session. User must navigate to login separately.

**New:** After creating the user, immediately create an Appwrite session via the REST API (same pattern as the login route), set the `obeya_session` httpOnly cookie, and return user + session data. The signup page redirects to `/dashboard` on success.

```
POST /api/auth/signup
  1. Validate email, password, name
  2. getUsers().create() — create Appwrite user
  3. POST ${APPWRITE_ENDPOINT}/account/sessions/email — create session
  4. getUsers().get(session.userId) — fetch user data
  5. Set httpOnly cookie: obeya_session = session.userId
  6. Return { user, session }
```

**Files:** `web/app/api/auth/signup/route.ts`

### Logout Endpoint

**New route:** `POST /api/auth/logout`

```
POST /api/auth/logout
  1. Clear the obeya_session cookie (maxAge: 0)
  2. Return { ok: true }
```

**Files:**
- Create: `web/app/api/auth/logout/route.ts`
- Modify: `web/components/layout/sidebar.tsx` — add logout button at bottom

### Signup Page Cookie Redirect

**Current:** `web/app/(auth)/auth/signup/page.tsx` calls `/api/auth/signup` and shows success message.

**New:** On success, redirect to `/dashboard` (same as login page behavior). The httpOnly cookie is set by the API response.

**Files:** `web/app/(auth)/auth/signup/page.tsx`

---

## Phase 2: UI Polish — CLI-Matched Dark Theme

### Design System

The web UI adopts the CLI TUI's visual identity: dark background, monospace font, same colors.

| Token | Hex | Usage |
|---|---|---|
| `--bg-primary` | `#0d1117` | Page background, card background |
| `--bg-secondary` | `#161b22` | Sidebar, column backgrounds |
| `--border-default` | `#30363d` | Card borders, dividers |
| `--border-active` | `#58a6ff` | Active column, selected card (2px) |
| `--text-primary` | `#c9d1d9` | Main text |
| `--text-secondary` | `#8b949e` | Dim text, labels |
| `--text-faint` | `#484f58` | Help text, inactive headers |
| `--type-epic` | `#bc8cff` | Epic type label |
| `--type-story` | `#58a6ff` | Story type label |
| `--type-task` | `#8b949e` | Task type label |
| `--pri-critical` | `#f85149` | Critical/high priority dots |
| `--pri-medium` | `#e3b341` | Medium priority dots |
| `--pri-low` | `#3fb950` | Low priority dots |
| `--assignee` | `#58a6ff` | Assigned user (@name) |
| `--unassigned` | `#f85149` | Unassigned indicator |

### Font

**Primary:** JetBrains Mono (Google Fonts) with fallbacks: `'JetBrains Mono', 'Fira Code', 'SF Mono', 'Cascadia Code', 'Menlo', monospace`

### Logo

8-bit pixel kanban board icon (V1): 3-column grid with purple/blue/green column headers, stacked card blocks, blue border. Renders as a CSS grid at multiple sizes (favicon 16px, sidebar 24px, header 48px).

### Components to Restyle

| Component | Change |
|---|---|
| `app-shell.tsx` | Dark background `#0d1117` on main area |
| `sidebar.tsx` | Dark sidebar `#161b22`, pixel logo, logout button, monospace |
| `header.tsx` | Dark header `#0d1117`, border `#21262d` |
| `globals.css` | CSS custom properties for design tokens, JetBrains Mono import |
| `board-list.tsx` | Dark cards matching CLI card style |
| `button.tsx` | Dark theme variants |
| `input.tsx` | Dark input with visible text on dark bg |
| Login/signup pages | Dark theme, monospace, pixel logo |

### Error Handling

**Current:** `handleError` in `web/lib/response.ts` exposes raw error messages for non-AppError exceptions.

**New:** In production (`NODE_ENV === 'production'`), non-AppError exceptions return generic "Something went wrong" message. In development, keep the detailed message.

**File:** `web/lib/response.ts`

---

## Phase 3: End-to-End Verification

### Vercel Env Vars

Update Vercel environment variables to point to the correct Appwrite project (databases were recreated in `69b2c740003cbab398cc`):

```
APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
APPWRITE_PROJECT_ID=69b2c740003cbab398cc
APPWRITE_API_KEY=<the standard key>
APPWRITE_DATABASE_ID=obeya
NEXT_PUBLIC_APP_URL=https://obeya-web.vercel.app
NEXT_PUBLIC_APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
NEXT_PUBLIC_APPWRITE_PROJECT_ID=69b2c740003cbab398cc
```

### Local Verification Flow

1. Start local dev server on port 6001
2. Navigate to signup page
3. Create account → verify redirect to dashboard
4. Create a board → verify it appears
5. Navigate to board → verify kanban columns render
6. Verify logout works → redirect to login
7. Login with created account → verify dashboard

### Vercel Deployment

After local verification passes:
1. Update env vars via `vercel env`
2. `vercel deploy --prod`
3. Repeat verification flow on production URL

---

## Testing

### Phase 1 Tests

- `web/__tests__/api/auth/signup.test.ts` — update to verify cookie is set on signup response
- Create: `web/__tests__/api/auth/logout.test.ts` — verify cookie is cleared
- Integration: test signup → auto-login flow against real Appwrite

### Phase 2 Tests

- Visual regression via Chrome browser MCP (start local, screenshot key pages)
- Verify JetBrains Mono loads
- Verify dark theme applied (no white backgrounds)

### Phase 3 Tests

- Smoke test: signup + board creation + item add on production

---

## Architecture

### Files Modified

| File | Change |
|---|---|
| `web/app/api/auth/signup/route.ts` | Add session creation + cookie setting |
| `web/app/api/auth/logout/route.ts` | New — clears session cookie |
| `web/app/(auth)/auth/signup/page.tsx` | Redirect to dashboard on success |
| `web/components/layout/sidebar.tsx` | Add pixel logo, logout button, dark theme |
| `web/components/layout/header.tsx` | Dark theme |
| `web/components/layout/app-shell.tsx` | Dark background on main area |
| `web/app/globals.css` | Design tokens, JetBrains Mono import |
| `web/components/ui/input.tsx` | Dark theme input styling |
| `web/components/ui/button.tsx` | Dark theme button variants |
| `web/lib/response.ts` | Generic error messages in production |
| `web/app/(auth)/auth/login/page.tsx` | Dark theme, pixel logo |

### New Files

| File | Purpose |
|---|---|
| `web/app/api/auth/logout/route.ts` | Logout endpoint |
| `web/__tests__/api/auth/logout.test.ts` | Logout tests |
| `web/components/ui/pixel-logo.tsx` | Pixel kanban board logo component |
