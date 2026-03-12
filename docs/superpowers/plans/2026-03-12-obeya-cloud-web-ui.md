# Obeya Cloud Web UI (Part A) — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Next.js web UI for Obeya Cloud — shared infrastructure (API client, UI components, layout shell), authentication pages (login, signup, OAuth callback), and the dashboard page (board listing with creation dialog).

**Architecture:** Next.js 15 App Router. Server Components for data fetching, Client Components (`"use client"`) for interactivity. Tailwind CSS utility classes for styling. Vitest + @testing-library/react for component tests. All API calls from client components go through a typed fetch wrapper (`lib/api-client.ts`).

**Tech Stack:** Next.js 15, TypeScript, React 19, Tailwind CSS, Vitest, @testing-library/react, @testing-library/jest-dom

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md`

**Depends on:** `docs/superpowers/plans/2026-03-12-obeya-cloud-foundation.md` (Plan 1 — API routes, Appwrite integration, auth middleware must be complete)

**Repository:** `~/code/obeya-cloud` (same repo created by Plan 1)

---

## File Structure

```
obeya-cloud/
├── app/
│   ├── layout.tsx                                    # Root layout (already exists from Plan 1)
│   ├── page.tsx                                      # Landing placeholder (already exists)
│   ├── (auth)/
│   │   ├── layout.tsx                                # Centered card layout for auth pages
│   │   └── auth/
│   │       ├── login/
│   │       │   └── page.tsx                          # Login page (email + OAuth buttons)
│   │       ├── signup/
│   │       │   └── page.tsx                          # Signup page (registration form)
│   │       ├── callback/
│   │       │   └── page.tsx                          # OAuth callback redirect handler
│   │       └── error/
│   │           └── page.tsx                          # Auth error page
│   └── (dashboard)/
│       ├── layout.tsx                                # Auth-gated layout with sidebar + header
│       └── dashboard/
│           └── page.tsx                              # Dashboard — board list grouped by personal/org
├── lib/
│   ├── api-client.ts                                 # Typed fetch wrapper for /api/* routes
│   └── types.ts                                      # Shared UI types (Board, Item, User, Org)
├── components/
│   ├── ui/
│   │   ├── button.tsx                                # Button component
│   │   ├── input.tsx                                 # Input component
│   │   ├── badge.tsx                                 # Badge component
│   │   ├── modal.tsx                                 # Modal dialog component
│   │   └── avatar.tsx                                # Avatar component
│   ├── layout/
│   │   ├── app-shell.tsx                             # Main shell — sidebar + header + content area
│   │   ├── sidebar.tsx                               # Sidebar navigation
│   │   └── header.tsx                                # Top header bar with user menu
│   └── dashboard/
│       ├── board-card.tsx                            # Board card (name, item count, last updated)
│       ├── board-list.tsx                            # Board list grouped by personal/org
│       └── new-board-dialog.tsx                      # "New Board" creation modal
├── hooks/
│   └── use-auth.ts                                   # Auth state hook (current user)
├── __tests__/
│   ├── components/
│   │   ├── ui/
│   │   │   ├── button.test.tsx
│   │   │   ├── input.test.tsx
│   │   │   ├── badge.test.tsx
│   │   │   ├── modal.test.tsx
│   │   │   └── avatar.test.tsx
│   │   ├── layout/
│   │   │   ├── sidebar.test.tsx
│   │   │   └── header.test.tsx
│   │   └── dashboard/
│   │       ├── board-card.test.tsx
│   │       ├── board-list.test.tsx
│   │       └── new-board-dialog.test.tsx
│   └── lib/
│       └── api-client.test.ts
└── vitest.config.ts                                  # Updated for jsdom + react testing
```

---

## Prerequisites

Before starting, install UI testing dependencies:

```bash
cd ~/code/obeya-cloud
npm install -D @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom
```

Update `vitest.config.ts` to support React component testing:

```typescript
import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  test: {
    globals: true,
    environment: "jsdom",
    include: ["__tests__/**/*.test.{ts,tsx}"],
    setupFiles: ["__tests__/setup.ts"],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
    },
  },
});
```

Create test setup file:

Create: `obeya-cloud/__tests__/setup.ts`

```typescript
import "@testing-library/jest-dom/vitest";
```

---

## Chunk 1: Shared Infrastructure

### Task 1: API Client Helper

**Files:**
- Create: `obeya-cloud/lib/types.ts`
- Create: `obeya-cloud/lib/api-client.ts`
- Test: `obeya-cloud/__tests__/lib/api-client.test.ts`

- [ ] **Step 1: Create shared UI types**

Create: `obeya-cloud/lib/types.ts`

```typescript
export interface Board {
  id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: Column[];
  item_count: number;
  created_at: string;
  updated_at: string;
}

export interface Column {
  name: string;
  limit: number;
}

export interface Item {
  id: string;
  board_id: string;
  display_num: number;
  type: "epic" | "story" | "task";
  title: string;
  description: string;
  status: string;
  priority: "low" | "medium" | "high" | "critical";
  parent_id: string | null;
  assignee_id: string | null;
  blocked_by: string[];
  tags: string[];
  project: string | null;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
}

export interface Org {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  plan: "free" | "pro" | "enterprise";
  created_at: string;
}

export interface ApiResponse<T> {
  ok: true;
  data: T;
  meta?: { total?: number; page?: number };
}

export interface ApiError {
  ok: false;
  error: { code: string; message: string };
}

export type ApiResult<T> = ApiResponse<T> | ApiError;
```

- [ ] **Step 2: Write failing test**

Create: `obeya-cloud/__tests__/lib/api-client.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiClient, ApiClientError } from "@/lib/api-client";

describe("apiClient", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    global.fetch = vi.fn();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("makes GET request and returns data", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(JSON.stringify({ ok: true, data: { id: "1" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const result = await apiClient.get<{ id: string }>("/api/boards");

    expect(global.fetch).toHaveBeenCalledWith("/api/boards", {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });
    expect(result).toEqual({ id: "1" });
  });

  it("makes POST request with body", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(JSON.stringify({ ok: true, data: { id: "new" } }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    );

    const result = await apiClient.post<{ id: string }>("/api/boards", {
      name: "My Board",
    });

    expect(global.fetch).toHaveBeenCalledWith("/api/boards", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "My Board" }),
    });
    expect(result).toEqual({ id: "new" });
  });

  it("makes PATCH request with body", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(JSON.stringify({ ok: true, data: { id: "1" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    await apiClient.patch("/api/boards/1", { name: "Updated" });

    expect(global.fetch).toHaveBeenCalledWith("/api/boards/1", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "Updated" }),
    });
  });

  it("makes DELETE request", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(JSON.stringify({ ok: true, data: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    await apiClient.delete("/api/boards/1");

    expect(global.fetch).toHaveBeenCalledWith("/api/boards/1", {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
    });
  });

  it("throws ApiClientError on error response", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(
        JSON.stringify({
          ok: false,
          error: { code: "BOARD_NOT_FOUND", message: "Not found" },
        }),
        { status: 404, headers: { "Content-Type": "application/json" } }
      )
    );

    await expect(apiClient.get("/api/boards/999")).rejects.toThrow(
      ApiClientError
    );

    try {
      await apiClient.get("/api/boards/999");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiClientError);
      expect((err as ApiClientError).code).toBe("BOARD_NOT_FOUND");
      expect((err as ApiClientError).statusCode).toBe(404);
    }
  });

  it("throws on network failure", async () => {
    vi.mocked(global.fetch).mockRejectedValue(new Error("Network error"));

    await expect(apiClient.get("/api/boards")).rejects.toThrow("Network error");
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/api-client.test.ts
```

Expected: FAIL -- module not found

- [ ] **Step 4: Write implementation**

Create: `obeya-cloud/lib/api-client.ts`

```typescript
export class ApiClientError extends Error {
  public readonly code: string;
  public readonly statusCode: number;

  constructor(code: string, message: string, statusCode: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.statusCode = statusCode;
  }
}

async function request<T>(
  url: string,
  method: string,
  body?: unknown
): Promise<T> {
  const options: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
  };

  if (body !== undefined) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(url, options);
  const json = await response.json();

  if (!json.ok) {
    throw new ApiClientError(
      json.error.code,
      json.error.message,
      response.status
    );
  }

  return json.data as T;
}

export const apiClient = {
  get<T>(url: string): Promise<T> {
    return request<T>(url, "GET");
  },

  post<T>(url: string, body: unknown): Promise<T> {
    return request<T>(url, "POST", body);
  },

  patch<T>(url: string, body: unknown): Promise<T> {
    return request<T>(url, "PATCH", body);
  },

  delete<T = null>(url: string): Promise<T> {
    return request<T>(url, "DELETE");
  },
};
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/api-client.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/api-client.ts lib/types.ts __tests__/lib/api-client.test.ts
git commit -m "feat: add typed API client helper and shared UI types"
```

---

### Task 2: Shared UI Components

**Files:**
- Create: `obeya-cloud/components/ui/button.tsx`
- Create: `obeya-cloud/components/ui/input.tsx`
- Create: `obeya-cloud/components/ui/badge.tsx`
- Create: `obeya-cloud/components/ui/modal.tsx`
- Create: `obeya-cloud/components/ui/avatar.tsx`
- Test: `obeya-cloud/__tests__/components/ui/button.test.tsx`
- Test: `obeya-cloud/__tests__/components/ui/input.test.tsx`
- Test: `obeya-cloud/__tests__/components/ui/badge.test.tsx`
- Test: `obeya-cloud/__tests__/components/ui/modal.test.tsx`
- Test: `obeya-cloud/__tests__/components/ui/avatar.test.tsx`

- [ ] **Step 1: Write failing tests for all UI components**

Create: `obeya-cloud/__tests__/components/ui/button.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "@/components/ui/button";

describe("Button", () => {
  it("renders children", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument();
  });

  it("calls onClick handler", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("applies primary variant by default", () => {
    render(<Button>Primary</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-blue-600");
  });

  it("applies secondary variant", () => {
    render(<Button variant="secondary">Secondary</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-gray-100");
  });

  it("applies ghost variant", () => {
    render(<Button variant="ghost">Ghost</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-transparent");
  });

  it("disables the button", () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("renders as full width", () => {
    render(<Button fullWidth>Full</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("w-full");
  });
});
```

Create: `obeya-cloud/__tests__/components/ui/input.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Input } from "@/components/ui/input";

describe("Input", () => {
  it("renders with label", () => {
    render(<Input label="Email" name="email" />);
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("renders with placeholder", () => {
    render(<Input label="Email" name="email" placeholder="you@example.com" />);
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument();
  });

  it("displays error message", () => {
    render(<Input label="Email" name="email" error="Required field" />);
    expect(screen.getByText("Required field")).toBeInTheDocument();
  });

  it("accepts user input", async () => {
    const onChange = vi.fn();
    render(<Input label="Name" name="name" onChange={onChange} />);
    await userEvent.type(screen.getByLabelText("Name"), "Alice");
    expect(onChange).toHaveBeenCalled();
  });

  it("supports password type", () => {
    render(<Input label="Password" name="password" type="password" />);
    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
  });
});
```

Create: `obeya-cloud/__tests__/components/ui/badge.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Badge } from "@/components/ui/badge";

describe("Badge", () => {
  it("renders children", () => {
    render(<Badge>Active</Badge>);
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("applies default variant", () => {
    render(<Badge>Default</Badge>);
    expect(screen.getByText("Default").className).toContain("bg-gray-100");
  });

  it("applies success variant", () => {
    render(<Badge variant="success">Done</Badge>);
    expect(screen.getByText("Done").className).toContain("bg-green-100");
  });

  it("applies warning variant", () => {
    render(<Badge variant="warning">Blocked</Badge>);
    expect(screen.getByText("Blocked").className).toContain("bg-yellow-100");
  });

  it("applies danger variant", () => {
    render(<Badge variant="danger">Critical</Badge>);
    expect(screen.getByText("Critical").className).toContain("bg-red-100");
  });

  it("applies info variant", () => {
    render(<Badge variant="info">New</Badge>);
    expect(screen.getByText("New").className).toContain("bg-blue-100");
  });
});
```

Create: `obeya-cloud/__tests__/components/ui/modal.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Modal } from "@/components/ui/modal";

describe("Modal", () => {
  it("renders nothing when closed", () => {
    render(
      <Modal open={false} onClose={() => {}} title="Test">
        <p>Content</p>
      </Modal>
    );
    expect(screen.queryByText("Test")).not.toBeInTheDocument();
  });

  it("renders title and children when open", () => {
    render(
      <Modal open={true} onClose={() => {}} title="My Modal">
        <p>Modal body</p>
      </Modal>
    );
    expect(screen.getByText("My Modal")).toBeInTheDocument();
    expect(screen.getByText("Modal body")).toBeInTheDocument();
  });

  it("calls onClose when backdrop is clicked", async () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose} title="Close Test">
        <p>Body</p>
      </Modal>
    );
    await userEvent.click(screen.getByTestId("modal-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when close button is clicked", async () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose} title="Close Test">
        <p>Body</p>
      </Modal>
    );
    await userEvent.click(screen.getByLabelText("Close modal"));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
```

Create: `obeya-cloud/__tests__/components/ui/avatar.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Avatar } from "@/components/ui/avatar";

describe("Avatar", () => {
  it("renders initials from name", () => {
    render(<Avatar name="Alice Smith" />);
    expect(screen.getByText("AS")).toBeInTheDocument();
  });

  it("renders single initial for single name", () => {
    render(<Avatar name="Alice" />);
    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("renders image when src is provided", () => {
    render(<Avatar name="Alice" src="/avatar.png" />);
    expect(screen.getByAltText("Alice")).toBeInTheDocument();
  });

  it("applies small size", () => {
    render(<Avatar name="Alice" size="sm" />);
    const el = screen.getByText("A");
    expect(el.parentElement?.className).toContain("h-8");
  });

  it("applies large size", () => {
    render(<Avatar name="Alice" size="lg" />);
    const el = screen.getByText("A");
    expect(el.parentElement?.className).toContain("h-12");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/ui/
```

Expected: FAIL -- modules not found

- [ ] **Step 3: Write Button component**

Create: `obeya-cloud/components/ui/button.tsx`

```typescript
"use client";

import { type ButtonHTMLAttributes, type ReactNode } from "react";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: Variant;
  size?: Size;
  fullWidth?: boolean;
}

const variantStyles: Record<Variant, string> = {
  primary:
    "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500",
  secondary:
    "bg-gray-100 text-gray-900 hover:bg-gray-200 focus:ring-gray-400",
  ghost:
    "bg-transparent text-gray-700 hover:bg-gray-100 focus:ring-gray-400",
  danger:
    "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500",
};

const sizeStyles: Record<Size, string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
  lg: "px-6 py-3 text-base",
};

export function Button({
  children,
  variant = "primary",
  size = "md",
  fullWidth = false,
  className = "",
  disabled,
  ...props
}: ButtonProps) {
  const classes = [
    "inline-flex items-center justify-center rounded-md font-medium",
    "transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2",
    "disabled:opacity-50 disabled:cursor-not-allowed",
    variantStyles[variant],
    sizeStyles[size],
    fullWidth ? "w-full" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button className={classes} disabled={disabled} {...props}>
      {children}
    </button>
  );
}
```

- [ ] **Step 4: Write Input component**

Create: `obeya-cloud/components/ui/input.tsx`

```typescript
"use client";

import { type InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  name: string;
  error?: string;
}

export function Input({
  label,
  name,
  error,
  className = "",
  ...props
}: InputProps) {
  const inputId = `input-${name}`;

  return (
    <div className="space-y-1">
      <label
        htmlFor={inputId}
        className="block text-sm font-medium text-gray-700"
      >
        {label}
      </label>
      <input
        id={inputId}
        name={name}
        className={[
          "block w-full rounded-md border px-3 py-2 text-sm",
          "placeholder:text-gray-400",
          "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
          error
            ? "border-red-300 focus:ring-red-500 focus:border-red-500"
            : "border-gray-300",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...props}
      />
      {error && <p className="text-sm text-red-600">{error}</p>}
    </div>
  );
}
```

- [ ] **Step 5: Write Badge component**

Create: `obeya-cloud/components/ui/badge.tsx`

```typescript
import { type ReactNode } from "react";

type Variant = "default" | "success" | "warning" | "danger" | "info";

interface BadgeProps {
  children: ReactNode;
  variant?: Variant;
}

const variantStyles: Record<Variant, string> = {
  default: "bg-gray-100 text-gray-800",
  success: "bg-green-100 text-green-800",
  warning: "bg-yellow-100 text-yellow-800",
  danger: "bg-red-100 text-red-800",
  info: "bg-blue-100 text-blue-800",
};

export function Badge({ children, variant = "default" }: BadgeProps) {
  return (
    <span
      className={[
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        variantStyles[variant],
      ].join(" ")}
    >
      {children}
    </span>
  );
}
```

- [ ] **Step 6: Write Modal component**

Create: `obeya-cloud/components/ui/modal.tsx`

```typescript
"use client";

import { type ReactNode } from "react";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function Modal({ open, onClose, title, children }: ModalProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        data-testid="modal-backdrop"
        className="fixed inset-0 bg-black/50"
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
          <button
            onClick={onClose}
            aria-label="Close modal"
            className="rounded-md p-1 text-gray-400 hover:text-gray-600"
          >
            <svg
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth="2"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
```

- [ ] **Step 7: Write Avatar component**

Create: `obeya-cloud/components/ui/avatar.tsx`

```typescript
interface AvatarProps {
  name: string;
  src?: string;
  size?: "sm" | "md" | "lg";
}

const sizeStyles = {
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-12 w-12 text-base",
};

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0][0].toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function Avatar({ name, src, size = "md" }: AvatarProps) {
  if (src) {
    return (
      <div
        className={[
          "relative inline-flex items-center justify-center rounded-full bg-gray-200",
          sizeStyles[size],
        ].join(" ")}
      >
        <img
          src={src}
          alt={name}
          className="h-full w-full rounded-full object-cover"
        />
      </div>
    );
  }

  return (
    <div
      className={[
        "inline-flex items-center justify-center rounded-full bg-blue-600 text-white font-medium",
        sizeStyles[size],
      ].join(" ")}
    >
      <span>{getInitials(name)}</span>
    </div>
  );
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/ui/
```

Expected: PASS (all 5 test files)

- [ ] **Step 9: Commit**

```bash
cd ~/code/obeya-cloud
git add components/ui/ __tests__/components/ui/
git commit -m "feat: add shared UI components — Button, Input, Badge, Modal, Avatar"
```

---

### Task 3: Layout Shell

**Files:**
- Create: `obeya-cloud/hooks/use-auth.ts`
- Create: `obeya-cloud/components/layout/sidebar.tsx`
- Create: `obeya-cloud/components/layout/header.tsx`
- Create: `obeya-cloud/components/layout/app-shell.tsx`
- Create: `obeya-cloud/app/(dashboard)/layout.tsx`
- Test: `obeya-cloud/__tests__/components/layout/sidebar.test.tsx`
- Test: `obeya-cloud/__tests__/components/layout/header.test.tsx`

- [ ] **Step 1: Create auth hook**

Create: `obeya-cloud/hooks/use-auth.ts`

```typescript
"use client";

import { useState, useEffect } from "react";
import { apiClient, ApiClientError } from "@/lib/api-client";
import type { User } from "@/lib/types";

interface AuthState {
  user: User | null;
  loading: boolean;
  error: string | null;
}

export function useAuth(): AuthState {
  const [state, setState] = useState<AuthState>({
    user: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;

    async function fetchUser() {
      try {
        const user = await apiClient.get<User>("/api/auth/me");
        if (!cancelled) {
          setState({ user, loading: false, error: null });
        }
      } catch (err) {
        if (!cancelled) {
          const message =
            err instanceof ApiClientError ? err.message : "Failed to load user";
          setState({ user: null, loading: false, error: message });
        }
      }
    }

    fetchUser();
    return () => {
      cancelled = true;
    };
  }, []);

  return state;
}
```

- [ ] **Step 2: Write failing tests for Sidebar and Header**

Create: `obeya-cloud/__tests__/components/layout/sidebar.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Sidebar } from "@/components/layout/sidebar";

describe("Sidebar", () => {
  it("renders navigation links", () => {
    render(<Sidebar />);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("renders the Obeya logo text", () => {
    render(<Sidebar />);
    expect(screen.getByText("Obeya")).toBeInTheDocument();
  });

  it("renders dashboard link with correct href", () => {
    render(<Sidebar />);
    const link = screen.getByRole("link", { name: /dashboard/i });
    expect(link).toHaveAttribute("href", "/dashboard");
  });

  it("renders settings link with correct href", () => {
    render(<Sidebar />);
    const link = screen.getByRole("link", { name: /settings/i });
    expect(link).toHaveAttribute("href", "/settings");
  });
});
```

Create: `obeya-cloud/__tests__/components/layout/header.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Header } from "@/components/layout/header";

describe("Header", () => {
  it("renders user name when provided", () => {
    render(
      <Header user={{ id: "1", name: "Alice", email: "alice@test.com" }} />
    );
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("renders user avatar initials", () => {
    render(
      <Header user={{ id: "1", name: "Alice Smith", email: "a@test.com" }} />
    );
    expect(screen.getByText("AS")).toBeInTheDocument();
  });

  it("renders nothing meaningful when no user", () => {
    render(<Header user={null} />);
    expect(screen.queryByText("Alice")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/layout/
```

Expected: FAIL -- modules not found

- [ ] **Step 4: Write Sidebar component**

Create: `obeya-cloud/components/layout/sidebar.tsx`

```typescript
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

interface NavItem {
  label: string;
  href: string;
  icon: string;
}

const navItems: NavItem[] = [
  { label: "Dashboard", href: "/dashboard", icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" },
  { label: "Settings", href: "/settings", icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-full w-64 flex-col border-r border-gray-200 bg-white">
      <div className="flex h-16 items-center px-6 border-b border-gray-200">
        <Link href="/dashboard" className="text-xl font-bold text-gray-900">
          Obeya
        </Link>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={[
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-blue-50 text-blue-700"
                  : "text-gray-700 hover:bg-gray-50",
              ].join(" ")}
            >
              <svg
                className="h-5 w-5 flex-shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth="1.5"
                stroke="currentColor"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d={item.icon} />
              </svg>
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
```

- [ ] **Step 5: Write Header component**

Create: `obeya-cloud/components/layout/header.tsx`

```typescript
"use client";

import { Avatar } from "@/components/ui/avatar";
import type { User } from "@/lib/types";

interface HeaderProps {
  user: User | null;
}

export function Header({ user }: HeaderProps) {
  return (
    <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div />
      <div className="flex items-center gap-3">
        {user && (
          <>
            <span className="text-sm text-gray-700">{user.name}</span>
            <Avatar name={user.name} size="sm" />
          </>
        )}
      </div>
    </header>
  );
}
```

- [ ] **Step 6: Write AppShell component**

Create: `obeya-cloud/components/layout/app-shell.tsx`

```typescript
"use client";

import { type ReactNode } from "react";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import type { User } from "@/lib/types";

interface AppShellProps {
  children: ReactNode;
  user: User | null;
}

export function AppShell({ children, user }: AppShellProps) {
  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header user={user} />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
```

- [ ] **Step 7: Write dashboard layout (auth-gated)**

Create: `obeya-cloud/app/(dashboard)/layout.tsx`

```typescript
"use client";

import { type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import { AppShell } from "@/components/layout/app-shell";

export default function DashboardLayout({ children }: { children: ReactNode }) {
  const { user, loading, error } = useAuth();
  const router = useRouter();

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600" />
      </div>
    );
  }

  if (error || !user) {
    router.push("/auth/login");
    return null;
  }

  return <AppShell user={user}>{children}</AppShell>;
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/layout/
```

Expected: PASS

- [ ] **Step 9: Commit**

```bash
cd ~/code/obeya-cloud
git add components/layout/ hooks/use-auth.ts app/\(dashboard\)/layout.tsx __tests__/components/layout/
git commit -m "feat: add layout shell with sidebar, header, and auth-gated dashboard layout"
```

---

## Chunk 2: Auth Pages

### Task 4: Auth Layout and Login Page

**Files:**
- Create: `obeya-cloud/app/(auth)/layout.tsx`
- Create: `obeya-cloud/app/(auth)/auth/login/page.tsx`

- [ ] **Step 1: Write auth layout (centered card)**

Create: `obeya-cloud/app/(auth)/layout.tsx`

```typescript
import { type ReactNode } from "react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold text-gray-900">Obeya</h1>
          <p className="mt-2 text-sm text-gray-600">
            Task tracking for humans and AI agents
          </p>
        </div>
        <div className="rounded-lg bg-white px-8 py-10 shadow-sm border border-gray-200">
          {children}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Write Login page**

Create: `obeya-cloud/app/(auth)/auth/login/page.tsx`

```typescript
"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiClient, ApiClientError } from "@/lib/api-client";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await apiClient.post("/api/auth/login", { email, password });
      router.push("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiClientError
          ? err.message
          : "Login failed. Please try again.";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function handleOAuth(provider: "github" | "google") {
    window.location.href = `/api/auth/oauth/${provider}`;
  }

  return (
    <div>
      <h2 className="mb-6 text-center text-xl font-semibold text-gray-900">
        Sign in to your account
      </h2>

      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Email"
          name="email"
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Input
          label="Password"
          name="password"
          type="password"
          placeholder="Enter your password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        <Button type="submit" fullWidth disabled={loading}>
          {loading ? "Signing in..." : "Sign in"}
        </Button>
      </form>

      <div className="mt-6">
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-200" />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-white px-2 text-gray-500">
              Or continue with
            </span>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-3">
          <Button
            type="button"
            variant="secondary"
            onClick={() => handleOAuth("github")}
          >
            <GitHubIcon />
            <span className="ml-2">GitHub</span>
          </Button>
          <Button
            type="button"
            variant="secondary"
            onClick={() => handleOAuth("google")}
          >
            <GoogleIcon />
            <span className="ml-2">Google</span>
          </Button>
        </div>
      </div>

      <p className="mt-6 text-center text-sm text-gray-600">
        Don&apos;t have an account?{" "}
        <Link
          href="/auth/signup"
          className="font-medium text-blue-600 hover:text-blue-500"
        >
          Sign up
        </Link>
      </p>
    </div>
  );
}

function GitHubIcon() {
  return (
    <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
      <path
        fillRule="evenodd"
        d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
        clipRule="evenodd"
      />
    </svg>
  );
}

function GoogleIcon() {
  return (
    <svg className="h-5 w-5" viewBox="0 0 24 24">
      <path
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
        fill="#4285F4"
      />
      <path
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
        fill="#34A853"
      />
      <path
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
        fill="#FBBC05"
      />
      <path
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
        fill="#EA4335"
      />
    </svg>
  );
}
```

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya-cloud
git add app/\(auth\)/
git commit -m "feat: add auth layout and login page with email + OAuth buttons"
```

---

### Task 5: Signup, Callback, and Error Pages

**Files:**
- Create: `obeya-cloud/app/(auth)/auth/signup/page.tsx`
- Create: `obeya-cloud/app/(auth)/auth/callback/page.tsx`
- Create: `obeya-cloud/app/(auth)/auth/error/page.tsx`

- [ ] **Step 1: Write Signup page**

Create: `obeya-cloud/app/(auth)/auth/signup/page.tsx`

```typescript
"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiClient, ApiClientError } from "@/lib/api-client";

export default function SignupPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await apiClient.post("/api/auth/signup", { name, email, password });
      router.push("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiClientError
          ? err.message
          : "Signup failed. Please try again.";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function handleOAuth(provider: "github" | "google") {
    window.location.href = `/api/auth/oauth/${provider}`;
  }

  return (
    <div>
      <h2 className="mb-6 text-center text-xl font-semibold text-gray-900">
        Create your account
      </h2>

      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Name"
          name="name"
          placeholder="Your name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <Input
          label="Email"
          name="email"
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Input
          label="Password"
          name="password"
          type="password"
          placeholder="At least 8 characters"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={8}
          required
        />
        <Button type="submit" fullWidth disabled={loading}>
          {loading ? "Creating account..." : "Create account"}
        </Button>
      </form>

      <div className="mt-6">
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-200" />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-white px-2 text-gray-500">
              Or continue with
            </span>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-3">
          <Button
            type="button"
            variant="secondary"
            onClick={() => handleOAuth("github")}
          >
            GitHub
          </Button>
          <Button
            type="button"
            variant="secondary"
            onClick={() => handleOAuth("google")}
          >
            Google
          </Button>
        </div>
      </div>

      <p className="mt-6 text-center text-sm text-gray-600">
        Already have an account?{" "}
        <Link
          href="/auth/login"
          className="font-medium text-blue-600 hover:text-blue-500"
        >
          Sign in
        </Link>
      </p>
    </div>
  );
}
```

- [ ] **Step 2: Write OAuth callback page**

Create: `obeya-cloud/app/(auth)/auth/callback/page.tsx`

```typescript
"use client";

import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";

export default function AuthCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const error = searchParams.get("error");

    if (error) {
      router.replace(`/auth/error?message=${encodeURIComponent(error)}`);
      return;
    }

    router.replace("/dashboard");
  }, [router, searchParams]);

  return (
    <div className="text-center">
      <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600" />
      <p className="mt-4 text-sm text-gray-600">Completing sign in...</p>
    </div>
  );
}
```

- [ ] **Step 3: Write Auth error page**

Create: `obeya-cloud/app/(auth)/auth/error/page.tsx`

```typescript
"use client";

import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function AuthErrorPage() {
  const searchParams = useSearchParams();
  const message =
    searchParams.get("message") || "An authentication error occurred.";

  return (
    <div className="text-center">
      <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
        <svg
          className="h-6 w-6 text-red-600"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth="2"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
      </div>
      <h2 className="mb-2 text-xl font-semibold text-gray-900">
        Authentication Error
      </h2>
      <p className="mb-6 text-sm text-gray-600">{message}</p>
      <Link href="/auth/login">
        <Button>Back to Login</Button>
      </Link>
    </div>
  );
}
```

- [ ] **Step 4: Commit**

```bash
cd ~/code/obeya-cloud
git add app/\(auth\)/auth/signup/ app/\(auth\)/auth/callback/ app/\(auth\)/auth/error/
git commit -m "feat: add signup page, OAuth callback handler, and auth error page"
```

---

## Chunk 3: Dashboard

### Task 6: Dashboard Page with Board List and New Board Dialog

**Files:**
- Create: `obeya-cloud/components/dashboard/board-card.tsx`
- Create: `obeya-cloud/components/dashboard/board-list.tsx`
- Create: `obeya-cloud/components/dashboard/new-board-dialog.tsx`
- Create: `obeya-cloud/app/(dashboard)/dashboard/page.tsx`
- Test: `obeya-cloud/__tests__/components/dashboard/board-card.test.tsx`
- Test: `obeya-cloud/__tests__/components/dashboard/board-list.test.tsx`
- Test: `obeya-cloud/__tests__/components/dashboard/new-board-dialog.test.tsx`

- [ ] **Step 1: Write failing tests**

Create: `obeya-cloud/__tests__/components/dashboard/board-card.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BoardCard } from "@/components/dashboard/board-card";

describe("BoardCard", () => {
  const board = {
    id: "board-1",
    name: "Sprint 42",
    item_count: 15,
    updated_at: "2026-03-12T10:00:00Z",
  };

  it("renders board name", () => {
    render(<BoardCard board={board} />);
    expect(screen.getByText("Sprint 42")).toBeInTheDocument();
  });

  it("renders item count", () => {
    render(<BoardCard board={board} />);
    expect(screen.getByText("15 items")).toBeInTheDocument();
  });

  it("renders last updated date", () => {
    render(<BoardCard board={board} />);
    expect(screen.getByText(/updated/i)).toBeInTheDocument();
  });

  it("links to the board page", () => {
    render(<BoardCard board={board} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/boards/board-1");
  });

  it("renders singular item text for count of 1", () => {
    render(<BoardCard board={{ ...board, item_count: 1 }} />);
    expect(screen.getByText("1 item")).toBeInTheDocument();
  });
});
```

Create: `obeya-cloud/__tests__/components/dashboard/board-list.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { BoardList } from "@/components/dashboard/board-list";

describe("BoardList", () => {
  const personalBoards = [
    { id: "b1", name: "Personal Board", item_count: 5, updated_at: "2026-03-12T10:00:00Z" },
  ];
  const orgGroups = [
    {
      org: { id: "org1", name: "Acme Corp" },
      boards: [
        { id: "b2", name: "Acme Sprint", item_count: 10, updated_at: "2026-03-11T10:00:00Z" },
      ],
    },
  ];

  it("renders personal boards section", () => {
    render(<BoardList personalBoards={personalBoards} orgGroups={[]} />);
    expect(screen.getByText("Personal Boards")).toBeInTheDocument();
    expect(screen.getByText("Personal Board")).toBeInTheDocument();
  });

  it("renders org boards section with org name", () => {
    render(<BoardList personalBoards={[]} orgGroups={orgGroups} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText("Acme Sprint")).toBeInTheDocument();
  });

  it("renders empty state when no boards", () => {
    render(<BoardList personalBoards={[]} orgGroups={[]} />);
    expect(screen.getByText(/no boards yet/i)).toBeInTheDocument();
  });

  it("renders both personal and org boards", () => {
    render(<BoardList personalBoards={personalBoards} orgGroups={orgGroups} />);
    expect(screen.getByText("Personal Boards")).toBeInTheDocument();
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });
});
```

Create: `obeya-cloud/__tests__/components/dashboard/new-board-dialog.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NewBoardDialog } from "@/components/dashboard/new-board-dialog";

describe("NewBoardDialog", () => {
  it("renders nothing when closed", () => {
    render(<NewBoardDialog open={false} onClose={() => {}} onCreate={() => {}} />);
    expect(screen.queryByText("New Board")).not.toBeInTheDocument();
  });

  it("renders form when open", () => {
    render(<NewBoardDialog open={true} onClose={() => {}} onCreate={() => {}} />);
    expect(screen.getByText("New Board")).toBeInTheDocument();
    expect(screen.getByLabelText("Board Name")).toBeInTheDocument();
  });

  it("calls onCreate with board name on submit", async () => {
    const onCreate = vi.fn();
    render(<NewBoardDialog open={true} onClose={() => {}} onCreate={onCreate} />);

    await userEvent.type(screen.getByLabelText("Board Name"), "Sprint 43");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(onCreate).toHaveBeenCalledWith("Sprint 43");
  });

  it("calls onClose when cancel is clicked", async () => {
    const onClose = vi.fn();
    render(<NewBoardDialog open={true} onClose={onClose} onCreate={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not submit with empty name", async () => {
    const onCreate = vi.fn();
    render(<NewBoardDialog open={true} onClose={() => {}} onCreate={onCreate} />);

    await userEvent.click(screen.getByRole("button", { name: /create/i }));
    expect(onCreate).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/dashboard/
```

Expected: FAIL -- modules not found

- [ ] **Step 3: Write BoardCard component**

Create: `obeya-cloud/components/dashboard/board-card.tsx`

```typescript
import Link from "next/link";

interface BoardCardProps {
  board: {
    id: string;
    name: string;
    item_count: number;
    updated_at: string;
  };
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "Updated just now";
  if (diffMins < 60) return `Updated ${diffMins}m ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `Updated ${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 30) return `Updated ${diffDays}d ago`;

  return `Updated on ${date.toLocaleDateString()}`;
}

export function BoardCard({ board }: BoardCardProps) {
  const itemLabel = board.item_count === 1 ? "1 item" : `${board.item_count} items`;

  return (
    <Link
      href={`/boards/${board.id}`}
      className="block rounded-lg border border-gray-200 bg-white p-5 transition-shadow hover:shadow-md"
    >
      <h3 className="text-base font-semibold text-gray-900">{board.name}</h3>
      <div className="mt-3 flex items-center justify-between text-sm text-gray-500">
        <span>{itemLabel}</span>
        <span>{formatRelativeTime(board.updated_at)}</span>
      </div>
    </Link>
  );
}
```

- [ ] **Step 4: Write BoardList component**

Create: `obeya-cloud/components/dashboard/board-list.tsx`

```typescript
import { BoardCard } from "@/components/dashboard/board-card";

interface BoardSummary {
  id: string;
  name: string;
  item_count: number;
  updated_at: string;
}

interface OrgGroup {
  org: { id: string; name: string };
  boards: BoardSummary[];
}

interface BoardListProps {
  personalBoards: BoardSummary[];
  orgGroups: OrgGroup[];
}

export function BoardList({ personalBoards, orgGroups }: BoardListProps) {
  const hasBoards = personalBoards.length > 0 || orgGroups.length > 0;

  if (!hasBoards) {
    return (
      <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center">
        <p className="text-sm text-gray-500">
          No boards yet. Create your first board to get started.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {personalBoards.length > 0 && (
        <BoardSection title="Personal Boards" boards={personalBoards} />
      )}
      {orgGroups.map((group) => (
        <BoardSection
          key={group.org.id}
          title={group.org.name}
          boards={group.boards}
        />
      ))}
    </div>
  );
}

function BoardSection({
  title,
  boards,
}: {
  title: string;
  boards: BoardSummary[];
}) {
  return (
    <section>
      <h2 className="mb-4 text-lg font-semibold text-gray-900">{title}</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {boards.map((board) => (
          <BoardCard key={board.id} board={board} />
        ))}
      </div>
    </section>
  );
}
```

- [ ] **Step 5: Write NewBoardDialog component**

Create: `obeya-cloud/components/dashboard/new-board-dialog.tsx`

```typescript
"use client";

import { useState, type FormEvent } from "react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface NewBoardDialogProps {
  open: boolean;
  onClose: () => void;
  onCreate: (name: string) => void;
}

export function NewBoardDialog({ open, onClose, onCreate }: NewBoardDialogProps) {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();

    if (!trimmed) {
      setError("Board name is required");
      return;
    }

    onCreate(trimmed);
    setName("");
    setError(null);
  }

  function handleClose() {
    setName("");
    setError(null);
    onClose();
  }

  return (
    <Modal open={open} onClose={handleClose} title="New Board">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Board Name"
          name="boardName"
          placeholder="e.g. Sprint 43"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setError(null);
          }}
          error={error ?? undefined}
        />
        <div className="flex justify-end gap-3">
          <Button type="button" variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit">Create Board</Button>
        </div>
      </form>
    </Modal>
  );
}
```

- [ ] **Step 6: Write Dashboard page**

Create: `obeya-cloud/app/(dashboard)/dashboard/page.tsx`

```typescript
"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { BoardList } from "@/components/dashboard/board-list";
import { NewBoardDialog } from "@/components/dashboard/new-board-dialog";
import { apiClient, ApiClientError } from "@/lib/api-client";
import type { Board, Org } from "@/lib/types";

interface BoardSummary {
  id: string;
  name: string;
  item_count: number;
  updated_at: string;
}

interface OrgGroup {
  org: { id: string; name: string };
  boards: BoardSummary[];
}

export default function DashboardPage() {
  const router = useRouter();
  const [personalBoards, setPersonalBoards] = useState<BoardSummary[]>([]);
  const [orgGroups, setOrgGroups] = useState<OrgGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const fetchBoards = useCallback(async () => {
    try {
      setLoading(true);
      const boards = await apiClient.get<Board[]>("/api/boards");
      const orgs = await apiClient.get<Org[]>("/api/orgs");

      const personal = boards
        .filter((b) => !b.org_id)
        .map(toBoardSummary);

      const groups = orgs.map((org) => ({
        org: { id: org.id, name: org.name },
        boards: boards
          .filter((b) => b.org_id === org.id)
          .map(toBoardSummary),
      }));

      setPersonalBoards(personal);
      setOrgGroups(groups);
      setError(null);
    } catch (err) {
      const message =
        err instanceof ApiClientError
          ? err.message
          : "Failed to load boards";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchBoards();
  }, [fetchBoards]);

  async function handleCreateBoard(name: string) {
    try {
      const board = await apiClient.post<Board>("/api/boards", { name });
      setDialogOpen(false);
      router.push(`/boards/${board.id}`);
    } catch (err) {
      const message =
        err instanceof ApiClientError
          ? err.message
          : "Failed to create board";
      setError(message);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600" />
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Boards</h1>
        <Button onClick={() => setDialogOpen(true)}>New Board</Button>
      </div>

      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <BoardList personalBoards={personalBoards} orgGroups={orgGroups} />

      <NewBoardDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onCreate={handleCreateBoard}
      />
    </div>
  );
}

function toBoardSummary(board: Board): BoardSummary {
  return {
    id: board.id,
    name: board.name,
    item_count: board.item_count,
    updated_at: board.updated_at,
  };
}
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/dashboard/
```

Expected: PASS (all 3 test files)

- [ ] **Step 8: Commit**

```bash
cd ~/code/obeya-cloud
git add components/dashboard/ app/\(dashboard\)/dashboard/ __tests__/components/dashboard/
git commit -m "feat: add dashboard page with board list, board cards, and new board dialog"
```

---

<!-- CONTINUED IN PART B: Kanban board, item detail, org pages, settings -->
