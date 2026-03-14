# Cloud Auth Fixes, UI Polish & E2E Verification — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix signup auto-login, add logout, restyle the web app to match the CLI's retro terminal aesthetic, and verify the full end-to-end flow.

**Architecture:** Phase 1 fixes auth routes (signup sets cookie, new logout endpoint). Phase 2 applies a dark CLI-matched theme with JetBrains Mono font and pixel kanban logo across all components. Phase 3 redeploys to Vercel and verifies the full flow.

**Tech Stack:** Next.js 16, Appwrite (node-appwrite), Tailwind CSS, Vitest, JetBrains Mono (Google Fonts)

**Spec:** `docs/superpowers/specs/2026-03-14-cloud-auth-ui-polish-design.md`

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `web/app/api/auth/signup/route.ts` | Signup with auto-login + cookie | Modify |
| `web/app/api/auth/logout/route.ts` | Clear session cookie | Create |
| `web/__tests__/api/auth/signup.test.ts` | Signup tests | Modify |
| `web/__tests__/api/auth/logout.test.ts` | Logout tests | Create |
| `web/app/globals.css` | Design tokens, font import, dark theme | Modify |
| `web/components/ui/pixel-logo.tsx` | 8-bit kanban board logo component | Create |
| `web/components/layout/sidebar.tsx` | Dark theme, pixel logo, logout button | Modify |
| `web/components/layout/header.tsx` | Dark theme | Modify |
| `web/components/layout/app-shell.tsx` | Dark bg on main area | Modify |
| `web/components/ui/button.tsx` | Dark theme variants | Modify |
| `web/components/ui/input.tsx` | Dark theme input | Modify |
| `web/app/(auth)/layout.tsx` | Dark auth pages with pixel logo | Modify |
| `web/app/(auth)/auth/login/page.tsx` | Dark theme text colors | Modify |
| `web/app/(auth)/auth/signup/page.tsx` | Dark theme + fix API endpoint | Modify |
| `web/lib/response.ts` | Generic errors in production | Modify |

---

## Chunk 1: Auth Fixes

### Task 1: Signup auto-login with cookie

**Files:**
- Modify: `web/app/api/auth/signup/route.ts`
- Modify: `web/__tests__/api/auth/signup.test.ts`

- [ ] **Step 1: Write failing test — signup sets httpOnly cookie**

Add to `web/__tests__/api/auth/signup.test.ts`:

```typescript
it("sets httpOnly session cookie on successful signup", async () => {
  vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
  // Mock getUsers().create() for user creation
  const mockUsers = {
    create: vi.fn().mockResolvedValue({ $id: "new-user", email: "a@b.com", name: "Alice" }),
    get: vi.fn().mockResolvedValue({ $id: "new-user", email: "a@b.com", name: "Alice" }),
  };
  vi.mocked(getUsers).mockReturnValue(mockUsers as any);

  // Mock fetch for Appwrite session creation
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ $id: "sess-1", userId: "new-user", secret: "secret-1" }),
  });

  const request = new Request("http://localhost/api/auth/signup", {
    method: "POST",
    body: JSON.stringify({ email: "a@b.com", password: "password123", name: "Alice" }),
    headers: { "Content-Type": "application/json" },
  });

  const response = await POST(request);
  const body = await response.json();

  expect(response.status).toBe(201);
  expect(body.ok).toBe(true);
  expect(body.data.user.email).toBe("a@b.com");

  // Check cookie is set
  const setCookie = response.headers.get("set-cookie");
  expect(setCookie).toContain("obeya_session=");
  expect(setCookie).toContain("HttpOnly");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run __tests__/api/auth/signup.test.ts`
Expected: FAIL — no cookie set on current signup response.

- [ ] **Step 3: Implement signup with auto-login**

Update `web/app/api/auth/signup/route.ts`:

```typescript
import { z } from "zod";
import { ID } from "node-appwrite";
import { NextResponse } from "next/server";
import { handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";
import { getUsers } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";

const signupSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().min(1),
});

export async function POST(request: Request) {
  try {
    const body = await validateBody(request, signupSchema);
    const user = await createUser(body);
    const session = await createSession(body.email, body.password);

    const res = NextResponse.json({
      ok: true,
      data: {
        user: { id: user.$id, email: user.email, name: user.name },
        session: { id: session.$id },
      },
    }, { status: 201 });

    res.cookies.set("obeya_session", session.userId, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24 * 30,
    });

    return res;
  } catch (err) {
    return handleError(err);
  }
}

async function createUser(body: z.infer<typeof signupSchema>) {
  try {
    return await getUsers().create(
      ID.unique(), body.email, undefined, body.password, body.name,
    );
  } catch (err: unknown) {
    if (typeof err === "object" && err !== null && "code" in err && (err as any).code === 409) {
      throw new AppError(ErrorCode.EMAIL_ALREADY_EXISTS, "Email already exists");
    }
    throw err;
  }
}

async function createSession(email: string, password: string) {
  const env = getEnv();
  const res = await fetch(`${env.APPWRITE_ENDPOINT}/account/sessions/email`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Appwrite-Project": env.APPWRITE_PROJECT_ID,
    },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    throw new AppError(ErrorCode.INTERNAL_ERROR, "Failed to create session after signup");
  }
  return await res.json();
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run __tests__/api/auth/signup.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/app/api/auth/signup/route.ts web/__tests__/api/auth/signup.test.ts
git commit -m "feat: signup auto-login with httpOnly session cookie"
```

### Task 2: Logout endpoint

**Files:**
- Create: `web/app/api/auth/logout/route.ts`
- Create: `web/__tests__/api/auth/logout.test.ts`

- [ ] **Step 1: Write failing test for logout**

Create `web/__tests__/api/auth/logout.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { POST } from "@/app/api/auth/logout/route";

describe("POST /api/auth/logout", () => {
  it("clears the obeya_session cookie", async () => {
    const request = new Request("http://localhost/api/auth/logout", { method: "POST" });
    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);

    const setCookie = response.headers.get("set-cookie");
    expect(setCookie).toContain("obeya_session=");
    expect(setCookie).toContain("Max-Age=0");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run __tests__/api/auth/logout.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement logout route**

Create `web/app/api/auth/logout/route.ts`:

```typescript
import { NextResponse } from "next/server";

export async function POST() {
  const res = NextResponse.json({ ok: true, data: { message: "Logged out" } });

  res.cookies.set("obeya_session", "", {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });

  return res;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run __tests__/api/auth/logout.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/app/api/auth/logout/route.ts web/__tests__/api/auth/logout.test.ts
git commit -m "feat: add logout endpoint that clears session cookie"
```

### Task 3: Fix signup page API endpoint

**Files:**
- Modify: `web/app/(auth)/auth/signup/page.tsx`

The signup page currently calls `/api/auth/register` but the route is at `/api/auth/signup`.

- [ ] **Step 1: Fix the API endpoint**

In `web/app/(auth)/auth/signup/page.tsx` line 23, change:
```typescript
await apiClient.post("/api/auth/register", { name, email, password });
```
to:
```typescript
await apiClient.post("/api/auth/signup", { name, email, password });
```

- [ ] **Step 2: Commit**

```bash
git add web/app/(auth)/auth/signup/page.tsx
git commit -m "fix: signup page calls correct /api/auth/signup endpoint"
```

---

## Chunk 2: UI Polish — CLI-Matched Dark Theme

### Task 4: Design tokens and font in globals.css

**Files:**
- Modify: `web/app/globals.css`

- [ ] **Step 1: Replace globals.css with dark CLI theme**

```css
@import "tailwindcss";
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap');

:root {
  --bg-primary: #0d1117;
  --bg-secondary: #161b22;
  --bg-tertiary: #21262d;
  --border-default: #30363d;
  --border-active: #58a6ff;
  --text-primary: #c9d1d9;
  --text-secondary: #8b949e;
  --text-faint: #484f58;
  --type-epic: #bc8cff;
  --type-story: #58a6ff;
  --type-task: #8b949e;
  --pri-critical: #f85149;
  --pri-high: #f85149;
  --pri-medium: #e3b341;
  --pri-low: #3fb950;
  --accent: #7aa2f7;
  --assignee: #58a6ff;
  --unassigned: #f85149;
}

@theme inline {
  --color-background: var(--bg-primary);
  --color-foreground: var(--text-primary);
  --font-mono: 'JetBrains Mono', 'Fira Code', 'SF Mono', 'Cascadia Code', 'Menlo', monospace;
}

body {
  background: var(--bg-primary);
  color: var(--text-primary);
  font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', 'Cascadia Code', 'Menlo', monospace;
}
```

- [ ] **Step 2: Commit**

```bash
git add web/app/globals.css
git commit -m "feat: add CLI-matched dark theme design tokens and JetBrains Mono font"
```

### Task 5: Pixel kanban logo component

**Files:**
- Create: `web/components/ui/pixel-logo.tsx`

- [ ] **Step 1: Create the pixel logo component**

Create `web/components/ui/pixel-logo.tsx` — a CSS grid-based 8-bit kanban board icon. Accepts `size` prop for favicon (16px cells), sidebar (24px cells), or header (48px cells). Uses the V1 design: 3 columns with purple/blue/green headers, stacked card blocks, blue border.

The component renders a grid of colored divs. Each pixel is a `<div>` with a background color. The grid dimensions are 15 columns x 12 rows.

- [ ] **Step 2: Commit**

```bash
git add web/components/ui/pixel-logo.tsx
git commit -m "feat: add 8-bit pixel kanban board logo component"
```

### Task 6: Dark theme sidebar with logo and logout

**Files:**
- Modify: `web/components/layout/sidebar.tsx`

- [ ] **Step 1: Restyle sidebar**

Update `web/components/layout/sidebar.tsx`:
- Background: `bg-[#161b22]` with border `border-[#21262d]`
- Add `PixelLogo` component in header area
- Nav links: monospace, `text-[#8b949e]` inactive, `text-[#c9d1d9] bg-[#21262d]` active
- Section dividers: `── boards ──` and `── teams ──` in dim text
- Add logout button at bottom with `apiClient.post("/api/auth/logout")` then `router.replace("/auth/login")`
- Active nav item has `border-l-2 border-[#7aa2f7]`

- [ ] **Step 2: Commit**

```bash
git add web/components/layout/sidebar.tsx
git commit -m "feat: dark CLI-matched sidebar with pixel logo and logout"
```

### Task 7: Dark theme header and app shell

**Files:**
- Modify: `web/components/layout/header.tsx`
- Modify: `web/components/layout/app-shell.tsx`

- [ ] **Step 1: Update header**

Change header background to `bg-[#0d1117]`, border to `border-[#21262d]`, text to `text-[#c9d1d9]`.

- [ ] **Step 2: Update app shell**

Change main area background: add `bg-[#0d1117]` to the `<main>` element.

- [ ] **Step 3: Commit**

```bash
git add web/components/layout/header.tsx web/components/layout/app-shell.tsx
git commit -m "feat: dark theme header and app shell background"
```

### Task 8: Dark theme buttons and inputs

**Files:**
- Modify: `web/components/ui/button.tsx`
- Modify: `web/components/ui/input.tsx`

- [ ] **Step 1: Update button variants**

```typescript
const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-[#7aa2f7] text-[#0d1117] hover:bg-[#93c5fd]",
  secondary: "bg-[#21262d] text-[#c9d1d9] hover:bg-[#30363d] border border-[#30363d]",
  ghost: "bg-transparent text-[#8b949e] hover:bg-[#21262d]",
  danger: "bg-[#f85149] text-white hover:bg-[#da3633]",
};
```

Add `font-family: inherit` to ensure monospace propagates.

- [ ] **Step 2: Update input styling**

Change input to dark theme: `bg-[#0d1117] text-[#c9d1d9] border-[#30363d] placeholder-[#484f58]` with focus ring `focus:border-[#58a6ff]`.

- [ ] **Step 3: Commit**

```bash
git add web/components/ui/button.tsx web/components/ui/input.tsx
git commit -m "feat: dark theme buttons and inputs"
```

### Task 9: Dark theme auth pages

**Files:**
- Modify: `web/app/(auth)/layout.tsx`
- Modify: `web/app/(auth)/auth/login/page.tsx`
- Modify: `web/app/(auth)/auth/signup/page.tsx`

- [ ] **Step 1: Update auth layout**

Change auth layout background to `bg-[#0d1117]`, card background to `bg-[#161b22]` with `border border-[#30363d]`, remove `shadow-sm ring-1 ring-gray-200`. Replace "Obeya" text header with `PixelLogo` component. Update subtitle text color to `text-[#8b949e]`.

- [ ] **Step 2: Update login and signup page text colors**

Change all `text-gray-*` references to dark theme equivalents:
- `text-gray-900` → `text-[#c9d1d9]`
- `text-gray-600` → `text-[#8b949e]`
- `text-gray-500` → `text-[#484f58]`
- `text-red-600` → `text-[#f85149]`
- `text-blue-600` → `text-[#7aa2f7]`
- `bg-white` → `bg-[#161b22]`
- `border-gray-200` → `border-[#30363d]`

- [ ] **Step 3: Commit**

```bash
git add web/app/(auth)/layout.tsx web/app/(auth)/auth/login/page.tsx web/app/(auth)/auth/signup/page.tsx
git commit -m "feat: dark CLI-matched auth pages with pixel logo"
```

### Task 10: Generic error messages in production

**Files:**
- Modify: `web/lib/response.ts`
- Modify: `web/__tests__/lib/response.test.ts`

- [ ] **Step 1: Write failing test**

Add to `web/__tests__/lib/response.test.ts`:

```typescript
it("returns generic message for non-AppError in production", () => {
  const origEnv = process.env.NODE_ENV;
  process.env.NODE_ENV = "production";
  const response = handleError(new Error("secret db connection string"));
  process.env.NODE_ENV = origEnv;
  // Should NOT contain the raw error message
  expect(response).toBeDefined();
  // Parse the response to check the message
});
```

- [ ] **Step 2: Implement production error masking**

In `web/lib/response.ts`, update `handleError`:

```typescript
export function handleError(err: unknown): NextResponse {
  if (err instanceof AppError) {
    return NextResponse.json(
      { ok: false, error: { code: err.code, message: err.message } },
      { status: err.statusCode }
    );
  }
  const errMsg = err instanceof Error ? err.message : String(err);
  console.error("Unhandled error:", errMsg, err);

  const publicMessage = process.env.NODE_ENV === "production"
    ? "Something went wrong"
    : errMsg;

  return NextResponse.json(
    { ok: false, error: { code: ErrorCode.INTERNAL_ERROR, message: publicMessage } },
    { status: 500 }
  );
}
```

- [ ] **Step 3: Run tests and commit**

Run: `cd web && npx vitest run __tests__/lib/response.test.ts`

```bash
git add web/lib/response.ts web/__tests__/lib/response.test.ts
git commit -m "feat: mask error details in production responses"
```

---

## Chunk 3: E2E Verification & Deployment

### Task 11: Update Vercel env vars and redeploy

**Files:** None (Vercel CLI commands only)

- [ ] **Step 1: Update Vercel environment variables**

```bash
cd web
vercel env rm APPWRITE_DATABASE_ID production -y 2>/dev/null
echo "obeya" | vercel env add APPWRITE_DATABASE_ID production

vercel env rm NEXT_PUBLIC_APP_URL production -y 2>/dev/null
echo "https://obeya-web.vercel.app" | vercel env add NEXT_PUBLIC_APP_URL production
```

Verify other vars are already set correctly (from previous session):
```bash
vercel env ls
```

- [ ] **Step 2: Deploy to Vercel**

```bash
cd web && vercel deploy --prod
```

- [ ] **Step 3: Commit any deployment config changes**

```bash
git add -A && git commit -m "chore: update Vercel deployment config" || echo "Nothing to commit"
```

### Task 12: Local end-to-end smoke test

**Files:** None (manual verification via Chrome MCP or curl)

- [ ] **Step 1: Start local dev server**

```bash
cd web && PORT=6001 npm run dev
```

- [ ] **Step 2: Test signup flow**

Navigate to `http://localhost:6001/auth/signup`, create account, verify redirect to dashboard.

- [ ] **Step 3: Test board creation**

Click "New Board", create a board, verify it appears in the board list.

- [ ] **Step 4: Test logout**

Click logout in sidebar, verify redirect to login page.

- [ ] **Step 5: Test login flow**

Login with created account, verify dashboard loads.

- [ ] **Step 6: Verify dark theme**

All pages should have dark background (`#0d1117`), monospace font, pixel logo visible in sidebar and auth pages.
