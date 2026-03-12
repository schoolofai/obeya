# Obeya Cloud Board & Item APIs — Implementation Plan (Part B)

> Continuation of Part A. See `2026-03-12-obeya-cloud-board-item-apis.md` for Board CRUD and Item CRUD tasks.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement item action endpoints (move, assign, block, unblock), history/activity feeds, and plan management APIs.

**Architecture:** Next.js 15 App Router API routes. Appwrite Server SDK for persistence. Vitest for testing with mocked Appwrite SDK. All responses use the `{ok, data, error, meta}` envelope.

**Tech Stack:** Next.js 15, TypeScript, Appwrite Node SDK (`node-appwrite`), Vitest, zod

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md`

**Repository:** `~/code/obeya-cloud`

---

## File Structure (Part B additions)

```
obeya-cloud/
├── app/api/
│   ├── items/[id]/
│   │   ├── move/route.ts          # POST /api/items/:id/move
│   │   ├── assign/route.ts        # POST /api/items/:id/assign
│   │   ├── block/route.ts         # POST /api/items/:id/block
│   │   └── block/[bid]/route.ts   # DELETE /api/items/:id/block/:bid
│   │   └── history/route.ts       # GET /api/items/:id/history
│   ├── boards/[id]/
│   │   ├── activity/route.ts      # GET /api/boards/:id/activity
│   │   └── plans/route.ts         # GET + POST /api/boards/:id/plans
│   └── plans/[id]/
│       ├── route.ts               # GET /api/plans/:id
│       └── link/
│           ├── route.ts           # POST /api/plans/:id/link
│           └── [iid]/route.ts     # DELETE /api/plans/:id/link/:iid
├── lib/
│   └── history.ts                 # History creation helper utility
└── __tests__/
    ├── lib/
    │   └── history.test.ts
    └── api/
        ├── items/
        │   ├── move.test.ts
        │   ├── assign.test.ts
        │   ├── block.test.ts
        │   └── history.test.ts
        ├── boards/
        │   └── activity.test.ts
        └── plans/
            ├── plans.test.ts
            ├── plan-detail.test.ts
            └── plan-link.test.ts
```

---

## Chunk 3: Item Actions

### Task 1: History Creation Helper

**Files:**
- Create: `obeya-cloud/lib/history.ts`
- Test: `obeya-cloud/__tests__/lib/history.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/history.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ID } from "node-appwrite";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("node-appwrite", () => ({
  ID: { unique: vi.fn().mockReturnValue("hist-1") },
}));

import { createHistoryEntry } from "@/lib/history";
import { getDatabases } from "@/lib/appwrite/server";

describe("createHistoryEntry", () => {
  const mockCreate = vi.fn().mockResolvedValue({ $id: "hist-1" });

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      createDocument: mockCreate,
    } as any);
  });

  it("creates item_history document with correct fields", async () => {
    await createHistoryEntry({
      itemId: "item-1",
      boardId: "board-1",
      userId: "user-1",
      action: "moved",
      detail: "status: todo -> in-progress",
    });

    expect(mockCreate).toHaveBeenCalledWith(
      "obeya",
      "item_history",
      "hist-1",
      expect.objectContaining({
        item_id: "item-1",
        board_id: "board-1",
        user_id: "user-1",
        action: "moved",
        detail: "status: todo -> in-progress",
      })
    );
  });

  it("includes timestamp in ISO format", async () => {
    await createHistoryEntry({
      itemId: "item-1",
      boardId: "board-1",
      userId: "user-1",
      action: "assigned",
      detail: "assignee: user-2",
    });

    const callArgs = mockCreate.mock.calls[0][3];
    expect(callArgs.timestamp).toBeDefined();
    expect(() => new Date(callArgs.timestamp)).not.toThrow();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/history.test.ts
```

Expected: FAIL -- module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/history.ts`

```typescript
import { ID } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";

type HistoryAction =
  | "created"
  | "moved"
  | "edited"
  | "assigned"
  | "blocked"
  | "unblocked";

interface HistoryParams {
  itemId: string;
  boardId: string;
  userId: string;
  action: HistoryAction;
  detail: string;
}

export async function createHistoryEntry(
  params: HistoryParams
): Promise<void> {
  const env = getEnv();
  const db = getDatabases();

  await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ITEM_HISTORY,
    ID.unique(),
    {
      item_id: params.itemId,
      board_id: params.boardId,
      user_id: params.userId,
      action: params.action,
      detail: params.detail,
      timestamp: new Date().toISOString(),
    }
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/lib/history.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lib/history.ts __tests__/lib/history.test.ts
git commit -m "feat: add history creation helper utility"
```

---

### Task 2: POST /api/items/:id/move

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/move/route.ts`
- Test: `obeya-cloud/__tests__/api/items/move.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/move.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("@/lib/history", () => ({
  createHistoryEntry: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("node-appwrite", () => ({
  ID: { unique: vi.fn().mockReturnValue("new-id") },
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    limit: vi.fn((n: number) => `limit=${n}`),
  },
}));

import { POST } from "@/app/api/items/[id]/move/route";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

describe("POST /api/items/:id/move", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
      listDocuments: mockListDocs,
    } as any);
  });

  it("moves item to new column and creates history entry", async () => {
    mockGetDoc
      .mockResolvedValueOnce({
        $id: "item-1", board_id: "board-1", status: "todo", title: "Task 1",
      })
      .mockResolvedValueOnce({
        $id: "board-1",
        columns: JSON.stringify([
          { name: "todo", limit: 0 },
          { name: "in-progress", limit: 3 },
          { name: "done", limit: 0 },
        ]),
      });

    mockListDocs.mockResolvedValue({ total: 1 });
    mockUpdateDoc.mockResolvedValue({ $id: "item-1", status: "in-progress" });

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "in-progress" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalled();
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "moved", itemId: "item-1" })
    );
  });

  it("rejects move when WIP limit is reached", async () => {
    mockGetDoc
      .mockResolvedValueOnce({
        $id: "item-1", board_id: "board-1", status: "todo", title: "Task 1",
      })
      .mockResolvedValueOnce({
        $id: "board-1",
        columns: JSON.stringify([
          { name: "todo", limit: 0 },
          { name: "in-progress", limit: 2 },
        ]),
      });

    mockListDocs.mockResolvedValue({ total: 2 });

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "in-progress" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe("VALIDATION_ERROR");
  });

  it("rejects move to non-existent column", async () => {
    mockGetDoc
      .mockResolvedValueOnce({
        $id: "item-1", board_id: "board-1", status: "todo", title: "Task 1",
      })
      .mockResolvedValueOnce({
        $id: "board-1",
        columns: JSON.stringify([{ name: "todo", limit: 0 }]),
      });

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "nonexistent" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/items/move.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/move/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";
import { createHistoryEntry } from "@/lib/history";

const paramsSchema = z.object({ id: z.string().min(1) });
const bodySchema = z.object({ status: z.string().min(1) });

interface Column {
  name: string;
  limit: number;
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const { status } = await validateBody(request, bodySchema);

    const env = getEnv();
    const db = getDatabases();

    const item = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id
    );

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      item.board_id
    );

    const columns: Column[] = JSON.parse(board.columns);
    const targetColumn = columns.find((c) => c.name === status);
    if (!targetColumn) {
      throw new AppError(
        ErrorCode.VALIDATION_ERROR,
        `Column "${status}" does not exist on this board`
      );
    }

    if (targetColumn.limit > 0) {
      const countResult = await db.listDocuments(
        env.APPWRITE_DATABASE_ID,
        COLLECTIONS.ITEMS,
        [
          Query.equal("board_id", item.board_id),
          Query.equal("status", status),
          Query.limit(targetColumn.limit + 1),
        ]
      );

      if (countResult.total >= targetColumn.limit) {
        throw new AppError(
          ErrorCode.VALIDATION_ERROR,
          `WIP limit reached for column "${status}" (limit: ${targetColumn.limit})`
        );
      }
    }

    const oldStatus = item.status;
    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id,
      { status, updated_at: new Date().toISOString() }
    );

    await createHistoryEntry({
      itemId: id,
      boardId: item.board_id,
      userId: user.id,
      action: "moved",
      detail: `status: ${oldStatus} -> ${status}`,
    });

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/items/move.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/items/\[id\]/move/ __tests__/api/items/move.test.ts
git commit -m "feat: add POST /api/items/:id/move with WIP limit checking"
```

---

### Task 3: POST /api/items/:id/assign

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/assign/route.ts`
- Test: `obeya-cloud/__tests__/api/items/assign.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/assign.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("@/lib/history", () => ({
  createHistoryEntry: vi.fn().mockResolvedValue(undefined),
}));

import { POST } from "@/app/api/items/[id]/assign/route";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

describe("POST /api/items/:id/assign", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("assigns user to item and creates history entry", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1", assignee_id: null, title: "Task 1",
    });
    mockUpdateDoc.mockResolvedValue({
      $id: "item-1", assignee_id: "user-2",
    });

    const request = new Request("http://localhost/api/items/item-1/assign", {
      method: "POST",
      body: JSON.stringify({ assignee_id: "user-2" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "items", "item-1",
      expect.objectContaining({ assignee_id: "user-2" })
    );
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "assigned" })
    );
  });

  it("allows unassigning by passing null", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1", assignee_id: "user-2", title: "Task 1",
    });
    mockUpdateDoc.mockResolvedValue({ $id: "item-1", assignee_id: null });

    const request = new Request("http://localhost/api/items/item-1/assign", {
      method: "POST",
      body: JSON.stringify({ assignee_id: null }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/items/assign.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/assign/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { createHistoryEntry } from "@/lib/history";

const paramsSchema = z.object({ id: z.string().min(1) });
const bodySchema = z.object({
  assignee_id: z.string().min(1).nullable(),
});

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const { assignee_id } = await validateBody(request, bodySchema);

    const env = getEnv();
    const db = getDatabases();

    const item = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id
    );

    const oldAssignee = item.assignee_id ?? "none";
    const newAssignee = assignee_id ?? "none";

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id,
      { assignee_id, updated_at: new Date().toISOString() }
    );

    await createHistoryEntry({
      itemId: id,
      boardId: item.board_id,
      userId: user.id,
      action: "assigned",
      detail: `assignee: ${oldAssignee} -> ${newAssignee}`,
    });

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/items/assign.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/items/\[id\]/assign/ __tests__/api/items/assign.test.ts
git commit -m "feat: add POST /api/items/:id/assign with history tracking"
```

---

### Task 4: POST /api/items/:id/block

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/block/route.ts`
- Test: `obeya-cloud/__tests__/api/items/block.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/block.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("@/lib/history", () => ({
  createHistoryEntry: vi.fn().mockResolvedValue(undefined),
}));

import { POST } from "@/app/api/items/[id]/block/route";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

describe("POST /api/items/:id/block", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("adds blocker to blocked_by array and creates history", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1", blocked_by: "[]", title: "Task 1",
    });
    mockUpdateDoc.mockResolvedValue({
      $id: "item-1", blocked_by: '["blocker-1"]',
    });

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "blocker-1" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "items", "item-1",
      expect.objectContaining({
        blocked_by: JSON.stringify(["blocker-1"]),
      })
    );
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "blocked" })
    );
  });

  it("appends to existing blocked_by array without duplicates", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1",
      blocked_by: JSON.stringify(["blocker-1"]), title: "Task 1",
    });
    mockUpdateDoc.mockResolvedValue({ $id: "item-1" });

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "blocker-2" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    expect(response.status).toBe(200);

    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "items", "item-1",
      expect.objectContaining({
        blocked_by: JSON.stringify(["blocker-1", "blocker-2"]),
      })
    );
  });

  it("rejects duplicate blocker", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1",
      blocked_by: JSON.stringify(["blocker-1"]), title: "Task 1",
    });

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "blocker-1" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/items/block.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/block/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";
import { createHistoryEntry } from "@/lib/history";

const paramsSchema = z.object({ id: z.string().min(1) });
const bodySchema = z.object({ blocked_by_id: z.string().min(1) });

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const { blocked_by_id } = await validateBody(request, bodySchema);

    const env = getEnv();
    const db = getDatabases();

    const item = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id
    );

    const blockedBy: string[] = JSON.parse(item.blocked_by || "[]");

    if (blockedBy.includes(blocked_by_id)) {
      throw new AppError(
        ErrorCode.VALIDATION_ERROR,
        `Item is already blocked by "${blocked_by_id}"`
      );
    }

    blockedBy.push(blocked_by_id);

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id,
      {
        blocked_by: JSON.stringify(blockedBy),
        updated_at: new Date().toISOString(),
      }
    );

    await createHistoryEntry({
      itemId: id,
      boardId: item.board_id,
      userId: user.id,
      action: "blocked",
      detail: `blocked by: ${blocked_by_id}`,
    });

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/items/block.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/items/\[id\]/block/route.ts __tests__/api/items/block.test.ts
git commit -m "feat: add POST /api/items/:id/block with duplicate detection"
```

---

### Task 5: DELETE /api/items/:id/block/:bid

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/block/[bid]/route.ts`
- Test: (covered in `__tests__/api/items/block.test.ts` — append tests)

- [ ] **Step 1: Append failing test to block.test.ts**

Append to: `obeya-cloud/__tests__/api/items/block.test.ts`

```typescript
import { DELETE } from "@/app/api/items/[id]/block/[bid]/route";

describe("DELETE /api/items/:id/block/:bid", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("removes blocker from blocked_by array and creates history", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1",
      blocked_by: JSON.stringify(["blocker-1", "blocker-2"]), title: "Task 1",
    });
    mockUpdateDoc.mockResolvedValue({ $id: "item-1" });

    const request = new Request("http://localhost/api/items/item-1/block/blocker-1", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "item-1", bid: "blocker-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "items", "item-1",
      expect.objectContaining({
        blocked_by: JSON.stringify(["blocker-2"]),
      })
    );
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "unblocked" })
    );
  });

  it("returns error when blocker ID not found in array", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "item-1", board_id: "board-1",
      blocked_by: JSON.stringify(["blocker-1"]), title: "Task 1",
    });

    const request = new Request("http://localhost/api/items/item-1/block/nonexistent", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "item-1", bid: "nonexistent" }),
    });
    const body = await response.json();

    expect(response.status).toBe(404);
    expect(body.ok).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/items/block.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/block/[bid]/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";
import { createHistoryEntry } from "@/lib/history";

const paramsSchema = z.object({
  id: z.string().min(1),
  bid: z.string().min(1),
});

export async function DELETE(
  request: NextRequest,
  context: { params: Promise<{ id: string; bid: string }> }
) {
  try {
    const user = await authenticate(request);
    const { id, bid } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const item = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id
    );

    const blockedBy: string[] = JSON.parse(item.blocked_by || "[]");
    const index = blockedBy.indexOf(bid);

    if (index === -1) {
      throw new AppError(
        ErrorCode.ITEM_NOT_FOUND,
        `Blocker "${bid}" not found in blocked_by list`
      );
    }

    blockedBy.splice(index, 1);

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id,
      {
        blocked_by: JSON.stringify(blockedBy),
        updated_at: new Date().toISOString(),
      }
    );

    await createHistoryEntry({
      itemId: id,
      boardId: item.board_id,
      userId: user.id,
      action: "unblocked",
      detail: `unblocked: ${bid}`,
    });

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/items/block.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/items/\[id\]/block/\[bid\]/ __tests__/api/items/block.test.ts
git commit -m "feat: add DELETE /api/items/:id/block/:bid for unblocking"
```

---

## Chunk 4: History & Activity

### Task 6: GET /api/items/:id/history

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/history/route.ts`
- Test: `obeya-cloud/__tests__/api/items/history.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/history.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("node-appwrite", () => ({
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    orderDesc: vi.fn((field: string) => `orderDesc=${field}`),
    limit: vi.fn((n: number) => `limit=${n}`),
    offset: vi.fn((n: number) => `offset=${n}`),
  },
}));

import { GET } from "@/app/api/items/[id]/history/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/items/:id/history", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns history entries sorted by timestamp desc", async () => {
    mockListDocs.mockResolvedValue({
      total: 2,
      documents: [
        { $id: "h2", action: "moved", timestamp: "2026-03-12T11:00:00Z" },
        { $id: "h1", action: "created", timestamp: "2026-03-12T10:00:00Z" },
      ],
    });

    const request = new Request("http://localhost/api/items/item-1/history");
    const response = await GET(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.meta.total).toBe(2);
  });

  it("returns empty array when no history exists", async () => {
    mockListDocs.mockResolvedValue({ total: 0, documents: [] });

    const request = new Request("http://localhost/api/items/item-1/history");
    const response = await GET(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toEqual([]);
    expect(body.meta.total).toBe(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/items/history.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/history/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const url = new URL(request.url);
    const limit = Math.min(parseInt(url.searchParams.get("limit") || "25"), 100);
    const offset = parseInt(url.searchParams.get("offset") || "0");

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEM_HISTORY,
      [
        Query.equal("item_id", id),
        Query.orderDesc("timestamp"),
        Query.limit(limit),
        Query.offset(offset),
      ]
    );

    return ok(result.documents, {
      meta: { total: result.total, limit, offset },
    });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/items/history.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/items/\[id\]/history/ __tests__/api/items/history.test.ts
git commit -m "feat: add GET /api/items/:id/history with pagination"
```

---

### Task 7: GET /api/boards/:id/activity

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/activity/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/activity.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/activity.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("node-appwrite", () => ({
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    orderDesc: vi.fn((field: string) => `orderDesc=${field}`),
    limit: vi.fn((n: number) => `limit=${n}`),
    offset: vi.fn((n: number) => `offset=${n}`),
  },
}));

import { GET } from "@/app/api/boards/[id]/activity/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards/:id/activity", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns board-wide activity feed paginated", async () => {
    mockListDocs.mockResolvedValue({
      total: 50,
      documents: [
        { $id: "h3", item_id: "item-2", action: "assigned", timestamp: "2026-03-12T12:00:00Z" },
        { $id: "h2", item_id: "item-1", action: "moved", timestamp: "2026-03-12T11:00:00Z" },
      ],
    });

    const request = new Request("http://localhost/api/boards/board-1/activity?limit=2&offset=0");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.meta.total).toBe(50);
    expect(body.meta.limit).toBe(2);
  });

  it("uses default pagination when no query params", async () => {
    mockListDocs.mockResolvedValue({ total: 0, documents: [] });

    const request = new Request("http://localhost/api/boards/board-1/activity");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.meta.limit).toBe(25);
    expect(body.meta.offset).toBe(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/boards/activity.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/[id]/activity/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const url = new URL(request.url);
    const limit = Math.min(parseInt(url.searchParams.get("limit") || "25"), 100);
    const offset = parseInt(url.searchParams.get("offset") || "0");

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEM_HISTORY,
      [
        Query.equal("board_id", id),
        Query.orderDesc("timestamp"),
        Query.limit(limit),
        Query.offset(offset),
      ]
    );

    return ok(result.documents, {
      meta: { total: result.total, limit, offset },
    });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/boards/activity.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/boards/\[id\]/activity/ __tests__/api/boards/activity.test.ts
git commit -m "feat: add GET /api/boards/:id/activity with pagination"
```

---

## Chunk 5: Plans

### Task 8: GET + POST /api/boards/:id/plans

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/plans/route.ts`
- Test: `obeya-cloud/__tests__/api/plans/plans.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/plans/plans.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("node-appwrite", () => ({
  ID: { unique: vi.fn().mockReturnValue("plan-1") },
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    orderDesc: vi.fn((field: string) => `orderDesc=${field}`),
    limit: vi.fn((n: number) => `limit=${n}`),
    offset: vi.fn((n: number) => `offset=${n}`),
  },
}));

import { GET, POST } from "@/app/api/boards/[id]/plans/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards/:id/plans", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns list of plans for a board", async () => {
    mockListDocs.mockResolvedValue({
      total: 1,
      documents: [
        { $id: "plan-1", title: "Sprint Plan", display_num: 5, board_id: "board-1" },
      ],
    });

    const request = new Request("http://localhost/api/boards/board-1/plans");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].title).toBe("Sprint Plan");
  });
});

describe("POST /api/boards/:id/plans", () => {
  const mockGetDoc = vi.fn();
  const mockCreateDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      createDocument: mockCreateDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("creates plan with incremented display counter", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "board-1", display_counter: 10,
    });
    mockUpdateDoc.mockResolvedValue({ $id: "board-1", display_counter: 11 });
    mockCreateDoc.mockResolvedValue({
      $id: "plan-1", title: "Release Plan", display_num: 11, board_id: "board-1",
    });

    const request = new Request("http://localhost/api/boards/board-1/plans", {
      method: "POST",
      body: JSON.stringify({
        title: "Release Plan",
        source_path: "docs/plans/release.md",
        content: "# Release Plan\n\nSteps...",
      }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "boards", "board-1",
      expect.objectContaining({ display_counter: 11 })
    );
    expect(mockCreateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        title: "Release Plan",
        board_id: "board-1",
        display_num: 11,
        linked_items: "[]",
      })
    );
  });

  it("rejects plan creation with missing title", async () => {
    const request = new Request("http://localhost/api/boards/board-1/plans", {
      method: "POST",
      body: JSON.stringify({ content: "# no title" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/plans/plans.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/[id]/plans/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { ID, Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

const createPlanSchema = z.object({
  title: z.string().min(1),
  source_path: z.string().default(""),
  content: z.string().default(""),
});

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const url = new URL(request.url);
    const limit = Math.min(parseInt(url.searchParams.get("limit") || "25"), 100);
    const offset = parseInt(url.searchParams.get("offset") || "0");

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      [
        Query.equal("board_id", id),
        Query.orderDesc("created_at"),
        Query.limit(limit),
        Query.offset(offset),
      ]
    );

    return ok(result.documents, {
      meta: { total: result.total, limit, offset },
    });
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const body = await validateBody(request, createPlanSchema);

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id
    );

    const displayNum = board.display_counter + 1;

    await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id,
      { display_counter: displayNum }
    );

    const plan = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      ID.unique(),
      {
        board_id: id,
        display_num: displayNum,
        title: body.title,
        source_path: body.source_path,
        content: body.content,
        linked_items: "[]",
        created_at: new Date().toISOString(),
      }
    );

    return ok(plan, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/plans/plans.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/boards/\[id\]/plans/ __tests__/api/plans/plans.test.ts
git commit -m "feat: add GET + POST /api/boards/:id/plans with display counter"
```

---

### Task 9: GET /api/plans/:id

**Files:**
- Create: `obeya-cloud/app/api/plans/[id]/route.ts`
- Test: `obeya-cloud/__tests__/api/plans/plan-detail.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/plans/plan-detail.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

import { GET } from "@/app/api/plans/[id]/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/plans/:id", () => {
  const mockGetDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
    } as any);
  });

  it("returns plan with linked items resolved", async () => {
    mockGetDoc
      .mockResolvedValueOnce({
        $id: "plan-1",
        title: "Release Plan",
        board_id: "board-1",
        linked_items: JSON.stringify(["item-1", "item-2"]),
        content: "# Plan",
      })
      .mockResolvedValueOnce({
        $id: "item-1", title: "Task A", status: "done",
      })
      .mockResolvedValueOnce({
        $id: "item-2", title: "Task B", status: "in-progress",
      });

    const request = new Request("http://localhost/api/plans/plan-1");
    const response = await GET(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.plan.$id).toBe("plan-1");
    expect(body.data.linked_items).toHaveLength(2);
    expect(body.data.linked_items[0].title).toBe("Task A");
  });

  it("returns plan with empty linked items when none linked", async () => {
    mockGetDoc.mockResolvedValueOnce({
      $id: "plan-1",
      title: "Empty Plan",
      board_id: "board-1",
      linked_items: "[]",
      content: "# Empty",
    });

    const request = new Request("http://localhost/api/plans/plan-1");
    const response = await GET(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.linked_items).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/plans/plan-detail.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/plans/[id]/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const linkedItemIds: string[] = JSON.parse(plan.linked_items || "[]");

    const linkedItems = await resolveLinkedItems(db, env.APPWRITE_DATABASE_ID, linkedItemIds);

    return ok({ plan, linked_items: linkedItems });
  } catch (err) {
    return handleError(err);
  }
}

async function resolveLinkedItems(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  itemIds: string[]
): Promise<unknown[]> {
  if (itemIds.length === 0) return [];

  const results = await Promise.all(
    itemIds.map((itemId) =>
      db.getDocument(databaseId, COLLECTIONS.ITEMS, itemId).catch(() => null)
    )
  );

  return results.filter(Boolean);
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- __tests__/api/plans/plan-detail.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/api/plans/\[id\]/route.ts __tests__/api/plans/plan-detail.test.ts
git commit -m "feat: add GET /api/plans/:id with linked item resolution"
```

---

### Task 10: POST /api/plans/:id/link + DELETE /api/plans/:id/link/:iid

**Files:**
- Create: `obeya-cloud/app/api/plans/[id]/link/route.ts`
- Create: `obeya-cloud/app/api/plans/[id]/link/[iid]/route.ts`
- Test: `obeya-cloud/__tests__/api/plans/plan-link.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/plans/plan-link.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

import { POST } from "@/app/api/plans/[id]/link/route";
import { DELETE } from "@/app/api/plans/[id]/link/[iid]/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("POST /api/plans/:id/link", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("links item IDs to plan", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });
    mockUpdateDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1", "item-2", "item-3"]),
    });

    const request = new Request("http://localhost/api/plans/plan-1/link", {
      method: "POST",
      body: JSON.stringify({ item_ids: ["item-2", "item-3"] }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-1", "item-2", "item-3"]),
      })
    );
  });

  it("skips duplicate item IDs when linking", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });
    mockUpdateDoc.mockResolvedValue({ $id: "plan-1" });

    const request = new Request("http://localhost/api/plans/plan-1/link", {
      method: "POST",
      body: JSON.stringify({ item_ids: ["item-1", "item-2"] }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "plan-1" }) });
    expect(response.status).toBe(200);

    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-1", "item-2"]),
      })
    );
  });
});

describe("DELETE /api/plans/:id/link/:iid", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("removes item ID from linked_items", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1", "item-2"]),
    });
    mockUpdateDoc.mockResolvedValue({ $id: "plan-1" });

    const request = new Request("http://localhost/api/plans/plan-1/link/item-1", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "plan-1", iid: "item-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-2"]),
      })
    );
  });

  it("returns error when item ID not in linked_items", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });

    const request = new Request("http://localhost/api/plans/plan-1/link/nonexistent", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "plan-1", iid: "nonexistent" }),
    });
    const body = await response.json();

    expect(response.status).toBe(404);
    expect(body.ok).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- __tests__/api/plans/plan-link.test.ts
```

Expected: FAIL

- [ ] **Step 3: Write link implementation**

Create: `obeya-cloud/app/api/plans/[id]/link/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });
const bodySchema = z.object({
  item_ids: z.array(z.string().min(1)).min(1),
});

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const { item_ids } = await validateBody(request, bodySchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const existing: string[] = JSON.parse(plan.linked_items || "[]");
    const merged = [...new Set([...existing, ...item_ids])];

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id,
      { linked_items: JSON.stringify(merged) }
    );

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Write unlink implementation**

Create: `obeya-cloud/app/api/plans/[id]/link/[iid]/route.ts`

```typescript
import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";

const paramsSchema = z.object({
  id: z.string().min(1),
  iid: z.string().min(1),
});

export async function DELETE(
  request: NextRequest,
  context: { params: Promise<{ id: string; iid: string }> }
) {
  try {
    await authenticate(request);
    const { id, iid } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const linkedItems: string[] = JSON.parse(plan.linked_items || "[]");
    const index = linkedItems.indexOf(iid);

    if (index === -1) {
      throw new AppError(
        ErrorCode.ITEM_NOT_FOUND,
        `Item "${iid}" is not linked to this plan`
      );
    }

    linkedItems.splice(index, 1);

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id,
      { linked_items: JSON.stringify(linkedItems) }
    );

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
npm test -- __tests__/api/plans/plan-link.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/api/plans/\[id\]/link/ __tests__/api/plans/plan-link.test.ts
git commit -m "feat: add POST /api/plans/:id/link and DELETE /api/plans/:id/link/:iid"
```

---

## Plan 2 Summary (Part A + Part B)

| Chunk | Task | Endpoint | Status |
|-------|------|----------|--------|
| **1 (Part A)** | Board List | `GET /api/boards` | `- [ ]` |
| **1 (Part A)** | Board Create | `POST /api/boards` | `- [ ]` |
| **1 (Part A)** | Board Get | `GET /api/boards/:id` | `- [ ]` |
| **1 (Part A)** | Board Update | `PATCH /api/boards/:id` | `- [ ]` |
| **1 (Part A)** | Board Delete | `DELETE /api/boards/:id` | `- [ ]` |
| **2 (Part A)** | Item List | `GET /api/boards/:id/items` | `- [ ]` |
| **2 (Part A)** | Item Create | `POST /api/boards/:id/items` | `- [ ]` |
| **2 (Part A)** | Item Get | `GET /api/items/:id` | `- [ ]` |
| **2 (Part A)** | Item Update | `PATCH /api/items/:id` | `- [ ]` |
| **2 (Part A)** | Item Delete | `DELETE /api/items/:id` | `- [ ]` |
| **3 (Part B)** | History Helper | `lib/history.ts` | `- [ ]` |
| **3 (Part B)** | Item Move | `POST /api/items/:id/move` | `- [ ]` |
| **3 (Part B)** | Item Assign | `POST /api/items/:id/assign` | `- [ ]` |
| **3 (Part B)** | Item Block | `POST /api/items/:id/block` | `- [ ]` |
| **3 (Part B)** | Item Unblock | `DELETE /api/items/:id/block/:bid` | `- [ ]` |
| **4 (Part B)** | Item History | `GET /api/items/:id/history` | `- [ ]` |
| **4 (Part B)** | Board Activity | `GET /api/boards/:id/activity` | `- [ ]` |
| **5 (Part B)** | List Plans | `GET /api/boards/:id/plans` | `- [ ]` |
| **5 (Part B)** | Create Plan | `POST /api/boards/:id/plans` | `- [ ]` |
| **5 (Part B)** | Get Plan | `GET /api/plans/:id` | `- [ ]` |
| **5 (Part B)** | Link Items | `POST /api/plans/:id/link` | `- [ ]` |
| **5 (Part B)** | Unlink Item | `DELETE /api/plans/:id/link/:iid` | `- [ ]` |

**Next plan:** Plan 3 — Org & Board Sharing APIs (see `2026-03-12-obeya-cloud-orgs-sharing.md`)
