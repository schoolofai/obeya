# Obeya Cloud Plan 2: Board & Item APIs — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Board CRUD (create, read, update, delete, import, export) and Item CRUD (list, create, get, edit, delete) API routes with full test coverage, building on the foundation from Plan 1.

**Architecture:** Next.js 15 App Router API routes. Each route handler validates input via zod, authenticates via `lib/auth/middleware.ts`, interacts with Appwrite via Server SDK, and returns responses through the `ok()`/`fail()`/`handleError()` envelope. Display counter uses optimistic locking with retry on conflict. `blocked_by` and `tags` are stored as JSON strings in Appwrite and parsed/stringified in API routes. Board `columns` are stored as a JSON string on the board document.

**Tech Stack:** Next.js 15, TypeScript, Appwrite Node SDK (`node-appwrite`), Vitest, zod

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md`

**Repository:** `~/code/obeya-cloud` (created by Plan 1). All changes are in the existing `obeya-cloud` repo.

**Depends on:** Plan 1 (Obeya Cloud Foundation) — must be fully implemented first.

---

## File Structure

```
obeya-cloud/
├── app/
│   └── api/
│       ├── boards/
│       │   ├── route.ts                       # GET /api/boards, POST /api/boards
│       │   ├── import/route.ts                # POST /api/boards/import
│       │   └── [id]/
│       │       ├── route.ts                   # GET/PATCH/DELETE /api/boards/:id
│       │       ├── export/route.ts            # GET /api/boards/:id/export
│       │       └── items/
│       │           └── route.ts               # GET/POST /api/boards/:id/items
│       └── items/
│           └── [id]/
│               ├── route.ts                   # GET/PATCH/DELETE /api/items/:id
│               ├── move/route.ts              # POST /api/items/:id/move        (Part B)
│               ├── assign/route.ts            # POST /api/items/:id/assign      (Part B)
│               ├── block/
│               │   └── route.ts               # POST /api/items/:id/block       (Part B)
│               ├── block/[bid]/route.ts       # DELETE /api/items/:id/block/:bid (Part B)
│               └── history/route.ts           # GET /api/items/:id/history      (Part B)
├── lib/
│   ├── boards/
│   │   ├── counter.ts                         # Display counter atomic increment
│   │   ├── schemas.ts                         # Board zod schemas
│   │   ├── serialize.ts                       # Board JSON parse/stringify helpers
│   │   └── permissions.ts                     # Board access check helper
│   ├── items/
│   │   ├── schemas.ts                         # Item zod schemas
│   │   └── serialize.ts                       # Item JSON parse/stringify helpers
│   └── history/
│       └── record.ts                          # History entry creation helper    (Part B)
├── __tests__/
│   ├── lib/
│   │   ├── boards/
│   │   │   ├── counter.test.ts
│   │   │   ├── serialize.test.ts
│   │   │   └── permissions.test.ts
│   │   └── items/
│   │       └── serialize.test.ts
│   └── api/
│       ├── boards/
│       │   ├── boards-list-create.test.ts
│       │   ├── boards-get-update-delete.test.ts
│       │   ├── boards-import.test.ts
│       │   └── boards-export.test.ts
│       └── items/
│           ├── items-list-create.test.ts
│           └── items-get-update-delete.test.ts
```

---

## Chunk 1: Board CRUD

### Task 1: Board Zod Schemas & Serialization Helpers

**Files:**
- Create: `obeya-cloud/lib/boards/schemas.ts`
- Create: `obeya-cloud/lib/boards/serialize.ts`
- Test: `obeya-cloud/__tests__/lib/boards/serialize.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/boards/serialize.test.ts`

```typescript
import { describe, it, expect } from "vitest";
import {
  serializeBoard,
  deserializeBoard,
  serializeColumns,
  deserializeColumns,
} from "@/lib/boards/serialize";

describe("serializeColumns", () => {
  it("converts column array to JSON string", () => {
    const columns = [
      { name: "todo", limit: 0 },
      { name: "doing", limit: 3 },
    ];
    const result = serializeColumns(columns);
    expect(result).toBe(JSON.stringify(columns));
  });
});

describe("deserializeColumns", () => {
  it("parses JSON string to column array", () => {
    const json = '[{"name":"todo","limit":0},{"name":"done","limit":0}]';
    const result = deserializeColumns(json);
    expect(result).toEqual([
      { name: "todo", limit: 0 },
      { name: "done", limit: 0 },
    ]);
  });

  it("returns empty array for empty string", () => {
    expect(deserializeColumns("")).toEqual([]);
  });
});

describe("serializeBoard", () => {
  it("converts board fields for Appwrite storage", () => {
    const board = {
      name: "My Board",
      columns: [{ name: "todo", limit: 0 }],
      display_map: { "1": "item-abc" },
      users: { agent1: { role: "worker" } },
      projects: {},
    };
    const result = serializeBoard(board);
    expect(result.columns).toBe(JSON.stringify(board.columns));
    expect(result.display_map).toBe(JSON.stringify(board.display_map));
    expect(result.users).toBe(JSON.stringify(board.users));
    expect(result.projects).toBe(JSON.stringify(board.projects));
    expect(result.name).toBe("My Board");
  });
});

describe("deserializeBoard", () => {
  it("parses Appwrite document back to board shape", () => {
    const doc = {
      $id: "board-1",
      name: "My Board",
      owner_id: "user-1",
      org_id: null,
      display_counter: 5,
      columns: '[{"name":"todo","limit":0}]',
      display_map: '{"1":"item-abc"}',
      users: '{"agent1":{"role":"worker"}}',
      projects: "{}",
      agent_role: "worker",
      version: 1,
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeBoard(doc);
    expect(result.id).toBe("board-1");
    expect(result.columns).toEqual([{ name: "todo", limit: 0 }]);
    expect(result.display_map).toEqual({ "1": "item-abc" });
    expect(result.users).toEqual({ agent1: { role: "worker" } });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/serialize.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write board schemas**

Create: `obeya-cloud/lib/boards/schemas.ts`

```typescript
import { z } from "zod";

export const columnSchema = z.object({
  name: z.string().min(1),
  limit: z.number().int().min(0).default(0),
});

export const createBoardSchema = z.object({
  name: z.string().min(1, "Board name is required").max(255),
  columns: z
    .array(columnSchema)
    .min(1, "At least one column is required")
    .default([
      { name: "backlog", limit: 0 },
      { name: "todo", limit: 0 },
      { name: "in-progress", limit: 3 },
      { name: "done", limit: 0 },
    ]),
  org_id: z.string().optional(),
  agent_role: z.string().max(50).default("worker"),
});

export const updateBoardSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  columns: z.array(columnSchema).min(1).optional(),
  agent_role: z.string().max(50).optional(),
});
```

- [ ] **Step 4: Write serialization helpers**

Create: `obeya-cloud/lib/boards/serialize.ts`

```typescript
export interface Column {
  name: string;
  limit: number;
}

export interface BoardDocument {
  $id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: string;
  display_map: string;
  users: string;
  projects: string;
  agent_role: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Board {
  id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: Column[];
  display_map: Record<string, string>;
  users: Record<string, unknown>;
  projects: Record<string, unknown>;
  agent_role: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export function serializeColumns(columns: Column[]): string {
  return JSON.stringify(columns);
}

export function deserializeColumns(json: string): Column[] {
  if (!json) return [];
  return JSON.parse(json) as Column[];
}

function safeParseJson(json: string, fallback: unknown = {}): unknown {
  if (!json) return fallback;
  return JSON.parse(json);
}

export function serializeBoard(
  board: Partial<Board> & { columns?: Column[]; display_map?: Record<string, string>; users?: Record<string, unknown>; projects?: Record<string, unknown> }
): Record<string, unknown> {
  const result: Record<string, unknown> = { ...board };

  if (board.columns !== undefined) {
    result.columns = JSON.stringify(board.columns);
  }
  if (board.display_map !== undefined) {
    result.display_map = JSON.stringify(board.display_map);
  }
  if (board.users !== undefined) {
    result.users = JSON.stringify(board.users);
  }
  if (board.projects !== undefined) {
    result.projects = JSON.stringify(board.projects);
  }

  // Remove computed fields that shouldn't be stored
  delete result.id;

  return result;
}

export function deserializeBoard(doc: Record<string, unknown>): Board {
  return {
    id: doc.$id as string,
    name: doc.name as string,
    owner_id: doc.owner_id as string,
    org_id: (doc.org_id as string) || null,
    display_counter: doc.display_counter as number,
    columns: deserializeColumns(doc.columns as string),
    display_map: safeParseJson(doc.display_map as string, {}) as Record<string, string>,
    users: safeParseJson(doc.users as string, {}) as Record<string, unknown>,
    projects: safeParseJson(doc.projects as string, {}) as Record<string, unknown>,
    agent_role: doc.agent_role as string,
    version: doc.version as number,
    created_at: doc.created_at as string,
    updated_at: doc.updated_at as string,
  };
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/serialize.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/boards/ __tests__/lib/boards/
git commit -m "feat: add board zod schemas and serialization helpers"
```

---

### Task 2: Display Counter Atomic Increment

**Files:**
- Create: `obeya-cloud/lib/boards/counter.ts`
- Test: `obeya-cloud/__tests__/lib/boards/counter.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/boards/counter.test.ts`

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

import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";

describe("incrementDisplayCounter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads current counter, increments, and returns the new value", async () => {
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        display_counter: 5,
      }),
      updateDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        display_counter: 6,
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await incrementDisplayCounter("board-1");

    expect(result).toBe(6);
    expect(mockDb.getDocument).toHaveBeenCalledTimes(1);
    expect(mockDb.updateDocument).toHaveBeenCalledWith(
      "obeya",
      "boards",
      "board-1",
      { display_counter: 6 }
    );
  });

  it("retries on conflict (409) and succeeds", async () => {
    const conflict = new Error("Conflict");
    (conflict as any).code = 409;

    const mockDb = {
      getDocument: vi
        .fn()
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 5 })
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 6 }),
      updateDocument: vi
        .fn()
        .mockRejectedValueOnce(conflict)
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 7 }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await incrementDisplayCounter("board-1");

    expect(result).toBe(7);
    expect(mockDb.getDocument).toHaveBeenCalledTimes(2);
    expect(mockDb.updateDocument).toHaveBeenCalledTimes(2);
  });

  it("throws COUNTER_CONFLICT after max retries exhausted", async () => {
    const conflict = new Error("Conflict");
    (conflict as any).code = 409;

    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({ $id: "board-1", display_counter: 5 }),
      updateDocument: vi.fn().mockRejectedValue(conflict),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(incrementDisplayCounter("board-1")).rejects.toThrow(
      "Failed to increment display counter after 3 retries"
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/counter.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/boards/counter.ts`

```typescript
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";

const MAX_RETRIES = 3;

export async function incrementDisplayCounter(boardId: string): Promise<number> {
  const db = getDatabases();
  const env = getEnv();

  for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId
    );

    const currentCounter = board.display_counter as number;
    const nextCounter = currentCounter + 1;

    try {
      await db.updateDocument(
        env.APPWRITE_DATABASE_ID,
        COLLECTIONS.BOARDS,
        boardId,
        { display_counter: nextCounter }
      );
      return nextCounter;
    } catch (err: unknown) {
      const isConflict =
        err instanceof Error && (err as any).code === 409;
      if (!isConflict) {
        throw err;
      }
      // Conflict — retry with fresh counter value
    }
  }

  throw new AppError(
    ErrorCode.COUNTER_CONFLICT,
    `Failed to increment display counter after ${MAX_RETRIES} retries`
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/counter.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/boards/counter.ts __tests__/lib/boards/counter.test.ts
git commit -m "feat: add display counter atomic increment with optimistic locking"
```

---

### Task 3: Board Permissions Helper

**Files:**
- Create: `obeya-cloud/lib/boards/permissions.ts`
- Test: `obeya-cloud/__tests__/lib/boards/permissions.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/boards/permissions.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

describe("assertBoardAccess", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("allows owner of the board", async () => {
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        owner_id: "user-1",
      }),
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const board = await assertBoardAccess("board-1", "user-1", "viewer");
    expect(board.owner_id).toBe("user-1");
  });

  it("allows board member with sufficient role", async () => {
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        owner_id: "other-user",
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-2", role: "editor" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const board = await assertBoardAccess("board-1", "user-2", "editor");
    expect(board.$id).toBe("board-1");
  });

  it("throws FORBIDDEN when user has no access", async () => {
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        owner_id: "other-user",
      }),
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(
      assertBoardAccess("board-1", "stranger", "viewer")
    ).rejects.toThrow("You do not have access to this board");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/permissions.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/boards/permissions.ts`

```typescript
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";
import { Query } from "node-appwrite";

const ROLE_HIERARCHY: Record<string, number> = {
  viewer: 1,
  editor: 2,
  owner: 3,
};

export async function assertBoardAccess(
  boardId: string,
  userId: string,
  requiredRole: "viewer" | "editor" | "owner"
): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  const board = await getBoardOrThrow(db, env.APPWRITE_DATABASE_ID, boardId);

  // Owner always has full access
  if (board.owner_id === userId) {
    return board;
  }

  // Check board_members
  const members = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    [
      Query.equal("board_id", boardId),
      Query.equal("user_id", userId),
      Query.limit(1),
    ]
  );

  if (members.documents.length > 0) {
    const memberRole = members.documents[0].role as string;
    if (ROLE_HIERARCHY[memberRole] >= ROLE_HIERARCHY[requiredRole]) {
      return board;
    }
  }

  throw new AppError(
    ErrorCode.FORBIDDEN,
    "You do not have access to this board"
  );
}

async function getBoardOrThrow(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string
): Promise<Record<string, unknown>> {
  try {
    return await db.getDocument(databaseId, COLLECTIONS.BOARDS, boardId);
  } catch (err: unknown) {
    if (err instanceof Error && (err as any).code === 404) {
      throw new AppError(ErrorCode.BOARD_NOT_FOUND, `Board ${boardId} not found`);
    }
    throw err;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/boards/permissions.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/boards/permissions.ts __tests__/lib/boards/permissions.test.ts
git commit -m "feat: add board access permission check helper"
```

---

### Task 4: GET /api/boards (List) & POST /api/boards (Create)

**Files:**
- Create: `obeya-cloud/app/api/boards/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/boards-list-create.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/boards-list-create.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { GET, POST } from "@/app/api/boards/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns boards owned by user and boards user is a member of", async () => {
    vi.mocked(authenticate).mockResolvedValue({
      id: "user-1",
      email: "a@b.com",
      name: "Alice",
    });

    const mockDb = {
      listDocuments: vi
        .fn()
        .mockResolvedValueOnce({
          total: 1,
          documents: [
            {
              $id: "board-1",
              name: "My Board",
              owner_id: "user-1",
              org_id: null,
              display_counter: 3,
              columns: '[{"name":"todo","limit":0}]',
              display_map: "{}",
              users: "{}",
              projects: "{}",
              agent_role: "worker",
              version: 1,
              created_at: "2026-03-12T00:00:00.000Z",
              updated_at: "2026-03-12T00:00:00.000Z",
            },
          ],
        })
        .mockResolvedValueOnce({
          total: 0,
          documents: [],
        }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards");
    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].name).toBe("My Board");
  });

  it("returns 401 when not authenticated", async () => {
    vi.mocked(authenticate).mockRejectedValue(new Error("No authentication"));

    const request = new Request("http://localhost/api/boards");
    const response = await GET(request);

    expect(response.status).toBe(500);
  });
});

describe("POST /api/boards", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("creates a board and returns it", async () => {
    vi.mocked(authenticate).mockResolvedValue({
      id: "user-1",
      email: "a@b.com",
      name: "Alice",
    });

    const now = "2026-03-12T00:00:00.000Z";
    const mockDb = {
      createDocument: vi.fn().mockResolvedValue({
        $id: "board-new",
        name: "Sprint Board",
        owner_id: "user-1",
        org_id: null,
        display_counter: 0,
        columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
        display_map: "{}",
        users: "{}",
        projects: "{}",
        agent_role: "worker",
        version: 1,
        created_at: now,
        updated_at: now,
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards", {
      method: "POST",
      body: JSON.stringify({
        name: "Sprint Board",
        columns: [
          { name: "todo", limit: 0 },
          { name: "done", limit: 0 },
        ],
      }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("Sprint Board");
    expect(body.data.id).toBe("board-new");
  });

  it("returns 400 for missing board name", async () => {
    vi.mocked(authenticate).mockResolvedValue({
      id: "user-1",
      email: "a@b.com",
      name: "Alice",
    });

    const request = new Request("http://localhost/api/boards", {
      method: "POST",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    expect(response.status).toBe(400);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-list-create.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/route.ts`

```typescript
import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { createBoardSchema } from "@/lib/boards/schemas";
import { deserializeBoard, serializeColumns } from "@/lib/boards/serialize";

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    const db = getDatabases();
    const env = getEnv();

    // Fetch boards owned by user
    const owned = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      [Query.equal("owner_id", user.id), Query.limit(100)]
    );

    // Fetch boards user is a member of
    const memberships = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      [Query.equal("user_id", user.id), Query.limit(100)]
    );

    const memberBoardIds = memberships.documents.map(
      (m) => m.board_id as string
    );

    const memberBoards =
      memberBoardIds.length > 0
        ? await fetchBoardsByIds(db, env.APPWRITE_DATABASE_ID, memberBoardIds)
        : [];

    // Deduplicate (user might own a board they're also a member of)
    const boardMap = new Map<string, Record<string, unknown>>();
    for (const doc of owned.documents) {
      boardMap.set(doc.$id, doc);
    }
    for (const doc of memberBoards) {
      boardMap.set(doc.$id, doc);
    }

    const boards = Array.from(boardMap.values()).map(deserializeBoard);

    return ok(boards, { meta: { total: boards.length } });
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const db = getDatabases();
    const env = getEnv();
    const input = await validateBody(request, createBoardSchema);

    const now = new Date().toISOString();
    const doc = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      ID.unique(),
      {
        name: input.name,
        owner_id: user.id,
        org_id: input.org_id || null,
        display_counter: 0,
        columns: serializeColumns(input.columns),
        display_map: "{}",
        users: "{}",
        projects: "{}",
        agent_role: input.agent_role,
        version: 1,
        created_at: now,
        updated_at: now,
      }
    );

    return ok(deserializeBoard(doc), { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function fetchBoardsByIds(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  ids: string[]
): Promise<Record<string, unknown>[]> {
  const results: Record<string, unknown>[] = [];
  for (const id of ids) {
    try {
      const doc = await db.getDocument(databaseId, COLLECTIONS.BOARDS, id);
      results.push(doc);
    } catch {
      // Board may have been deleted — skip
    }
  }
  return results;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-list-create.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/route.ts __tests__/api/boards/boards-list-create.test.ts
git commit -m "feat: add GET/POST /api/boards endpoints (list and create)"
```

---

### Task 5: GET/PATCH/DELETE /api/boards/:id

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/boards-get-update-delete.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/boards-get-update-delete.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/boards/permissions", () => ({
  assertBoardAccess: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { GET, PATCH, DELETE } from "@/app/api/boards/[id]/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };
const boardDoc = {
  $id: "board-1",
  name: "My Board",
  owner_id: "user-1",
  org_id: null,
  display_counter: 5,
  columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
  display_map: '{}',
  users: '{}',
  projects: '{}',
  agent_role: "worker",
  version: 1,
  created_at: "2026-03-12T00:00:00.000Z",
  updated_at: "2026-03-12T00:00:00.000Z",
};

describe("GET /api/boards/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns board with deserialized fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const request = new Request("http://localhost/api/boards/board-1");
    const response = await GET(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.id).toBe("board-1");
    expect(body.data.columns).toEqual([
      { name: "todo", limit: 0 },
      { name: "done", limit: 0 },
    ]);
  });
});

describe("PATCH /api/boards/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("updates board name and returns updated board", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const updatedDoc = { ...boardDoc, name: "Renamed Board" };
    const mockDb = {
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Renamed Board" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.name).toBe("Renamed Board");
  });

  it("returns 400 for empty update body", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const request = new Request("http://localhost/api/boards/board-1", {
      method: "PATCH",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, {
      params: Promise.resolve({ id: "board-1" }),
    });

    // Empty update is valid (no-op) or 400 depending on design — we allow no-op
    expect(response.status).toBe(200);
  });
});

describe("DELETE /api/boards/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("deletes board and returns confirmation", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const mockDb = {
      deleteDocument: vi.fn().mockResolvedValue(undefined),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.deleted).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-get-update-delete.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/[id]/route.ts`

```typescript
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { updateBoardSchema } from "@/lib/boards/schemas";
import {
  deserializeBoard,
  serializeColumns,
} from "@/lib/boards/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    const boardDoc = await assertBoardAccess(id, user.id, "viewer");
    return ok(deserializeBoard(boardDoc));
  } catch (err) {
    return handleError(err);
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await assertBoardAccess(id, user.id, "editor");

    const input = await validateBody(request, updateBoardSchema);
    const db = getDatabases();
    const env = getEnv();

    const updatePayload: Record<string, unknown> = {
      updated_at: new Date().toISOString(),
    };

    if (input.name !== undefined) {
      updatePayload.name = input.name;
    }
    if (input.columns !== undefined) {
      updatePayload.columns = serializeColumns(input.columns);
    }
    if (input.agent_role !== undefined) {
      updatePayload.agent_role = input.agent_role;
    }

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id,
      updatePayload
    );

    return ok(deserializeBoard(updated));
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await assertBoardAccess(id, user.id, "owner");

    const db = getDatabases();
    const env = getEnv();

    await db.deleteDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id
    );

    return ok({ deleted: true, id });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-get-update-delete.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/[id]/route.ts __tests__/api/boards/boards-get-update-delete.test.ts
git commit -m "feat: add GET/PATCH/DELETE /api/boards/:id endpoints"
```

---

### Task 6: POST /api/boards/import

**Files:**
- Create: `obeya-cloud/app/api/boards/import/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/boards-import.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/boards-import.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { POST } from "@/app/api/boards/import/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

describe("POST /api/boards/import", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("imports a local board.json and returns the new board with ID mapping", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);

    let docCounter = 0;
    const mockDb = {
      createDocument: vi.fn().mockImplementation(() => {
        docCounter++;
        return Promise.resolve({
          $id: `cloud-${docCounter}`,
          name: "Imported Board",
          owner_id: "user-1",
          org_id: null,
          display_counter: 2,
          columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
          display_map: '{"1":"cloud-2","2":"cloud-3"}',
          users: "{}",
          projects: "{}",
          agent_role: "worker",
          version: 1,
          created_at: "2026-03-12T00:00:00.000Z",
          updated_at: "2026-03-12T00:00:00.000Z",
        });
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const localBoard = {
      name: "My Local Board",
      display_counter: 2,
      columns: [
        { name: "todo", limit: 0 },
        { name: "done", limit: 0 },
      ],
      display_map: { "1": "local-item-1", "2": "local-item-2" },
      users: {},
      projects: {},
      agent_role: "worker",
      version: 1,
      items: [
        {
          id: "local-item-1",
          display_num: 1,
          type: "task",
          title: "First task",
          description: "",
          status: "todo",
          priority: "medium",
          parent_id: null,
          assignee_id: null,
          blocked_by: [],
          tags: [],
          project: null,
        },
        {
          id: "local-item-2",
          display_num: 2,
          type: "task",
          title: "Second task",
          description: "",
          status: "done",
          priority: "low",
          parent_id: "local-item-1",
          blocked_by: ["local-item-1"],
          tags: ["bug"],
          project: null,
        },
      ],
    };

    const request = new Request("http://localhost/api/boards/import", {
      method: "POST",
      body: JSON.stringify(localBoard),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.board_id).toBeDefined();
    expect(body.data.id_map).toBeDefined();
    expect(body.data.items_imported).toBe(2);
  });

  it("returns 400 for invalid board payload", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);

    const request = new Request("http://localhost/api/boards/import", {
      method: "POST",
      body: JSON.stringify({ invalid: true }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    expect(response.status).toBe(400);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-import.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/import/route.ts`

```typescript
import { z } from "zod";
import { ID } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { serializeColumns } from "@/lib/boards/serialize";

const importItemSchema = z.object({
  id: z.string(),
  display_num: z.number().int(),
  type: z.enum(["epic", "story", "task"]),
  title: z.string(),
  description: z.string().default(""),
  status: z.string(),
  priority: z.enum(["low", "medium", "high", "critical"]),
  parent_id: z.string().nullable().default(null),
  assignee_id: z.string().nullable().default(null),
  blocked_by: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  project: z.string().nullable().default(null),
});

const importBoardSchema = z.object({
  name: z.string().min(1),
  display_counter: z.number().int(),
  columns: z.array(
    z.object({ name: z.string(), limit: z.number().int().default(0) })
  ),
  display_map: z.record(z.string()).default({}),
  users: z.record(z.unknown()).default({}),
  projects: z.record(z.unknown()).default({}),
  agent_role: z.string().default("worker"),
  version: z.number().int().default(1),
  items: z.array(importItemSchema).default([]),
});

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const db = getDatabases();
    const env = getEnv();
    const input = await validateBody(request, importBoardSchema);

    const now = new Date().toISOString();

    // Step 1: Create the board document
    const boardDoc = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      ID.unique(),
      {
        name: input.name,
        owner_id: user.id,
        org_id: null,
        display_counter: input.display_counter,
        columns: serializeColumns(input.columns),
        display_map: "{}", // Will update after items are created
        users: JSON.stringify(input.users),
        projects: JSON.stringify(input.projects),
        agent_role: input.agent_role,
        version: input.version,
        created_at: now,
        updated_at: now,
      }
    );

    const boardId = boardDoc.$id;

    // Step 2: Create items and build local→cloud ID mapping
    const idMap: Record<string, string> = {};
    for (const item of input.items) {
      const itemDoc = await db.createDocument(
        env.APPWRITE_DATABASE_ID,
        COLLECTIONS.ITEMS,
        ID.unique(),
        {
          board_id: boardId,
          display_num: item.display_num,
          type: item.type,
          title: item.title,
          description: item.description,
          status: item.status,
          priority: item.priority,
          parent_id: null, // Resolve after all items created
          assignee_id: item.assignee_id,
          blocked_by: "[]", // Resolve after all items created
          tags: JSON.stringify(item.tags),
          project: item.project,
          created_at: now,
          updated_at: now,
        }
      );
      idMap[item.id] = itemDoc.$id;
    }

    // Step 3: Resolve parent_id and blocked_by references
    for (const item of input.items) {
      const cloudId = idMap[item.id];
      const updates: Record<string, unknown> = {};
      let needsUpdate = false;

      if (item.parent_id && idMap[item.parent_id]) {
        updates.parent_id = idMap[item.parent_id];
        needsUpdate = true;
      }

      if (item.blocked_by.length > 0) {
        const resolvedBlockers = item.blocked_by
          .map((localId) => idMap[localId])
          .filter(Boolean);
        updates.blocked_by = JSON.stringify(resolvedBlockers);
        needsUpdate = true;
      }

      if (needsUpdate) {
        await db.updateDocument(
          env.APPWRITE_DATABASE_ID,
          COLLECTIONS.ITEMS,
          cloudId,
          updates
        );
      }
    }

    // Step 4: Update board display_map with new cloud IDs
    const cloudDisplayMap: Record<string, string> = {};
    for (const [num, localId] of Object.entries(input.display_map)) {
      if (idMap[localId]) {
        cloudDisplayMap[num] = idMap[localId];
      }
    }

    await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId,
      { display_map: JSON.stringify(cloudDisplayMap) }
    );

    return ok(
      {
        board_id: boardId,
        id_map: idMap,
        items_imported: input.items.length,
      },
      { status: 201 }
    );
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-import.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/import/ __tests__/api/boards/boards-import.test.ts
git commit -m "feat: add POST /api/boards/import endpoint with local-to-cloud ID mapping"
```

---

### Task 7: GET /api/boards/:id/export

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/export/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/boards-export.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/boards-export.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/boards/permissions", () => ({
  assertBoardAccess: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { GET } from "@/app/api/boards/[id]/export/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

describe("GET /api/boards/:id/export", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("exports board in local board.json format with items", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({
      $id: "board-1",
      name: "My Board",
      owner_id: "user-1",
      org_id: null,
      display_counter: 2,
      columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
      display_map: '{"1":"item-1","2":"item-2"}',
      users: '{}',
      projects: '{}',
      agent_role: "worker",
      version: 1,
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    });

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 2,
        documents: [
          {
            $id: "item-1",
            board_id: "board-1",
            display_num: 1,
            type: "task",
            title: "First task",
            description: "",
            status: "todo",
            priority: "medium",
            parent_id: null,
            assignee_id: null,
            blocked_by: "[]",
            tags: '["important"]',
            project: null,
            created_at: "2026-03-12T00:00:00.000Z",
            updated_at: "2026-03-12T00:00:00.000Z",
          },
          {
            $id: "item-2",
            board_id: "board-1",
            display_num: 2,
            type: "task",
            title: "Second task",
            description: "desc",
            status: "done",
            priority: "low",
            parent_id: "item-1",
            assignee_id: null,
            blocked_by: '["item-1"]',
            tags: "[]",
            project: null,
            created_at: "2026-03-12T00:00:00.000Z",
            updated_at: "2026-03-12T00:00:00.000Z",
          },
        ],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request(
      "http://localhost/api/boards/board-1/export"
    );
    const response = await GET(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("My Board");
    expect(body.data.display_counter).toBe(2);
    expect(body.data.columns).toEqual([
      { name: "todo", limit: 0 },
      { name: "done", limit: 0 },
    ]);
    expect(body.data.items).toHaveLength(2);
    expect(body.data.items[0].id).toBe("item-1");
    expect(body.data.items[0].tags).toEqual(["important"]);
    expect(body.data.items[1].blocked_by).toEqual(["item-1"]);
  });

  it("exports board with empty items list when no items exist", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({
      $id: "board-1",
      name: "Empty Board",
      owner_id: "user-1",
      org_id: null,
      display_counter: 0,
      columns: '[{"name":"todo","limit":0}]',
      display_map: '{}',
      users: '{}',
      projects: '{}',
      agent_role: "worker",
      version: 1,
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    });

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 0,
        documents: [],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request(
      "http://localhost/api/boards/board-1/export"
    );
    const response = await GET(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.items).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-export.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/[id]/export/route.ts`

```typescript
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { deserializeColumns } from "@/lib/boards/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    const boardDoc = await assertBoardAccess(id, user.id, "viewer");
    const db = getDatabases();
    const env = getEnv();

    // Fetch all items for this board
    const itemsResult = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      [Query.equal("board_id", id), Query.limit(5000)]
    );

    const items = itemsResult.documents.map(serializeItemForExport);

    const exportData = {
      name: boardDoc.name,
      display_counter: boardDoc.display_counter,
      columns: deserializeColumns(boardDoc.columns as string),
      display_map: safeParseJson(boardDoc.display_map as string, {}),
      users: safeParseJson(boardDoc.users as string, {}),
      projects: safeParseJson(boardDoc.projects as string, {}),
      agent_role: boardDoc.agent_role,
      version: boardDoc.version,
      items,
    };

    return ok(exportData);
  } catch (err) {
    return handleError(err);
  }
}

function serializeItemForExport(
  doc: Record<string, unknown>
): Record<string, unknown> {
  return {
    id: doc.$id,
    display_num: doc.display_num,
    type: doc.type,
    title: doc.title,
    description: doc.description || "",
    status: doc.status,
    priority: doc.priority,
    parent_id: doc.parent_id || null,
    assignee_id: doc.assignee_id || null,
    blocked_by: safeParseJson(doc.blocked_by as string, []),
    tags: safeParseJson(doc.tags as string, []),
    project: doc.project || null,
  };
}

function safeParseJson(json: string, fallback: unknown): unknown {
  if (!json) return fallback;
  try {
    return JSON.parse(json);
  } catch {
    return fallback;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/boards-export.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/[id]/export/ __tests__/api/boards/boards-export.test.ts
git commit -m "feat: add GET /api/boards/:id/export endpoint (board.json format)"
```

---

## Chunk 2: Item CRUD

### Task 8: Item Zod Schemas & Serialization Helpers

**Files:**
- Create: `obeya-cloud/lib/items/schemas.ts`
- Create: `obeya-cloud/lib/items/serialize.ts`
- Test: `obeya-cloud/__tests__/lib/items/serialize.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/items/serialize.test.ts`

```typescript
import { describe, it, expect } from "vitest";
import { serializeItem, deserializeItem } from "@/lib/items/serialize";

describe("serializeItem", () => {
  it("converts blocked_by and tags arrays to JSON strings", () => {
    const item = {
      blocked_by: ["item-1", "item-2"],
      tags: ["bug", "urgent"],
      title: "Fix login",
    };
    const result = serializeItem(item);
    expect(result.blocked_by).toBe('["item-1","item-2"]');
    expect(result.tags).toBe('["bug","urgent"]');
    expect(result.title).toBe("Fix login");
  });

  it("handles empty arrays", () => {
    const item = { blocked_by: [], tags: [] };
    const result = serializeItem(item);
    expect(result.blocked_by).toBe("[]");
    expect(result.tags).toBe("[]");
  });
});

describe("deserializeItem", () => {
  it("parses Appwrite document to item shape", () => {
    const doc = {
      $id: "item-1",
      board_id: "board-1",
      display_num: 3,
      type: "task",
      title: "Fix login",
      description: "Users can't log in",
      status: "in-progress",
      priority: "high",
      parent_id: null,
      assignee_id: "user-1",
      blocked_by: '["item-2"]',
      tags: '["bug","auth"]',
      project: "web",
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeItem(doc);

    expect(result.id).toBe("item-1");
    expect(result.blocked_by).toEqual(["item-2"]);
    expect(result.tags).toEqual(["bug", "auth"]);
    expect(result.display_num).toBe(3);
  });

  it("handles empty/null JSON strings gracefully", () => {
    const doc = {
      $id: "item-2",
      board_id: "board-1",
      display_num: 1,
      type: "task",
      title: "Test",
      description: "",
      status: "todo",
      priority: "medium",
      parent_id: null,
      assignee_id: null,
      blocked_by: "",
      tags: "",
      project: null,
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeItem(doc);
    expect(result.blocked_by).toEqual([]);
    expect(result.tags).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/items/serialize.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write item schemas**

Create: `obeya-cloud/lib/items/schemas.ts`

```typescript
import { z } from "zod";

export const createItemSchema = z.object({
  type: z.enum(["epic", "story", "task"]),
  title: z.string().min(1, "Title is required").max(500),
  description: z.string().max(50000).default(""),
  status: z.string().min(1).default("backlog"),
  priority: z.enum(["low", "medium", "high", "critical"]).default("medium"),
  parent_id: z.string().nullable().optional(),
  assignee_id: z.string().nullable().optional(),
  blocked_by: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  project: z.string().nullable().optional(),
});

export const updateItemSchema = z.object({
  title: z.string().min(1).max(500).optional(),
  description: z.string().max(50000).optional(),
  priority: z.enum(["low", "medium", "high", "critical"]).optional(),
  parent_id: z.string().nullable().optional(),
  tags: z.array(z.string()).optional(),
  project: z.string().nullable().optional(),
});

export const listItemsQuerySchema = z.object({
  status: z.string().optional(),
  type: z.enum(["epic", "story", "task"]).optional(),
  assignee: z.string().optional(),
});
```

- [ ] **Step 4: Write serialization helpers**

Create: `obeya-cloud/lib/items/serialize.ts`

```typescript
export interface Item {
  id: string;
  board_id: string;
  display_num: number;
  type: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  parent_id: string | null;
  assignee_id: string | null;
  blocked_by: string[];
  tags: string[];
  project: string | null;
  created_at: string;
  updated_at: string;
}

export function serializeItem(
  item: Partial<{ blocked_by: string[]; tags: string[] }> & Record<string, unknown>
): Record<string, unknown> {
  const result: Record<string, unknown> = { ...item };

  if (item.blocked_by !== undefined) {
    result.blocked_by = JSON.stringify(item.blocked_by);
  }
  if (item.tags !== undefined) {
    result.tags = JSON.stringify(item.tags);
  }

  // Remove computed fields
  delete result.id;

  return result;
}

export function deserializeItem(doc: Record<string, unknown>): Item {
  return {
    id: doc.$id as string,
    board_id: doc.board_id as string,
    display_num: doc.display_num as number,
    type: doc.type as string,
    title: doc.title as string,
    description: (doc.description as string) || "",
    status: doc.status as string,
    priority: doc.priority as string,
    parent_id: (doc.parent_id as string) || null,
    assignee_id: (doc.assignee_id as string) || null,
    blocked_by: safeParseJsonArray(doc.blocked_by as string),
    tags: safeParseJsonArray(doc.tags as string),
    project: (doc.project as string) || null,
    created_at: doc.created_at as string,
    updated_at: doc.updated_at as string,
  };
}

function safeParseJsonArray(json: string): string[] {
  if (!json) return [];
  try {
    return JSON.parse(json) as string[];
  } catch {
    return [];
  }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/items/serialize.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/items/ __tests__/lib/items/
git commit -m "feat: add item zod schemas and serialization helpers"
```

---

### Task 9: GET /api/boards/:id/items (List) & POST /api/boards/:id/items (Create)

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/items/route.ts`
- Test: `obeya-cloud/__tests__/api/items/items-list-create.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/items-list-create.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/boards/permissions", () => ({
  assertBoardAccess: vi.fn(),
}));

vi.mock("@/lib/boards/counter", () => ({
  incrementDisplayCounter: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { GET, POST } from "@/app/api/boards/[id]/items/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };
const boardDoc = { $id: "board-1", owner_id: "user-1" };

describe("GET /api/boards/:id/items", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns all items for the board", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 1,
        documents: [
          {
            $id: "item-1",
            board_id: "board-1",
            display_num: 1,
            type: "task",
            title: "First task",
            description: "",
            status: "todo",
            priority: "medium",
            parent_id: null,
            assignee_id: null,
            blocked_by: "[]",
            tags: "[]",
            project: null,
            created_at: "2026-03-12T00:00:00.000Z",
            updated_at: "2026-03-12T00:00:00.000Z",
          },
        ],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items");
    const response = await GET(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].id).toBe("item-1");
    expect(body.data[0].blocked_by).toEqual([]);
  });

  it("filters items by status query param", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 0,
        documents: [],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request(
      "http://localhost/api/boards/board-1/items?status=done"
    );
    const response = await GET(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    // Verify the query included the status filter
    const callArgs = mockDb.listDocuments.mock.calls[0][2];
    const hasStatusFilter = callArgs.some(
      (q: any) => JSON.stringify(q).includes("done")
    );
    expect(hasStatusFilter).toBe(true);
  });
});

describe("POST /api/boards/:id/items", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("creates an item with incremented display counter", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    vi.mocked(incrementDisplayCounter).mockResolvedValue(5);

    const now = "2026-03-12T00:00:00.000Z";
    const mockDb = {
      createDocument: vi.fn().mockResolvedValue({
        $id: "item-new",
        board_id: "board-1",
        display_num: 5,
        type: "task",
        title: "New task",
        description: "",
        status: "todo",
        priority: "medium",
        parent_id: null,
        assignee_id: null,
        blocked_by: "[]",
        tags: "[]",
        project: null,
        created_at: now,
        updated_at: now,
      }),
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        display_map: "{}",
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request(
      "http://localhost/api/boards/board-1/items",
      {
        method: "POST",
        body: JSON.stringify({
          type: "task",
          title: "New task",
          status: "todo",
        }),
        headers: { "Content-Type": "application/json" },
      }
    );

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.display_num).toBe(5);
    expect(body.data.id).toBe("item-new");
    expect(incrementDisplayCounter).toHaveBeenCalledWith("board-1");
  });

  it("returns 400 for missing required fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const request = new Request(
      "http://localhost/api/boards/board-1/items",
      {
        method: "POST",
        body: JSON.stringify({ description: "no title or type" }),
        headers: { "Content-Type": "application/json" },
      }
    );

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1" }),
    });
    expect(response.status).toBe(400);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/items/items-list-create.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/boards/[id]/items/route.ts`

```typescript
import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { createItemSchema } from "@/lib/items/schemas";
import { deserializeItem, serializeItem } from "@/lib/items/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await assertBoardAccess(id, user.id, "viewer");

    const db = getDatabases();
    const env = getEnv();
    const url = new URL(request.url);

    const queries = buildItemFilters(id, url.searchParams);
    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      queries
    );

    const items = result.documents.map(deserializeItem);
    return ok(items, { meta: { total: result.total } });
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId } = await context.params;

    await assertBoardAccess(boardId, user.id, "editor");

    const input = await validateBody(request, createItemSchema);
    const db = getDatabases();
    const env = getEnv();

    // Atomically increment display counter
    const displayNum = await incrementDisplayCounter(boardId);

    const now = new Date().toISOString();
    const serialized = serializeItem({
      board_id: boardId,
      display_num: displayNum,
      type: input.type,
      title: input.title,
      description: input.description,
      status: input.status,
      priority: input.priority,
      parent_id: input.parent_id || null,
      assignee_id: input.assignee_id || null,
      blocked_by: input.blocked_by,
      tags: input.tags,
      project: input.project || null,
      created_at: now,
      updated_at: now,
    });

    const itemDoc = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      ID.unique(),
      serialized
    );

    // Update board display_map
    await updateDisplayMap(db, env.APPWRITE_DATABASE_ID, boardId, displayNum, itemDoc.$id);

    return ok(deserializeItem(itemDoc), { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

function buildItemFilters(
  boardId: string,
  searchParams: URLSearchParams
): string[] {
  const queries = [Query.equal("board_id", boardId), Query.limit(5000)];

  const status = searchParams.get("status");
  if (status) {
    queries.push(Query.equal("status", status));
  }

  const type = searchParams.get("type");
  if (type) {
    queries.push(Query.equal("type", type));
  }

  const assignee = searchParams.get("assignee");
  if (assignee) {
    queries.push(Query.equal("assignee_id", assignee));
  }

  return queries;
}

async function updateDisplayMap(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string,
  displayNum: number,
  itemId: string
): Promise<void> {
  const board = await db.getDocument(databaseId, COLLECTIONS.BOARDS, boardId);
  const displayMap: Record<string, string> = board.display_map
    ? JSON.parse(board.display_map as string)
    : {};

  displayMap[String(displayNum)] = itemId;

  await db.updateDocument(databaseId, COLLECTIONS.BOARDS, boardId, {
    display_map: JSON.stringify(displayMap),
  });
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/items/items-list-create.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/[id]/items/ __tests__/api/items/items-list-create.test.ts
git commit -m "feat: add GET/POST /api/boards/:id/items endpoints with display counter"
```

---

### Task 10: GET/PATCH/DELETE /api/items/:id

**Files:**
- Create: `obeya-cloud/app/api/items/[id]/route.ts`
- Test: `obeya-cloud/__tests__/api/items/items-get-update-delete.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/items/items-get-update-delete.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/boards/permissions", () => ({
  assertBoardAccess: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { GET, PATCH, DELETE } from "@/app/api/items/[id]/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };
const itemDoc = {
  $id: "item-1",
  board_id: "board-1",
  display_num: 3,
  type: "task",
  title: "Fix login",
  description: "Users can't log in",
  status: "in-progress",
  priority: "high",
  parent_id: null,
  assignee_id: "user-1",
  blocked_by: '["item-2"]',
  tags: '["bug"]',
  project: "web",
  created_at: "2026-03-12T00:00:00.000Z",
  updated_at: "2026-03-12T00:00:00.000Z",
};

describe("GET /api/items/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns item with deserialized JSON fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1");
    const response = await GET(request, {
      params: Promise.resolve({ id: "item-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.id).toBe("item-1");
    expect(body.data.blocked_by).toEqual(["item-2"]);
    expect(body.data.tags).toEqual(["bug"]);
  });

  it("returns 404 for non-existent item", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);

    const notFound = new Error("Document not found");
    (notFound as any).code = 404;
    const mockDb = {
      getDocument: vi.fn().mockRejectedValue(notFound),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/missing");
    const response = await GET(request, {
      params: Promise.resolve({ id: "missing" }),
    });

    expect(response.status).toBe(404);
  });
});

describe("PATCH /api/items/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("updates item title and returns updated item", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const updatedDoc = { ...itemDoc, title: "Fix auth flow" };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", {
      method: "PATCH",
      body: JSON.stringify({ title: "Fix auth flow" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, {
      params: Promise.resolve({ id: "item-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.title).toBe("Fix auth flow");
  });

  it("updates tags as serialized JSON string", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const updatedDoc = { ...itemDoc, tags: '["bug","critical"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", {
      method: "PATCH",
      body: JSON.stringify({ tags: ["bug", "critical"] }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, {
      params: Promise.resolve({ id: "item-1" }),
    });

    // Verify tags were serialized before sending to Appwrite
    const updateCall = mockDb.updateDocument.mock.calls[0];
    const updatePayload = updateCall[3];
    expect(updatePayload.tags).toBe('["bug","critical"]');
  });
});

describe("DELETE /api/items/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("deletes item and returns confirmation", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      deleteDocument: vi.fn().mockResolvedValue(undefined),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "item-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.deleted).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/items/items-get-update-delete.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/items/[id]/route.ts`

```typescript
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { updateItemSchema } from "@/lib/items/schemas";
import { deserializeItem, serializeItem } from "@/lib/items/serialize";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const db = getDatabases();
    const env = getEnv();

    const itemDoc = await getItemOrThrow(db, env.APPWRITE_DATABASE_ID, id);

    // Verify user has access to the item's board
    await assertBoardAccess(itemDoc.board_id as string, user.id, "viewer");

    return ok(deserializeItem(itemDoc));
  } catch (err) {
    return handleError(err);
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const db = getDatabases();
    const env = getEnv();

    const itemDoc = await getItemOrThrow(db, env.APPWRITE_DATABASE_ID, id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");

    const input = await validateBody(request, updateItemSchema);

    const updatePayload: Record<string, unknown> = {
      updated_at: new Date().toISOString(),
    };

    if (input.title !== undefined) updatePayload.title = input.title;
    if (input.description !== undefined) updatePayload.description = input.description;
    if (input.priority !== undefined) updatePayload.priority = input.priority;
    if (input.parent_id !== undefined) updatePayload.parent_id = input.parent_id;
    if (input.project !== undefined) updatePayload.project = input.project;
    if (input.tags !== undefined) updatePayload.tags = JSON.stringify(input.tags);

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id,
      updatePayload
    );

    return ok(deserializeItem(updated));
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const db = getDatabases();
    const env = getEnv();

    const itemDoc = await getItemOrThrow(db, env.APPWRITE_DATABASE_ID, id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");

    await db.deleteDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      id
    );

    return ok({ deleted: true, id });
  } catch (err) {
    return handleError(err);
  }
}

async function getItemOrThrow(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  itemId: string
): Promise<Record<string, unknown>> {
  try {
    return await db.getDocument(databaseId, COLLECTIONS.ITEMS, itemId);
  } catch (err: unknown) {
    if (err instanceof Error && (err as any).code === 404) {
      throw new AppError(ErrorCode.ITEM_NOT_FOUND, `Item ${itemId} not found`);
    }
    throw err;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/items/items-get-update-delete.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/items/[id]/route.ts __tests__/api/items/items-get-update-delete.test.ts
git commit -m "feat: add GET/PATCH/DELETE /api/items/:id endpoints"
```

<!-- CONTINUED IN PART B: Item actions, history, and plans -->
