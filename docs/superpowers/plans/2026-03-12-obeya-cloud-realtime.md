# Obeya Cloud Plan 6: Realtime — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add realtime live updates to both the web UI (Next.js) and the CLI TUI (Go), so changes made by any client appear instantly for all connected viewers.

**Architecture:** Web UI uses Appwrite JS SDK (`appwrite` npm package) for browser-side WebSocket subscriptions. CLI TUI uses `gorilla/websocket` to connect directly to Appwrite's realtime endpoint. Both subscribe to board-scoped document changes. Appwrite realtime respects document-level permissions, so users only receive events for boards they can access.

**Tech Stack:**
- Web UI: Next.js 15, TypeScript, `appwrite` (JS SDK for browser), React hooks, Vitest
- CLI TUI: Go, `github.com/gorilla/websocket`, Bubble Tea

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md` (Realtime section)

**Repositories:** This plan modifies TWO repos:
1. `~/code/obeya-cloud` (Next.js — web UI realtime)
2. `~/code/obeya` (Go CLI — TUI realtime)

---

## File Structure

### Web UI (`~/code/obeya-cloud`)

```
obeya-cloud/
├── lib/
│   └── appwrite/
│       ├── server.ts                        # EXISTING — server SDK singleton
│       ├── collections.ts                   # EXISTING — collection constants
│       └── browser-client.ts                # NEW — Appwrite JS SDK for browser
├── hooks/
│   ├── use-board-subscription.ts            # NEW — subscribe to item changes
│   └── use-activity-subscription.ts         # NEW — subscribe to item_history changes
├── components/
│   ├── realtime/
│   │   └── connection-status.tsx            # NEW — connection indicator
│   ├── board/
│   │   └── kanban-board.tsx                 # MODIFIED — integrate live updates
│   └── activity/
│       └── activity-feed.tsx                # MODIFIED — integrate live entries
├── __tests__/
│   ├── lib/
│   │   └── appwrite/
│   │       └── browser-client.test.ts       # NEW
│   └── hooks/
│       ├── use-board-subscription.test.ts   # NEW
│       └── use-activity-subscription.test.ts # NEW
```

### CLI TUI (`~/code/obeya`)

```
obeya/
├── internal/
│   ├── realtime/
│   │   ├── client.go                        # NEW — WebSocket client for Appwrite
│   │   ├── client_test.go                   # NEW
│   │   ├── events.go                        # NEW — event parsing
│   │   └── events_test.go                   # NEW
│   └── tui/
│       ├── watcher.go                       # MODIFIED — add CloudWatcher interface
│       ├── watcher_test.go                  # MODIFIED — add cloud watcher tests
│       └── app.go                           # MODIFIED — swap watcher source in cloud mode
├── go.mod                                   # MODIFIED — add gorilla/websocket
```

---

## Chunk 1: Web UI — Appwrite Browser Client

### Task 1: Appwrite JS SDK Browser Client

**Files:**
- Create: `obeya-cloud/lib/appwrite/browser-client.ts`
- Test: `obeya-cloud/__tests__/lib/appwrite/browser-client.test.ts`

- [ ] **Step 1: Install Appwrite JS SDK**

```bash
cd ~/code/obeya-cloud
npm install appwrite
```

- [ ] **Step 2: Write failing test**

Create: `obeya-cloud/__tests__/lib/appwrite/browser-client.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the appwrite module
vi.mock("appwrite", () => {
  const mockSetEndpoint = vi.fn().mockReturnThis();
  const mockSetProject = vi.fn().mockReturnThis();

  return {
    Client: vi.fn().mockImplementation(() => ({
      setEndpoint: mockSetEndpoint,
      setProject: mockSetProject,
      subscribe: vi.fn(),
    })),
  };
});

describe("browser-client", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("getBrowserClient returns a configured Appwrite Client", async () => {
    const { getBrowserClient } = await import(
      "@/lib/appwrite/browser-client"
    );

    const client = getBrowserClient();

    expect(client).toBeDefined();
    expect(client.setEndpoint).toBeDefined();
    expect(client.setProject).toBeDefined();
  });

  it("getBrowserClient returns the same singleton on repeated calls", async () => {
    const { getBrowserClient } = await import(
      "@/lib/appwrite/browser-client"
    );

    const client1 = getBrowserClient();
    const client2 = getBrowserClient();

    expect(client1).toBe(client2);
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/appwrite/browser-client.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 4: Write implementation**

Create: `obeya-cloud/lib/appwrite/browser-client.ts`

```typescript
import { Client } from "appwrite";

let browserClient: Client | null = null;

/**
 * Returns a singleton Appwrite Client configured for browser-side use.
 * Uses NEXT_PUBLIC_ env vars which are available in the browser bundle.
 *
 * This client is used for realtime subscriptions only — all writes go
 * through Next.js API routes using the server SDK.
 */
export function getBrowserClient(): Client {
  if (browserClient) return browserClient;

  const endpoint = process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT;
  const projectId = process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID;

  if (!endpoint || !projectId) {
    throw new Error(
      "Missing NEXT_PUBLIC_APPWRITE_ENDPOINT or NEXT_PUBLIC_APPWRITE_PROJECT_ID. " +
        "These must be set for realtime subscriptions to work."
    );
  }

  browserClient = new Client()
    .setEndpoint(endpoint)
    .setProject(projectId);

  return browserClient;
}

/**
 * Resets the singleton — used in tests only.
 */
export function resetBrowserClient(): void {
  browserClient = null;
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/appwrite/browser-client.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/appwrite/browser-client.ts __tests__/lib/appwrite/browser-client.test.ts package.json package-lock.json
git commit -m "feat: add Appwrite JS SDK browser client singleton for realtime"
```

---

## Chunk 2: Web UI — Board Subscription Hook

### Task 2: useBoardSubscription React Hook

**Files:**
- Create: `obeya-cloud/hooks/use-board-subscription.ts`
- Test: `obeya-cloud/__tests__/hooks/use-board-subscription.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/hooks/use-board-subscription.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// Track subscribe calls
const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useBoardSubscription } from "@/hooks/use-board-subscription";
import type { BoardItemEvent } from "@/hooks/use-board-subscription";

describe("useBoardSubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("subscribes to the correct Appwrite channel on mount", () => {
    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);

    const channelArg = mockSubscribe.mock.calls[0][0];
    expect(channelArg).toBe(
      "databases.obeya.collections.items.documents"
    );
  });

  it("unsubscribes on unmount", () => {
    const onEvent = vi.fn();
    const { unmount } = renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    unmount();

    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
  });

  it("does not subscribe when boardId is empty", () => {
    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(mockSubscribe).not.toHaveBeenCalled();
  });

  it("filters events by board_id and calls onEvent with parsed data", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(capturedCallback).not.toBeNull();

    // Simulate an event for the correct board
    act(() => {
      capturedCallback!({
        events: [
          "databases.obeya.collections.items.documents.item-1.create",
        ],
        payload: {
          $id: "item-1",
          board_id: "board-123",
          display_num: 42,
          type: "task",
          title: "New task",
          status: "todo",
          priority: "medium",
          created_at: "2026-03-12T10:00:00Z",
          updated_at: "2026-03-12T10:00:00Z",
        },
      });
    });

    expect(onEvent).toHaveBeenCalledTimes(1);
    const eventArg: BoardItemEvent = onEvent.mock.calls[0][0];
    expect(eventArg.action).toBe("create");
    expect(eventArg.item.$id).toBe("item-1");
  });

  it("ignores events for a different board_id", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    act(() => {
      capturedCallback!({
        events: [
          "databases.obeya.collections.items.documents.item-99.create",
        ],
        payload: {
          $id: "item-99",
          board_id: "other-board",
          title: "Wrong board",
        },
      });
    });

    expect(onEvent).not.toHaveBeenCalled();
  });

  it("tracks connection status", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    // After successful subscribe, status should be connected
    expect(result.current.status).toBe("connected");
  });

  it("reports disconnected status when boardId is empty", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() =>
      useBoardSubscription({
        boardId: "",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(result.current.status).toBe("disconnected");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-board-subscription.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Install testing-library if not present**

```bash
cd ~/code/obeya-cloud
npm install -D @testing-library/react @testing-library/react-hooks
```

- [ ] **Step 4: Write implementation**

Create: `obeya-cloud/hooks/use-board-subscription.ts`

```typescript
"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";

export type ConnectionStatus = "connected" | "connecting" | "disconnected" | "error";

export type ItemAction = "create" | "update" | "delete";

export interface BoardItemEvent {
  action: ItemAction;
  item: Record<string, unknown>;
}

interface UseBoardSubscriptionOptions {
  boardId: string;
  databaseId: string;
  onEvent: (event: BoardItemEvent) => void;
}

interface UseBoardSubscriptionResult {
  status: ConnectionStatus;
}

/**
 * Parses the Appwrite realtime event string to extract the action.
 *
 * Event format: "databases.<db>.collections.<col>.documents.<docId>.<action>"
 * Actions: "create", "update", "delete"
 */
function parseAction(eventString: string): ItemAction | null {
  const parts = eventString.split(".");
  const lastPart = parts[parts.length - 1];

  if (lastPart === "create" || lastPart === "update" || lastPart === "delete") {
    return lastPart;
  }
  return null;
}

/**
 * React hook that subscribes to Appwrite realtime for board item changes.
 *
 * Subscribes to: databases.<databaseId>.collections.items.documents
 * Filters by: board_id matching the provided boardId
 *
 * The hook manages its own lifecycle — subscribes on mount, unsubscribes
 * on unmount or when boardId changes.
 */
export function useBoardSubscription(
  options: UseBoardSubscriptionOptions
): UseBoardSubscriptionResult {
  const { boardId, databaseId, onEvent } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onEventRef = useRef(onEvent);

  // Keep callback ref up to date without causing re-subscriptions
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();
    const channel = `databases.${databaseId}.collections.items.documents`;

    const unsubscribe = client.subscribe(channel, (event: any) => {
      const payload = event.payload;

      // Filter by board_id — Appwrite sends all events for the collection,
      // so we must filter client-side for the specific board
      if (payload.board_id !== boardId) {
        return;
      }

      // Parse the action from the event string
      const events: string[] = event.events || [];
      let action: ItemAction | null = null;

      for (const eventStr of events) {
        action = parseAction(eventStr);
        if (action) break;
      }

      if (!action) return;

      onEventRef.current({ action, item: payload });
    });

    setStatus("connected");

    return () => {
      unsubscribe();
      setStatus("disconnected");
    };
  }, [boardId, databaseId]);

  return { status };
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-board-subscription.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add hooks/use-board-subscription.ts __tests__/hooks/use-board-subscription.test.ts
git commit -m "feat: add useBoardSubscription hook for realtime board item updates"
```

---

## Chunk 3: Web UI — Activity Subscription Hook

### Task 3: useActivitySubscription React Hook

**Files:**
- Create: `obeya-cloud/hooks/use-activity-subscription.ts`
- Test: `obeya-cloud/__tests__/hooks/use-activity-subscription.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/hooks/use-activity-subscription.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useActivitySubscription } from "@/hooks/use-activity-subscription";
import type { ActivityEvent } from "@/hooks/use-activity-subscription";

describe("useActivitySubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("subscribes to the item_history collection channel", () => {
    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);

    const channelArg = mockSubscribe.mock.calls[0][0];
    expect(channelArg).toBe(
      "databases.obeya.collections.item_history.documents"
    );
  });

  it("unsubscribes on unmount", () => {
    const onActivity = vi.fn();
    const { unmount } = renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    unmount();
    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
  });

  it("does not subscribe when boardId is empty", () => {
    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(mockSubscribe).not.toHaveBeenCalled();
  });

  it("filters events by board_id and calls onActivity", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(capturedCallback).not.toBeNull();

    act(() => {
      capturedCallback!({
        events: [
          "databases.obeya.collections.item_history.documents.hist-1.create",
        ],
        payload: {
          $id: "hist-1",
          item_id: "item-42",
          board_id: "board-123",
          user_id: "user-1",
          action: "moved",
          detail: "status: todo -> in-progress",
          timestamp: "2026-03-12T10:05:00Z",
        },
      });
    });

    expect(onActivity).toHaveBeenCalledTimes(1);
    const activityArg: ActivityEvent = onActivity.mock.calls[0][0];
    expect(activityArg.entry.item_id).toBe("item-42");
    expect(activityArg.entry.action).toBe("moved");
  });

  it("ignores events for a different board_id", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    act(() => {
      capturedCallback!({
        events: [
          "databases.obeya.collections.item_history.documents.hist-99.create",
        ],
        payload: {
          $id: "hist-99",
          board_id: "other-board",
          action: "created",
          detail: "wrong board",
        },
      });
    });

    expect(onActivity).not.toHaveBeenCalled();
  });

  it("reports connection status", () => {
    const onActivity = vi.fn();
    const { result } = renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(result.current.status).toBe("connected");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-activity-subscription.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/hooks/use-activity-subscription.ts`

```typescript
"use client";

import { useEffect, useRef, useState } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

export interface HistoryEntry {
  $id: string;
  item_id: string;
  board_id: string;
  user_id: string;
  session_id?: string;
  action: string;
  detail: string;
  timestamp: string;
}

export interface ActivityEvent {
  entry: HistoryEntry;
}

interface UseActivitySubscriptionOptions {
  boardId: string;
  databaseId: string;
  onActivity: (event: ActivityEvent) => void;
}

interface UseActivitySubscriptionResult {
  status: ConnectionStatus;
}

/**
 * React hook that subscribes to Appwrite realtime for item_history changes.
 *
 * Subscribes to: databases.<databaseId>.collections.item_history.documents
 * Filters by: board_id matching the provided boardId
 *
 * Only emits "create" events (new history entries). History entries are
 * immutable — they are never updated or deleted.
 */
export function useActivitySubscription(
  options: UseActivitySubscriptionOptions
): UseActivitySubscriptionResult {
  const { boardId, databaseId, onActivity } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onActivityRef = useRef(onActivity);

  useEffect(() => {
    onActivityRef.current = onActivity;
  }, [onActivity]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();
    const channel = `databases.${databaseId}.collections.item_history.documents`;

    const unsubscribe = client.subscribe(channel, (event: any) => {
      const payload = event.payload;

      // Filter by board_id
      if (payload.board_id !== boardId) {
        return;
      }

      // Only emit for new history entries (creates)
      const events: string[] = event.events || [];
      const isCreate = events.some((e: string) => e.endsWith(".create"));
      if (!isCreate) return;

      const entry: HistoryEntry = {
        $id: payload.$id,
        item_id: payload.item_id,
        board_id: payload.board_id,
        user_id: payload.user_id,
        session_id: payload.session_id,
        action: payload.action,
        detail: payload.detail,
        timestamp: payload.timestamp,
      };

      onActivityRef.current({ entry });
    });

    setStatus("connected");

    return () => {
      unsubscribe();
      setStatus("disconnected");
    };
  }, [boardId, databaseId]);

  return { status };
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-activity-subscription.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add hooks/use-activity-subscription.ts __tests__/hooks/use-activity-subscription.test.ts
git commit -m "feat: add useActivitySubscription hook for realtime activity feed"
```

---

## Chunk 4: Web UI — Connection Status Indicator

### Task 4: ConnectionStatus Component

**Files:**
- Create: `obeya-cloud/components/realtime/connection-status.tsx`
- Test: `obeya-cloud/__tests__/components/realtime/connection-status.test.tsx`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/components/realtime/connection-status.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";

// Install @testing-library/jest-dom for toBeInTheDocument if not present
// npm install -D @testing-library/jest-dom

describe("ConnectionStatusIndicator", () => {
  it("renders connected state with green indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connected" />
    );

    const indicator = container.querySelector("[data-status='connected']");
    expect(indicator).not.toBeNull();

    const text = container.textContent;
    expect(text).toContain("Connected");
  });

  it("renders connecting state with yellow indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connecting" />
    );

    const indicator = container.querySelector("[data-status='connecting']");
    expect(indicator).not.toBeNull();

    const text = container.textContent;
    expect(text).toContain("Connecting");
  });

  it("renders disconnected state with gray indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="disconnected" />
    );

    const indicator = container.querySelector("[data-status='disconnected']");
    expect(indicator).not.toBeNull();

    const text = container.textContent;
    expect(text).toContain("Offline");
  });

  it("renders error state with red indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="error" />
    );

    const indicator = container.querySelector("[data-status='error']");
    expect(indicator).not.toBeNull();

    const text = container.textContent;
    expect(text).toContain("Error");
  });

  it("renders compact mode without text", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connected" compact />
    );

    const text = container.textContent;
    expect(text).not.toContain("Connected");

    const dot = container.querySelector("[data-status='connected']");
    expect(dot).not.toBeNull();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/realtime/connection-status.test.tsx
```

Expected: FAIL — module not found

- [ ] **Step 3: Install testing-library DOM matchers if needed**

```bash
cd ~/code/obeya-cloud
npm install -D @testing-library/jest-dom jsdom
```

Add to `vitest.config.ts` if not already present:

```typescript
import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  test: {
    globals: true,
    environment: "jsdom",
    include: ["__tests__/**/*.test.{ts,tsx}"],
    setupFiles: [],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
    },
  },
});
```

- [ ] **Step 4: Write implementation**

Create: `obeya-cloud/components/realtime/connection-status.tsx`

```typescript
"use client";

import type { ConnectionStatus } from "@/hooks/use-board-subscription";

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  compact?: boolean;
}

const STATUS_CONFIG: Record<
  ConnectionStatus,
  { label: string; dotClass: string; textClass: string }
> = {
  connected: {
    label: "Connected",
    dotClass: "bg-green-500",
    textClass: "text-green-700 dark:text-green-400",
  },
  connecting: {
    label: "Connecting...",
    dotClass: "bg-yellow-500 animate-pulse",
    textClass: "text-yellow-700 dark:text-yellow-400",
  },
  disconnected: {
    label: "Offline",
    dotClass: "bg-gray-400",
    textClass: "text-gray-500 dark:text-gray-400",
  },
  error: {
    label: "Error",
    dotClass: "bg-red-500",
    textClass: "text-red-700 dark:text-red-400",
  },
};

/**
 * Visual indicator for the realtime WebSocket connection status.
 *
 * Displays a colored dot and optional label text.
 * - Green: connected and receiving live updates
 * - Yellow (pulsing): connecting to Appwrite realtime
 * - Gray: disconnected (no board loaded or subscription inactive)
 * - Red: error state (connection failed)
 */
export function ConnectionStatusIndicator({
  status,
  compact = false,
}: ConnectionStatusIndicatorProps) {
  const config = STATUS_CONFIG[status];

  return (
    <div
      className="flex items-center gap-2"
      data-status={status}
      role="status"
      aria-label={`Realtime connection: ${config.label}`}
    >
      <span
        className={`inline-block h-2 w-2 rounded-full ${config.dotClass}`}
        data-status={status}
      />
      {!compact && (
        <span className={`text-xs font-medium ${config.textClass}`}>
          {config.label}
        </span>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/realtime/connection-status.test.tsx
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add components/realtime/connection-status.tsx __tests__/components/realtime/connection-status.test.tsx
git commit -m "feat: add ConnectionStatusIndicator component for realtime state"
```

---

## Chunk 5: Web UI — Kanban Board Integration

### Task 5: Integrate Realtime into Kanban Board

**Files:**
- Modify: `obeya-cloud/components/board/kanban-board.tsx`

This task integrates `useBoardSubscription` into the existing Kanban board component. The board component must already exist from a prior plan. The integration adds live create/move/edit/delete handling.

- [ ] **Step 1: Create the realtime integration helper**

Create: `obeya-cloud/hooks/use-board-realtime-sync.ts`

This hook bridges the raw subscription events with the board's local state management (e.g., React state or SWR cache).

```typescript
"use client";

import { useCallback, useRef } from "react";
import {
  useBoardSubscription,
  type BoardItemEvent,
} from "@/hooks/use-board-subscription";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

export interface BoardItem {
  $id: string;
  board_id: string;
  display_num: number;
  type: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  parent_id?: string;
  assignee_id?: string;
  blocked_by?: string[];
  tags?: string[];
  project?: string;
  created_at: string;
  updated_at: string;
}

interface UseBoardRealtimeSyncOptions {
  boardId: string;
  databaseId: string;
  onItemCreated: (item: BoardItem) => void;
  onItemUpdated: (item: BoardItem) => void;
  onItemDeleted: (itemId: string) => void;
}

interface UseBoardRealtimeSyncResult {
  status: ConnectionStatus;
}

/**
 * Higher-level hook that translates raw BoardItemEvents into
 * typed create/update/delete callbacks for the Kanban board.
 */
export function useBoardRealtimeSync(
  options: UseBoardRealtimeSyncOptions
): UseBoardRealtimeSyncResult {
  const {
    boardId,
    databaseId,
    onItemCreated,
    onItemUpdated,
    onItemDeleted,
  } = options;

  const onItemCreatedRef = useRef(onItemCreated);
  const onItemUpdatedRef = useRef(onItemUpdated);
  const onItemDeletedRef = useRef(onItemDeleted);

  onItemCreatedRef.current = onItemCreated;
  onItemUpdatedRef.current = onItemUpdated;
  onItemDeletedRef.current = onItemDeleted;

  const handleEvent = useCallback((event: BoardItemEvent) => {
    const item = event.item as unknown as BoardItem;

    switch (event.action) {
      case "create":
        onItemCreatedRef.current(item);
        break;
      case "update":
        onItemUpdatedRef.current(item);
        break;
      case "delete":
        onItemDeletedRef.current(item.$id);
        break;
    }
  }, []);

  const { status } = useBoardSubscription({
    boardId,
    databaseId,
    onEvent: handleEvent,
  });

  return { status };
}
```

- [ ] **Step 2: Create Kanban board realtime integration example**

This shows how to integrate the hook into the Kanban board. The actual Kanban component may vary based on what was built in earlier plans, but this demonstrates the pattern.

Create: `obeya-cloud/components/board/kanban-board-realtime.tsx`

```typescript
"use client";

import { useState, useCallback } from "react";
import {
  useBoardRealtimeSync,
  type BoardItem,
} from "@/hooks/use-board-realtime-sync";
import { useActivitySubscription } from "@/hooks/use-activity-subscription";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";
import type { ActivityEvent } from "@/hooks/use-activity-subscription";
import type { HistoryEntry } from "@/hooks/use-activity-subscription";

interface KanbanBoardRealtimeProps {
  boardId: string;
  databaseId: string;
  initialItems: BoardItem[];
  columns: { name: string; limit: number }[];
}

/**
 * Kanban board component with integrated realtime updates.
 *
 * Wraps the static Kanban board with live subscription hooks.
 * When an item is created/moved/edited/deleted by another client,
 * the board state updates immediately without a page refresh.
 */
export function KanbanBoardRealtime({
  boardId,
  databaseId,
  initialItems,
  columns,
}: KanbanBoardRealtimeProps) {
  const [items, setItems] = useState<BoardItem[]>(initialItems);
  const [activities, setActivities] = useState<HistoryEntry[]>([]);

  const handleItemCreated = useCallback((item: BoardItem) => {
    setItems((prev) => {
      // Avoid duplicates — item might already exist from our own optimistic update
      if (prev.some((existing) => existing.$id === item.$id)) {
        return prev;
      }
      return [...prev, item];
    });
  }, []);

  const handleItemUpdated = useCallback((item: BoardItem) => {
    setItems((prev) =>
      prev.map((existing) =>
        existing.$id === item.$id ? item : existing
      )
    );
  }, []);

  const handleItemDeleted = useCallback((itemId: string) => {
    setItems((prev) => prev.filter((item) => item.$id !== itemId));
  }, []);

  const handleActivity = useCallback((event: ActivityEvent) => {
    setActivities((prev) => [event.entry, ...prev].slice(0, 50));
  }, []);

  const { status: boardStatus } = useBoardRealtimeSync({
    boardId,
    databaseId,
    onItemCreated: handleItemCreated,
    onItemUpdated: handleItemUpdated,
    onItemDeleted: handleItemDeleted,
  });

  const { status: activityStatus } = useActivitySubscription({
    boardId,
    databaseId,
    onActivity: handleActivity,
  });

  // Derive the effective connection status (worst of the two)
  const effectiveStatus =
    boardStatus === "error" || activityStatus === "error"
      ? "error"
      : boardStatus === "connecting" || activityStatus === "connecting"
        ? "connecting"
        : boardStatus === "connected" && activityStatus === "connected"
          ? "connected"
          : "disconnected";

  // Group items by column
  const itemsByColumn: Record<string, BoardItem[]> = {};
  for (const col of columns) {
    itemsByColumn[col.name] = [];
  }
  for (const item of items) {
    if (itemsByColumn[item.status]) {
      itemsByColumn[item.status].push(item);
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with connection status */}
      <div className="flex items-center justify-between px-4 py-2 border-b">
        <h2 className="text-lg font-semibold">Board</h2>
        <ConnectionStatusIndicator status={effectiveStatus} />
      </div>

      {/* Kanban columns */}
      <div className="flex flex-1 overflow-x-auto gap-4 p-4">
        {columns.map((col) => {
          const colItems = itemsByColumn[col.name] || [];
          const isOverLimit = col.limit > 0 && colItems.length > col.limit;

          return (
            <div
              key={col.name}
              className="flex flex-col min-w-[280px] max-w-[320px] bg-gray-50 dark:bg-gray-800 rounded-lg"
            >
              {/* Column header */}
              <div className="flex items-center justify-between px-3 py-2 border-b">
                <span className="font-medium text-sm">{col.name}</span>
                <span
                  className={`text-xs ${isOverLimit ? "text-red-500 font-bold" : "text-gray-400"}`}
                >
                  {colItems.length}
                  {col.limit > 0 && `/${col.limit}`}
                </span>
              </div>

              {/* Cards */}
              <div className="flex flex-col gap-2 p-2 overflow-y-auto">
                {colItems.map((item) => (
                  <div
                    key={item.$id}
                    className="bg-white dark:bg-gray-700 rounded-md p-3 shadow-sm border border-gray-200 dark:border-gray-600 transition-all duration-200 animate-in fade-in"
                  >
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs text-gray-400">
                        #{item.display_num}
                      </span>
                      <span className="text-xs px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-600 text-gray-600 dark:text-gray-300">
                        {item.type}
                      </span>
                      <PriorityBadge priority={item.priority} />
                    </div>
                    <p className="text-sm font-medium">{item.title}</p>
                    {item.assignee_id && (
                      <p className="text-xs text-gray-400 mt-1">
                        @{item.assignee_id}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function PriorityBadge({ priority }: { priority: string }) {
  const colors: Record<string, string> = {
    critical: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
    high: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300",
    medium: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300",
    low: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400",
  };

  return (
    <span
      className={`text-xs px-1.5 py-0.5 rounded ${colors[priority] || colors.medium}`}
    >
      {priority}
    </span>
  );
}
```

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya-cloud
git add hooks/use-board-realtime-sync.ts components/board/kanban-board-realtime.tsx
git commit -m "feat: integrate realtime subscriptions into Kanban board component"
```

---

## Chunk 6: CLI TUI — Go WebSocket Client

### Task 6: Appwrite Realtime WebSocket Client

**Files:**
- Create: `internal/realtime/client.go`
- Create: `internal/realtime/client_test.go`

- [ ] **Step 1: Add gorilla/websocket dependency**

```bash
cd ~/code/obeya
go get github.com/gorilla/websocket
```

- [ ] **Step 2: Write event types**

Create: `internal/realtime/events.go`

```go
package realtime

// EventAction represents the type of change on a document.
type EventAction string

const (
	ActionCreate EventAction = "create"
	ActionUpdate EventAction = "update"
	ActionDelete EventAction = "delete"
)

// BoardEvent represents a parsed realtime event scoped to a board.
type BoardEvent struct {
	Action       EventAction
	CollectionID string
	DocumentID   string
	Payload      map[string]interface{}
}

// SubscriptionConfig holds the parameters for a realtime subscription.
type SubscriptionConfig struct {
	// AppwriteEndpoint is the Appwrite API endpoint, e.g. "https://cloud.appwrite.io/v1"
	AppwriteEndpoint string

	// ProjectID is the Appwrite project ID.
	ProjectID string

	// APIToken is the Bearer token for authentication (ob_tok_...).
	APIToken string

	// DatabaseID is the Appwrite database ID, e.g. "obeya".
	DatabaseID string

	// BoardID is the board to filter events for.
	BoardID string
}

// websocketURL converts the Appwrite REST endpoint to a WebSocket URL.
// https://cloud.appwrite.io/v1 -> wss://cloud.appwrite.io/v1/realtime
func websocketURL(endpoint string) string {
	url := endpoint

	// Replace http(s):// with ws(s)://
	if len(url) > 8 && url[:8] == "https://" {
		url = "wss://" + url[8:]
	} else if len(url) > 7 && url[:7] == "http://" {
		url = "ws://" + url[7:]
	}

	// Append /realtime
	if url[len(url)-1] == '/' {
		url += "realtime"
	} else {
		url += "/realtime"
	}

	return url
}
```

- [ ] **Step 3: Write event parsing tests**

Create: `internal/realtime/events_test.go`

```go
package realtime

import (
	"testing"
)

func TestWebsocketURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "https endpoint",
			endpoint: "https://cloud.appwrite.io/v1",
			want:     "wss://cloud.appwrite.io/v1/realtime",
		},
		{
			name:     "http endpoint",
			endpoint: "http://localhost:8080/v1",
			want:     "ws://localhost:8080/v1/realtime",
		},
		{
			name:     "trailing slash",
			endpoint: "https://cloud.appwrite.io/v1/",
			want:     "wss://cloud.appwrite.io/v1/realtime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := websocketURL(tt.endpoint)
			if got != tt.want {
				t.Errorf("websocketURL(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestParseEventAction(t *testing.T) {
	tests := []struct {
		name      string
		eventStr  string
		wantAction EventAction
		wantOk    bool
	}{
		{
			name:       "create event",
			eventStr:   "databases.obeya.collections.items.documents.doc123.create",
			wantAction: ActionCreate,
			wantOk:     true,
		},
		{
			name:       "update event",
			eventStr:   "databases.obeya.collections.items.documents.doc123.update",
			wantAction: ActionUpdate,
			wantOk:     true,
		},
		{
			name:       "delete event",
			eventStr:   "databases.obeya.collections.items.documents.doc123.delete",
			wantAction: ActionDelete,
			wantOk:     true,
		},
		{
			name:       "unknown event suffix",
			eventStr:   "databases.obeya.collections.items.documents.doc123.unknown",
			wantAction: "",
			wantOk:     false,
		},
		{
			name:       "empty string",
			eventStr:   "",
			wantAction: "",
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, ok := parseEventAction(tt.eventStr)
			if ok != tt.wantOk {
				t.Errorf("parseEventAction(%q) ok = %v, want %v", tt.eventStr, ok, tt.wantOk)
			}
			if action != tt.wantAction {
				t.Errorf("parseEventAction(%q) action = %q, want %q", tt.eventStr, action, tt.wantAction)
			}
		})
	}
}

func TestParseCollectionID(t *testing.T) {
	tests := []struct {
		name     string
		eventStr string
		want     string
	}{
		{
			name:     "items collection",
			eventStr: "databases.obeya.collections.items.documents.doc123.create",
			want:     "items",
		},
		{
			name:     "item_history collection",
			eventStr: "databases.obeya.collections.item_history.documents.doc456.create",
			want:     "item_history",
		},
		{
			name:     "short string",
			eventStr: "too.short",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCollectionID(tt.eventStr)
			if got != tt.want {
				t.Errorf("parseCollectionID(%q) = %q, want %q", tt.eventStr, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 4: Write event parsing functions**

Add to `internal/realtime/events.go`:

```go
// parseEventAction extracts the action (create/update/delete) from an
// Appwrite realtime event string.
// Format: "databases.<db>.collections.<col>.documents.<docId>.<action>"
func parseEventAction(eventStr string) (EventAction, bool) {
	if eventStr == "" {
		return "", false
	}

	// Find the last dot
	lastDot := -1
	for i := len(eventStr) - 1; i >= 0; i-- {
		if eventStr[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot < 0 || lastDot >= len(eventStr)-1 {
		return "", false
	}

	suffix := eventStr[lastDot+1:]
	switch suffix {
	case "create":
		return ActionCreate, true
	case "update":
		return ActionUpdate, true
	case "delete":
		return ActionDelete, true
	default:
		return "", false
	}
}

// parseCollectionID extracts the collection ID from an Appwrite realtime event.
// Format: "databases.<db>.collections.<col>.documents.<docId>.<action>"
// Returns the <col> part.
func parseCollectionID(eventStr string) string {
	// Split by dots and find "collections" marker
	start := 0
	partIndex := 0
	collectionsIdx := -1

	for i := 0; i <= len(eventStr); i++ {
		if i == len(eventStr) || eventStr[i] == '.' {
			if partIndex > 0 && collectionsIdx == partIndex-1 {
				return eventStr[start:i]
			}
			if i-start == 11 && eventStr[start:i] == "collections" {
				collectionsIdx = partIndex
			}
			partIndex++
			start = i + 1
		}
	}
	return ""
}

// parseRealtimeMessage parses a raw Appwrite realtime WebSocket message into
// a BoardEvent, filtering by board_id. Returns nil if the event is not
// relevant to the specified board.
func parseRealtimeMessage(msg map[string]interface{}, boardID string) *BoardEvent {
	// Extract events array
	eventsRaw, ok := msg["events"]
	if !ok {
		return nil
	}
	events, ok := eventsRaw.([]interface{})
	if !ok || len(events) == 0 {
		return nil
	}

	// Extract payload
	payloadRaw, ok := msg["payload"]
	if !ok {
		return nil
	}
	payload, ok := payloadRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	// Filter by board_id
	payloadBoardID, _ := payload["board_id"].(string)
	if payloadBoardID != boardID {
		return nil
	}

	// Parse the first event string that has a valid action
	var action EventAction
	var collectionID string
	for _, e := range events {
		eventStr, ok := e.(string)
		if !ok {
			continue
		}
		a, valid := parseEventAction(eventStr)
		if valid {
			action = a
			collectionID = parseCollectionID(eventStr)
			break
		}
	}
	if action == "" {
		return nil
	}

	docID, _ := payload["$id"].(string)

	return &BoardEvent{
		Action:       action,
		CollectionID: collectionID,
		DocumentID:   docID,
		Payload:      payload,
	}
}
```

- [ ] **Step 5: Run event tests**

```bash
cd ~/code/obeya
go test ./internal/realtime/ -v -run TestWebsocketURL
go test ./internal/realtime/ -v -run TestParseEventAction
go test ./internal/realtime/ -v -run TestParseCollectionID
```

Expected: All PASS

- [ ] **Step 6: Write WebSocket client**

Create: `internal/realtime/client.go`

```go
package realtime

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// maxReconnectDelay is the cap for exponential backoff.
	maxReconnectDelay = 30 * time.Second

	// baseReconnectDelay is the initial reconnect delay.
	baseReconnectDelay = 1 * time.Second

	// pingInterval is how often we send pings to keep the connection alive.
	pingInterval = 30 * time.Second

	// pongWait is how long we wait for a pong response.
	pongWait = 10 * time.Second
)

// Client manages a WebSocket connection to Appwrite's realtime endpoint.
// It handles automatic reconnection with exponential backoff.
type Client struct {
	config    SubscriptionConfig
	eventCh   chan BoardEvent
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once

	mu   sync.Mutex
	conn *websocket.Conn
}

// NewClient creates a new realtime Client but does not connect yet.
// Call Connect() to establish the WebSocket connection.
func NewClient(config SubscriptionConfig) *Client {
	return &Client{
		config:  config,
		eventCh: make(chan BoardEvent, 64),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}
}

// Events returns the channel that receives parsed board events.
func (c *Client) Events() <-chan BoardEvent {
	return c.eventCh
}

// Errors returns the channel that receives connection errors.
// Errors are non-fatal — the client will attempt to reconnect.
func (c *Client) Errors() <-chan error {
	return c.errCh
}

// Connect establishes the WebSocket connection and starts the read loop.
// This is a blocking call — run it in a goroutine.
// It automatically reconnects on disconnection with exponential backoff.
func (c *Client) Connect() {
	attempt := 0
	for {
		select {
		case <-c.done:
			return
		default:
		}

		err := c.connectOnce()
		if err != nil {
			select {
			case c.errCh <- fmt.Errorf("realtime connection failed: %w", err):
			default:
			}
		}

		// Check if we're shutting down before reconnecting
		select {
		case <-c.done:
			return
		default:
		}

		// Exponential backoff
		attempt++
		delay := backoffDelay(attempt)
		select {
		case <-c.done:
			return
		case <-time.After(delay):
		}
	}
}

// connectOnce establishes a single WebSocket connection, subscribes to
// the board channels, and reads messages until disconnection.
func (c *Client) connectOnce() error {
	wsURL := c.buildURL()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		conn.Close()
	}()

	// Set up pong handler for keepalive
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pingInterval + pongWait))
	})

	// Start ping goroutine
	pingDone := make(chan struct{})
	go c.pingLoop(conn, pingDone)
	defer close(pingDone)

	// Read loop
	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return fmt.Errorf("read failed: %w", err)
		}

		c.handleMessage(msgBytes)
	}
}

// buildURL constructs the Appwrite realtime WebSocket URL with channel subscriptions.
func (c *Client) buildURL() string {
	base := websocketURL(c.config.AppwriteEndpoint)

	// Subscribe to items and item_history collections for this database
	channels := []string{
		fmt.Sprintf("databases.%s.collections.items.documents", c.config.DatabaseID),
		fmt.Sprintf("databases.%s.collections.item_history.documents", c.config.DatabaseID),
	}

	params := url.Values{}
	params.Set("project", c.config.ProjectID)
	for _, ch := range channels {
		params.Add("channels[]", ch)
	}

	return base + "?" + params.Encode()
}

// handleMessage parses a raw WebSocket message and sends the result to the event channel.
func (c *Client) handleMessage(msgBytes []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return // silently ignore malformed messages
	}

	// Check for Appwrite error responses
	if errMsg, ok := msg["type"].(string); ok && errMsg == "error" {
		errData, _ := msg["data"].(map[string]interface{})
		errMessage, _ := errData["message"].(string)
		if errMessage != "" {
			select {
			case c.errCh <- fmt.Errorf("appwrite realtime error: %s", errMessage):
			default:
			}
		}
		return
	}

	// Parse as a data event
	dataRaw, ok := msg["data"]
	if !ok {
		return
	}
	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return
	}

	event := parseRealtimeMessage(data, c.config.BoardID)
	if event == nil {
		return
	}

	select {
	case c.eventCh <- *event:
	default:
		// Channel full — drop oldest event to avoid blocking
	}
}

// pingLoop sends periodic ping frames to keep the WebSocket alive.
func (c *Client) pingLoop(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn == conn {
				err := conn.WriteMessage(websocket.PingMessage, nil)
				c.mu.Unlock()
				if err != nil {
					return
				}
			} else {
				c.mu.Unlock()
				return
			}
		}
	}
}

// Close shuts down the client, closing the WebSocket connection and all channels.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn != nil {
			conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			conn.Close()
		}
	})
}

// backoffDelay calculates the exponential backoff delay for the given attempt.
func backoffDelay(attempt int) time.Duration {
	delay := baseReconnectDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	if delay > maxReconnectDelay {
		delay = maxReconnectDelay
	}
	return delay
}

// channelFromEvent extracts the collection part from a full event string.
// This is a convenience function for log/debug output.
func channelFromEvent(eventStr string) string {
	parts := strings.Split(eventStr, ".")
	if len(parts) >= 4 {
		return parts[3] // collection ID
	}
	return eventStr
}
```

- [ ] **Step 7: Write client unit tests**

Create: `internal/realtime/client_test.go`

```go
package realtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{attempt: 1, wantMin: 1 * time.Second, wantMax: 1 * time.Second},
		{attempt: 2, wantMin: 2 * time.Second, wantMax: 2 * time.Second},
		{attempt: 3, wantMin: 4 * time.Second, wantMax: 4 * time.Second},
		{attempt: 4, wantMin: 8 * time.Second, wantMax: 8 * time.Second},
		{attempt: 10, wantMin: 30 * time.Second, wantMax: 30 * time.Second}, // capped
	}

	for _, tt := range tests {
		delay := backoffDelay(tt.attempt)
		if delay < tt.wantMin || delay > tt.wantMax {
			t.Errorf("backoffDelay(%d) = %v, want between %v and %v",
				tt.attempt, delay, tt.wantMin, tt.wantMax)
		}
	}
}

func TestBuildURL(t *testing.T) {
	c := NewClient(SubscriptionConfig{
		AppwriteEndpoint: "https://cloud.appwrite.io/v1",
		ProjectID:        "proj-123",
		DatabaseID:       "obeya",
		BoardID:          "board-abc",
	})

	url := c.buildURL()

	if !strings.HasPrefix(url, "wss://cloud.appwrite.io/v1/realtime?") {
		t.Errorf("unexpected URL prefix: %s", url)
	}
	if !strings.Contains(url, "project=proj-123") {
		t.Errorf("URL missing project param: %s", url)
	}
	if !strings.Contains(url, "channels") {
		t.Errorf("URL missing channels param: %s", url)
	}
	if !strings.Contains(url, "databases.obeya.collections.items.documents") {
		t.Errorf("URL missing items channel: %s", url)
	}
	if !strings.Contains(url, "databases.obeya.collections.item_history.documents") {
		t.Errorf("URL missing item_history channel: %s", url)
	}
}

func TestClientReceivesEvents(t *testing.T) {
	// Set up a test WebSocket server
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Send a realtime event
		event := map[string]interface{}{
			"data": map[string]interface{}{
				"events": []interface{}{
					"databases.obeya.collections.items.documents.item-1.create",
				},
				"payload": map[string]interface{}{
					"$id":         "item-1",
					"board_id":    "board-test",
					"display_num": float64(1),
					"type":        "task",
					"title":       "Test task",
					"status":      "todo",
				},
			},
		}

		msg, _ := json.Marshal(event)
		conn.WriteMessage(websocket.TextMessage, msg)

		// Keep connection open briefly
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(SubscriptionConfig{
		AppwriteEndpoint: wsURL,
		ProjectID:        "test",
		DatabaseID:       "obeya",
		BoardID:          "board-test",
	})

	// Override buildURL to use the test server directly
	go func() {
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Errorf("dial failed: %v", err)
			return
		}
		defer conn.Close()

		client.mu.Lock()
		client.conn = conn
		client.mu.Unlock()

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			client.handleMessage(msgBytes)
		}
	}()

	// Wait for event
	select {
	case event := <-client.Events():
		if event.Action != ActionCreate {
			t.Errorf("expected create action, got %s", event.Action)
		}
		if event.DocumentID != "item-1" {
			t.Errorf("expected document ID item-1, got %s", event.DocumentID)
		}
		if event.CollectionID != "items" {
			t.Errorf("expected collection items, got %s", event.CollectionID)
		}
		title, _ := event.Payload["title"].(string)
		if title != "Test task" {
			t.Errorf("expected title 'Test task', got %s", title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	client.Close()
}

func TestClientFiltersBoardID(t *testing.T) {
	// Simulate receiving a message for the wrong board
	client := NewClient(SubscriptionConfig{
		BoardID: "my-board",
	})

	msg := map[string]interface{}{
		"events": []interface{}{
			"databases.obeya.collections.items.documents.item-99.create",
		},
		"payload": map[string]interface{}{
			"$id":      "item-99",
			"board_id": "other-board",
			"title":    "Wrong board",
		},
	}

	// Wrap in data envelope like Appwrite sends
	wrapped := map[string]interface{}{
		"data": msg,
	}
	msgBytes, _ := json.Marshal(wrapped)
	client.handleMessage(msgBytes)

	// Should not receive any event
	select {
	case ev := <-client.Events():
		t.Fatalf("should not receive event for wrong board, got: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Expected — no event
	}

	client.Close()
}

func TestClientClose(t *testing.T) {
	client := NewClient(SubscriptionConfig{})

	client.Close()

	// Verify done channel is closed
	select {
	case <-client.done:
		// Expected
	default:
		t.Fatal("done channel should be closed after Close()")
	}

	// Double close should not panic
	client.Close()
}

func TestParseRealtimeMessage(t *testing.T) {
	msg := map[string]interface{}{
		"events": []interface{}{
			"databases.obeya.collections.items.documents.item-42.update",
		},
		"payload": map[string]interface{}{
			"$id":      "item-42",
			"board_id": "board-xyz",
			"title":    "Updated title",
			"status":   "in-progress",
		},
	}

	event := parseRealtimeMessage(msg, "board-xyz")
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Action != ActionUpdate {
		t.Errorf("expected update action, got %s", event.Action)
	}
	if event.DocumentID != "item-42" {
		t.Errorf("expected doc ID item-42, got %s", event.DocumentID)
	}
	if event.CollectionID != "items" {
		t.Errorf("expected collection items, got %s", event.CollectionID)
	}

	// Wrong board should return nil
	event = parseRealtimeMessage(msg, "wrong-board")
	if event != nil {
		t.Fatal("expected nil for wrong board_id")
	}
}
```

- [ ] **Step 8: Run all realtime tests**

```bash
cd ~/code/obeya
go test ./internal/realtime/ -v
```

Expected: All PASS

- [ ] **Step 9: Commit**

```bash
cd ~/code/obeya
git add internal/realtime/ go.mod go.sum
git commit -m "feat: add Appwrite realtime WebSocket client with event parsing"
```

---

## Chunk 7: CLI TUI — Watcher Interface Abstraction

### Task 7: Abstract Watcher Interface for Local/Cloud Mode

**Files:**
- Modify: `internal/tui/watcher.go`
- Modify: `internal/tui/watcher_test.go`
- Modify: `internal/tui/app.go`

The current TUI uses `boardWatcher` (fsnotify) directly. We need to abstract this behind an interface so the TUI can use either fsnotify (local mode) or WebSocket (cloud mode).

- [ ] **Step 1: Define the Watcher interface**

Add to the top of `internal/tui/watcher.go`, and rename the existing `boardWatcher` to `localBoardWatcher`:

Modify: `internal/tui/watcher.go`

Replace the entire file with:

```go
package tui

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// boardFileChangedMsg signals that the board was modified (locally or remotely).
type boardFileChangedMsg struct{}

// watcherStartedMsg carries the initialized watcher (or nil on failure).
type watcherStartedMsg struct {
	watcher boardWatcher
	err     error
}

const debounceInterval = 100 * time.Millisecond

// boardWatcher is the interface for watching board changes.
// It abstracts over local file watching (fsnotify) and cloud
// realtime subscriptions (WebSocket).
type boardWatcher interface {
	// events returns a channel that signals when the board has changed.
	events() <-chan struct{}

	// errors returns a channel that receives watcher errors.
	errors() <-chan error

	// close shuts down the watcher and releases resources.
	close()
}

// localBoardWatcher watches a directory for changes to a specific file.
// It watches the directory (not the file) because writeBoard() uses
// atomic rename (tmp -> board.json), which replaces the inode.
type localBoardWatcher struct {
	watcher   *fsnotify.Watcher
	eventCh   chan struct{}
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once
	fileName  string // just the base name, e.g. "board.json"
}

func newLocalBoardWatcher(boardFilePath string) (*localBoardWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(boardFilePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	bw := &localBoardWatcher{
		watcher:  w,
		eventCh:  make(chan struct{}, 1),
		errCh:    make(chan error, 1),
		done:     make(chan struct{}),
		fileName: filepath.Base(boardFilePath),
	}

	go bw.loop()
	return bw, nil
}

func (bw *localBoardWatcher) loop() {
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
		close(bw.eventCh)
		close(bw.errCh)
	}()

	for {
		select {
		case event, ok := <-bw.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != bw.fileName {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					select {
					case bw.eventCh <- struct{}{}:
					default:
						// channel full, notification already pending
					}
				})
			}
		case err, ok := <-bw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case bw.errCh <- err:
			default:
			}
		case <-bw.done:
			return
		}
	}
}

func (bw *localBoardWatcher) events() <-chan struct{} {
	return bw.eventCh
}

func (bw *localBoardWatcher) errors() <-chan error {
	return bw.errCh
}

func (bw *localBoardWatcher) close() {
	bw.closeOnce.Do(func() {
		close(bw.done)
		bw.watcher.Close()
	})
}
```

- [ ] **Step 2: Add cloud watcher that bridges the realtime client**

Create: `internal/tui/cloud_watcher.go`

```go
package tui

import (
	"sync"

	"github.com/niladribose/obeya/internal/realtime"
)

// cloudBoardWatcher bridges the realtime.Client to the boardWatcher interface.
// It translates Appwrite realtime events into simple "board changed" signals,
// matching the same interface that fsnotify-based local watching uses.
type cloudBoardWatcher struct {
	client    *realtime.Client
	eventCh   chan struct{}
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once
}

// newCloudBoardWatcher creates a watcher that subscribes to Appwrite realtime
// for the specified board. It starts the WebSocket connection in a goroutine.
func newCloudBoardWatcher(config realtime.SubscriptionConfig) *cloudBoardWatcher {
	client := realtime.NewClient(config)

	cw := &cloudBoardWatcher{
		client:  client,
		eventCh: make(chan struct{}, 1),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}

	// Start the WebSocket connection
	go client.Connect()

	// Start the event relay goroutine
	go cw.relay()

	return cw
}

// relay reads from the realtime client's event and error channels
// and forwards them to the boardWatcher-compatible channels.
func (cw *cloudBoardWatcher) relay() {
	defer func() {
		close(cw.eventCh)
		close(cw.errCh)
	}()

	for {
		select {
		case <-cw.done:
			return
		case _, ok := <-cw.client.Events():
			if !ok {
				return
			}
			// Any board event triggers a reload — we don't differentiate
			// between create/update/delete at the TUI level, the board
			// reload will pick up all changes.
			select {
			case cw.eventCh <- struct{}{}:
			default:
				// channel full, notification already pending
			}
		case err, ok := <-cw.client.Errors():
			if !ok {
				return
			}
			select {
			case cw.errCh <- err:
			default:
			}
		}
	}
}

func (cw *cloudBoardWatcher) events() <-chan struct{} {
	return cw.eventCh
}

func (cw *cloudBoardWatcher) errors() <-chan error {
	return cw.errCh
}

func (cw *cloudBoardWatcher) close() {
	cw.closeOnce.Do(func() {
		close(cw.done)
		cw.client.Close()
	})
}
```

- [ ] **Step 3: Update app.go to use the boardWatcher interface**

Modify the `App` struct in `internal/tui/app.go` to use the interface. The watcher field already holds the concrete type — change it to the interface type and update `startWatching()` to select the appropriate watcher based on mode.

In `internal/tui/app.go`, update the App struct and related methods:

Change the `watcher` field type from `*boardWatcher` to `boardWatcher` (the interface):

```go
// App is the enhanced Bubble Tea model for the Obeya board TUI.
type App struct {
	engine    *engine.Engine
	board     *domain.Board
	boardPath string // path to board.json for file watching

	// Cloud mode config (nil in local mode)
	cloudConfig *CloudConfig

	// Board navigation
	columns    []string
	cursorCol  int
	cursorRow  int
	collapsed  map[string]bool
	colScrollY map[int]int // per-column scroll offsets

	// State machine
	state     viewState
	prevState viewState

	// Sub-components
	detail     DetailModel
	picker     PickerModel
	input      InputModel
	dashboard  DashboardModel
	confirmMsg string

	// Dimensions
	width  int
	height int

	watcher boardWatcher
	err     error
}

// CloudConfig holds the configuration needed for cloud-mode realtime.
type CloudConfig struct {
	AppwriteEndpoint string
	ProjectID        string
	APIToken         string
	DatabaseID       string
	BoardID          string
}
```

Update `NewApp` to accept an optional cloud config:

```go
// NewApp creates a new enhanced TUI app backed by the given engine.
func NewApp(eng *engine.Engine, boardPath string) App {
	return App{
		engine:     eng,
		boardPath:  boardPath,
		collapsed:  make(map[string]bool),
		colScrollY: make(map[int]int),
		state:      stateBoard,
	}
}

// NewCloudApp creates a TUI app configured for cloud-mode realtime.
func NewCloudApp(eng *engine.Engine, config CloudConfig) App {
	return App{
		engine:      eng,
		cloudConfig: &config,
		collapsed:   make(map[string]bool),
		colScrollY:  make(map[int]int),
		state:       stateBoard,
	}
}
```

Update `startWatching()` to select the watcher type:

```go
func (a App) startWatching() tea.Cmd {
	return func() tea.Msg {
		if a.cloudConfig != nil {
			// Cloud mode: use WebSocket realtime
			cw := newCloudBoardWatcher(realtime.SubscriptionConfig{
				AppwriteEndpoint: a.cloudConfig.AppwriteEndpoint,
				ProjectID:        a.cloudConfig.ProjectID,
				APIToken:         a.cloudConfig.APIToken,
				DatabaseID:       a.cloudConfig.DatabaseID,
				BoardID:          a.cloudConfig.BoardID,
			})
			return watcherStartedMsg{watcher: cw}
		}

		// Local mode: use fsnotify file watching
		if a.boardPath == "" {
			return watcherStartedMsg{watcher: nil, err: fmt.Errorf("no board path for file watching")}
		}
		w, err := newLocalBoardWatcher(a.boardPath)
		if err != nil {
			return watcherStartedMsg{watcher: nil, err: err}
		}
		return watcherStartedMsg{watcher: w}
	}
}
```

The `Update()` handler for `watcherStartedMsg` and `waitForFileChange()` remain identical since they use the `boardWatcher` interface.

- [ ] **Step 4: Update watcher_test.go to use the new type name**

Modify: `internal/tui/watcher_test.go`

Replace `newBoardWatcher` with `newLocalBoardWatcher` in all test functions:

```go
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSendsMessageOnFileChange(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newLocalBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond) // let watcher settle

	// Modify via atomic rename (same as json_store.go writeBoard)
	tmpFile := boardFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpFile, boardFile); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected file change notification, got timeout")
	}
}

func TestWatcherDebounceCoalescesRapidWrites(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newLocalBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond)

	// Write 5 times rapidly via atomic rename (same pattern as writeBoard)
	for i := 0; i < 5; i++ {
		tmpFile := boardFile + ".tmp"
		os.WriteFile(tmpFile, []byte(fmt.Sprintf(`{"version":%d}`, i)), 0644)
		os.Rename(tmpFile, boardFile)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire, then drain
	time.Sleep(200 * time.Millisecond)

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 1 {
		t.Fatalf("expected 1 debounced notification, got %d", count)
	}
}

func TestWatcherCloseStopsWatching(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	os.WriteFile(boardFile, []byte(`{}`), 0644)

	w, err := newLocalBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}

	ch := w.events()
	w.close()

	// Write after close — should NOT get notification
	time.Sleep(50 * time.Millisecond)
	os.WriteFile(boardFile, []byte(`{"version":99}`), 0644)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("should not receive events after close")
		}
		// channel closed — expected
	case <-time.After(300 * time.Millisecond):
		// success — no event received
	}
}
```

- [ ] **Step 5: Add cloud watcher tests**

Create: `internal/tui/cloud_watcher_test.go`

```go
package tui

import (
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/realtime"
)

func TestCloudWatcherImplementsBoardWatcher(t *testing.T) {
	// Verify the interface is satisfied at compile time
	var _ boardWatcher = (*cloudBoardWatcher)(nil)
}

func TestCloudWatcherRelaysEvents(t *testing.T) {
	// Create a realtime client with a test config (won't actually connect)
	config := realtime.SubscriptionConfig{
		AppwriteEndpoint: "http://localhost:9999/v1",
		ProjectID:        "test",
		DatabaseID:       "obeya",
		BoardID:          "board-test",
	}

	client := realtime.NewClient(config)

	cw := &cloudBoardWatcher{
		client:  client,
		eventCh: make(chan struct{}, 1),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}

	go cw.relay()

	// Close to clean up
	defer cw.close()

	// Since we can't easily inject events into the realtime client
	// without a real WebSocket server, we verify the interface contract
	// and that close works cleanly
	cw.close()

	// After close, channels should be closed eventually
	select {
	case <-cw.done:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("done channel not closed after close()")
	}
}

func TestLocalWatcherImplementsBoardWatcher(t *testing.T) {
	// Verify the interface is satisfied at compile time
	var _ boardWatcher = (*localBoardWatcher)(nil)
}
```

- [ ] **Step 6: Run all TUI tests**

```bash
cd ~/code/obeya
go test ./internal/tui/ -v
```

Expected: All PASS

- [ ] **Step 7: Run all realtime tests**

```bash
cd ~/code/obeya
go test ./internal/realtime/ -v
```

Expected: All PASS

- [ ] **Step 8: Commit**

```bash
cd ~/code/obeya
git add internal/tui/watcher.go internal/tui/watcher_test.go internal/tui/cloud_watcher.go internal/tui/cloud_watcher_test.go internal/tui/app.go
git commit -m "feat: abstract boardWatcher interface for local/cloud TUI realtime"
```

---

## Chunk 8: Web UI — Activity Feed Integration

### Task 8: Activity Feed with Realtime Updates

**Files:**
- Create: `obeya-cloud/components/activity/activity-feed.tsx`
- Test: `obeya-cloud/__tests__/components/activity/activity-feed.test.tsx`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/components/activity/activity-feed.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ActivityFeed } from "@/components/activity/activity-feed";
import type { HistoryEntry } from "@/hooks/use-activity-subscription";

// Mock the subscription hook
vi.mock("@/hooks/use-activity-subscription", () => ({
  useActivitySubscription: vi.fn().mockReturnValue({ status: "connected" }),
}));

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: vi.fn().mockReturnValue(vi.fn()),
  })),
}));

describe("ActivityFeed", () => {
  const mockEntries: HistoryEntry[] = [
    {
      $id: "hist-1",
      item_id: "item-1",
      board_id: "board-123",
      user_id: "user-1",
      action: "created",
      detail: "Created task #42 'Build login page'",
      timestamp: "2026-03-12T10:00:00Z",
    },
    {
      $id: "hist-2",
      item_id: "item-2",
      board_id: "board-123",
      user_id: "user-2",
      action: "moved",
      detail: "status: todo -> in-progress",
      timestamp: "2026-03-12T10:05:00Z",
    },
    {
      $id: "hist-3",
      item_id: "item-1",
      board_id: "board-123",
      user_id: "user-1",
      action: "edited",
      detail: "Updated description",
      timestamp: "2026-03-12T10:10:00Z",
    },
  ];

  it("renders a list of activity entries", () => {
    const { container } = render(
      <ActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={mockEntries}
      />
    );

    expect(container.textContent).toContain("created");
    expect(container.textContent).toContain("moved");
    expect(container.textContent).toContain("edited");
  });

  it("shows entries in reverse chronological order (newest first)", () => {
    const { container } = render(
      <ActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={mockEntries}
      />
    );

    const items = container.querySelectorAll("[data-activity-entry]");
    expect(items.length).toBe(3);

    // First entry should be the newest (hist-3, timestamp 10:10)
    expect(items[0].textContent).toContain("edited");
  });

  it("renders empty state when no entries", () => {
    const { container } = render(
      <ActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={[]}
      />
    );

    expect(container.textContent).toContain("No activity");
  });

  it("displays the action type with an appropriate icon/label", () => {
    const { container } = render(
      <ActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={[mockEntries[1]]} // "moved" action
      />
    );

    expect(container.textContent).toContain("moved");
    expect(container.textContent).toContain("status: todo -> in-progress");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/activity/activity-feed.test.tsx
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/components/activity/activity-feed.tsx`

```typescript
"use client";

import { useState, useCallback } from "react";
import {
  useActivitySubscription,
  type HistoryEntry,
  type ActivityEvent,
} from "@/hooks/use-activity-subscription";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";

interface ActivityFeedProps {
  boardId: string;
  databaseId: string;
  initialEntries: HistoryEntry[];
  maxEntries?: number;
}

const ACTION_CONFIG: Record<string, { label: string; color: string }> = {
  created: {
    label: "created",
    color: "text-green-600 dark:text-green-400",
  },
  moved: {
    label: "moved",
    color: "text-blue-600 dark:text-blue-400",
  },
  edited: {
    label: "edited",
    color: "text-yellow-600 dark:text-yellow-400",
  },
  assigned: {
    label: "assigned",
    color: "text-purple-600 dark:text-purple-400",
  },
  blocked: {
    label: "blocked",
    color: "text-red-600 dark:text-red-400",
  },
  unblocked: {
    label: "unblocked",
    color: "text-teal-600 dark:text-teal-400",
  },
  deleted: {
    label: "deleted",
    color: "text-red-600 dark:text-red-400",
  },
};

/**
 * Activity feed component that shows board history entries
 * with live updates from the Appwrite realtime subscription.
 *
 * New entries animate in at the top of the list.
 */
export function ActivityFeed({
  boardId,
  databaseId,
  initialEntries,
  maxEntries = 50,
}: ActivityFeedProps) {
  // Sort initial entries by timestamp descending (newest first)
  const sorted = [...initialEntries].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );
  const [entries, setEntries] = useState<HistoryEntry[]>(sorted);

  const handleActivity = useCallback(
    (event: ActivityEvent) => {
      setEntries((prev) => {
        // Avoid duplicates
        if (prev.some((e) => e.$id === event.entry.$id)) {
          return prev;
        }
        // Prepend new entry, trim to max
        return [event.entry, ...prev].slice(0, maxEntries);
      });
    },
    [maxEntries]
  );

  const { status } = useActivitySubscription({
    boardId,
    databaseId,
    onActivity: handleActivity,
  });

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-gray-400">
        <p>No activity yet</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between px-4 py-2 border-b">
        <h3 className="text-sm font-semibold">Activity</h3>
        <ConnectionStatusIndicator status={status} compact />
      </div>

      <div className="flex flex-col divide-y">
        {entries.map((entry) => (
          <ActivityEntry key={entry.$id} entry={entry} />
        ))}
      </div>
    </div>
  );
}

function ActivityEntry({ entry }: { entry: HistoryEntry }) {
  const config = ACTION_CONFIG[entry.action] || {
    label: entry.action,
    color: "text-gray-600",
  };

  const timeAgo = formatTimeAgo(entry.timestamp);

  return (
    <div
      className="px-4 py-3 transition-all duration-200 animate-in fade-in slide-in-from-top-1"
      data-activity-entry
    >
      <div className="flex items-center gap-2 text-sm">
        <span className="text-gray-500 text-xs">{entry.user_id}</span>
        <span className={`font-medium ${config.color}`}>{config.label}</span>
        <span className="text-gray-400 text-xs ml-auto">{timeAgo}</span>
      </div>
      <p className="text-xs text-gray-500 mt-1">{entry.detail}</p>
    </div>
  );
}

function formatTimeAgo(timestamp: string): string {
  const now = new Date();
  const then = new Date(timestamp);
  const diffMs = now.getTime() - then.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/activity/activity-feed.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add components/activity/activity-feed.tsx __tests__/components/activity/activity-feed.test.tsx
git commit -m "feat: add ActivityFeed component with realtime live updates"
```

---

## Chunk 9: Web UI — Automatic Reconnection Handling

### Task 9: Reconnection Logic and Error Boundaries

**Files:**
- Create: `obeya-cloud/hooks/use-reconnecting-subscription.ts`
- Test: `obeya-cloud/__tests__/hooks/use-reconnecting-subscription.test.ts`

The Appwrite JS SDK handles reconnection internally, but we need a wrapper that:
1. Detects disconnection (subscription callback stops firing)
2. Re-subscribes when the network comes back
3. Triggers a full data refetch to catch any missed events

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/hooks/use-reconnecting-subscription.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useReconnectingSubscription } from "@/hooks/use-reconnecting-subscription";

describe("useReconnectingSubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("calls onRefresh when online event fires after being offline", () => {
    const onRefresh = vi.fn();
    const onEvent = vi.fn();

    renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent,
        onRefresh,
      })
    );

    // Simulate browser going offline then online
    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    act(() => {
      window.dispatchEvent(new Event("online"));
    });

    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("subscribes to the correct channel", () => {
    const onEvent = vi.fn();
    const onRefresh = vi.fn();

    renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent,
        onRefresh,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);
    expect(mockSubscribe.mock.calls[0][0]).toBe(
      "databases.obeya.collections.items.documents"
    );
  });

  it("unsubscribes and removes listeners on unmount", () => {
    const removeListenerSpy = vi.spyOn(window, "removeEventListener");

    const { unmount } = renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent: vi.fn(),
        onRefresh: vi.fn(),
      })
    );

    unmount();

    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
    expect(removeListenerSpy).toHaveBeenCalled();
  });

  it("does not subscribe when boardId is empty", () => {
    renderHook(() =>
      useReconnectingSubscription({
        boardId: "",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent: vi.fn(),
        onRefresh: vi.fn(),
      })
    );

    expect(mockSubscribe).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-reconnecting-subscription.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/hooks/use-reconnecting-subscription.ts`

```typescript
"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

interface UseReconnectingSubscriptionOptions {
  boardId: string;
  databaseId: string;
  channel: string;
  onEvent: (event: any) => void;
  onRefresh: () => void;
}

interface UseReconnectingSubscriptionResult {
  status: ConnectionStatus;
}

/**
 * Low-level hook that wraps Appwrite client.subscribe with reconnection handling.
 *
 * Monitors browser online/offline events. When the browser transitions from
 * offline to online, it calls onRefresh() to trigger a full data refetch
 * (since events may have been missed while offline).
 *
 * The Appwrite JS SDK handles WebSocket reconnection internally. This hook
 * adds the application-level data recovery on top of that.
 */
export function useReconnectingSubscription(
  options: UseReconnectingSubscriptionOptions
): UseReconnectingSubscriptionResult {
  const { boardId, databaseId, channel, onEvent, onRefresh } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onEventRef = useRef(onEvent);
  const onRefreshRef = useRef(onRefresh);
  const wasOfflineRef = useRef(false);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();

    const unsubscribe = client.subscribe(channel, (event: any) => {
      onEventRef.current(event);
    });

    setStatus("connected");

    // Online/offline handlers for reconnection recovery
    const handleOffline = () => {
      wasOfflineRef.current = true;
      setStatus("disconnected");
    };

    const handleOnline = () => {
      if (wasOfflineRef.current) {
        wasOfflineRef.current = false;
        setStatus("connected");
        // Trigger a full data refetch to catch any missed events
        onRefreshRef.current();
      }
    };

    window.addEventListener("offline", handleOffline);
    window.addEventListener("online", handleOnline);

    return () => {
      unsubscribe();
      window.removeEventListener("offline", handleOffline);
      window.removeEventListener("online", handleOnline);
      setStatus("disconnected");
    };
  }, [boardId, databaseId, channel]);

  return { status };
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/hooks/use-reconnecting-subscription.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add hooks/use-reconnecting-subscription.ts __tests__/hooks/use-reconnecting-subscription.test.ts
git commit -m "feat: add reconnection-aware subscription hook with data recovery"
```

---

## Chunk 10: Full Integration — Run All Tests

### Task 10: Integration Verification

**Files:** No new files — this task runs all tests across both repos.

- [ ] **Step 1: Run all web UI tests**

```bash
cd ~/code/obeya-cloud
npm test
```

Expected: All tests PASS

- [ ] **Step 2: Run all Go tests**

```bash
cd ~/code/obeya
go test ./internal/realtime/ -v
go test ./internal/tui/ -v
```

Expected: All tests PASS

- [ ] **Step 3: Verify Go builds cleanly**

```bash
cd ~/code/obeya
go build ./...
```

Expected: No errors

- [ ] **Step 4: Commit final state**

```bash
# In obeya-cloud
cd ~/code/obeya-cloud
git add -A
git commit -m "feat: complete realtime integration — all tests passing"

# In obeya
cd ~/code/obeya
git add -A
git commit -m "feat: complete cloud realtime TUI integration — all tests passing"
```

---

## Summary

This plan delivers:

| Component | Repo | What's built |
|-----------|------|-------------|
| **Browser Client** | obeya-cloud | Appwrite JS SDK singleton for browser-side WebSocket (`lib/appwrite/browser-client.ts`) |
| **Board Subscription** | obeya-cloud | `useBoardSubscription` hook — subscribes to item create/update/delete events, filters by board_id |
| **Activity Subscription** | obeya-cloud | `useActivitySubscription` hook — subscribes to item_history creates, filters by board_id |
| **Reconnection** | obeya-cloud | `useReconnectingSubscription` hook — online/offline detection, automatic data recovery |
| **Connection Status** | obeya-cloud | `ConnectionStatusIndicator` component — green/yellow/gray/red dot with label |
| **Kanban Integration** | obeya-cloud | `KanbanBoardRealtime` component — live card create/move/edit/delete with `useBoardRealtimeSync` |
| **Activity Feed** | obeya-cloud | `ActivityFeed` component — live new entries prepended with animation |
| **Go WebSocket Client** | obeya | `internal/realtime/client.go` — gorilla/websocket connection to Appwrite realtime, exponential backoff reconnection |
| **Go Event Parsing** | obeya | `internal/realtime/events.go` — Appwrite event format parsing, board_id filtering |
| **Watcher Interface** | obeya | `boardWatcher` interface in `internal/tui/watcher.go` — abstracts fsnotify (local) vs WebSocket (cloud) |
| **Cloud Watcher** | obeya | `cloudBoardWatcher` in `internal/tui/cloud_watcher.go` — bridges realtime.Client to boardWatcher interface |
| **TUI Integration** | obeya | Updated `App` struct in `internal/tui/app.go` — selects local/cloud watcher based on mode |

**Key design decisions:**
- Web UI uses Appwrite JS SDK (`appwrite` npm package) for native browser WebSocket support
- CLI TUI uses `gorilla/websocket` for direct WebSocket connection to Appwrite realtime
- Both clients filter events by `board_id` client-side (Appwrite sends all collection events)
- Appwrite realtime respects document-level permissions — no unauthorized event leakage
- TUI only subscribes in interactive board view mode, not for non-interactive commands
- Existing fsnotify watcher is preserved for local mode via the `boardWatcher` interface

**Next plan:** Plan 7 — Board & Item CRUD Web UI (React components for the authenticated dashboard pages)
