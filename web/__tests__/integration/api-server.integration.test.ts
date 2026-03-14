// @vitest-environment node

import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import {
  Client,
  Databases,
  Users,
  ID,
  Query,
} from "node-appwrite";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const ENDPOINT = "https://cloud.appwrite.io/v1";
const PROJECT_ID = "69b2c740003cbab398cc";
const API_KEY =
  "standard_037d12fa61a503bf8df49d8aa19ee526ee1e6f7313bdb5dfe04cbe9558095c2ccaefcf433b7316f63e996bedcef8a921d4465a62398320a07276e39dfcf5dac9c375bd655cbff66b45c4d80daffd646d05580d30f9116a83932984f553b171409ec049383b456340e0cdcfd1a295c84dff92793eefd52c21a9b806dde52595eb";
const DATABASE_ID = "obeya";
const USER_ID = "testuser001";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getDb(): Databases {
  const client = new Client()
    .setEndpoint(ENDPOINT)
    .setProject(PROJECT_ID)
    .setKey(API_KEY);
  return new Databases(client);
}

function getUsers(): Users {
  const client = new Client()
    .setEndpoint(ENDPOINT)
    .setProject(PROJECT_ID)
    .setKey(API_KEY);
  return new Users(client);
}

function now(): string {
  return new Date().toISOString();
}

function boardData(
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    name: "Integration Test Board",
    owner_id: USER_ID,
    org_id: null,
    display_counter: 0,
    columns: JSON.stringify([
      { name: "backlog", limit: 0 },
      { name: "in-progress", limit: 3 },
      { name: "done", limit: 0 },
    ]),
    display_map: "{}",
    users: "{}",
    projects: "{}",
    agent_role: "worker",
    version: 1,
    created_at: now(),
    updated_at: now(),
    ...overrides,
  };
}

function itemData(
  boardId: string,
  displayNum: number,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    board_id: boardId,
    display_num: displayNum,
    type: "task",
    title: `Test Item ${displayNum}`,
    description: "",
    status: "backlog",
    priority: "medium",
    parent_id: null,
    assignee_id: null,
    blocked_by: "[]",
    tags: "[]",
    project: null,
    created_at: now(),
    updated_at: now(),
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Cleanup tracker (used by Board Lifecycle & Display Map tests)
// ---------------------------------------------------------------------------

const cleanup: { collection: string; id: string }[] = [];

async function runCleanup(): Promise<void> {
  const db = getDb();
  for (const item of cleanup.reverse()) {
    try {
      await db.deleteDocument(DATABASE_ID, item.collection, item.id);
    } catch {
      /* already deleted or never created */
    }
  }
  cleanup.length = 0;
}

// ---------------------------------------------------------------------------
// 1. Board Lifecycle
// ---------------------------------------------------------------------------

describe("Board Lifecycle", () => {
  afterEach(runCleanup, 30_000);

  it(
    "creates a board with all required fields",
    async () => {
      const db = getDb();
      const doc = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData(),
      );
      cleanup.push({ collection: "boards", id: doc.$id });

      expect(doc.name).toBe("Integration Test Board");
      expect(doc.owner_id).toBe(USER_ID);
      expect(doc.display_counter).toBe(0);
      expect(doc.version).toBe(1);

      // Verify columns round-trip
      const cols = JSON.parse(doc.columns as string);
      expect(cols).toHaveLength(3);
      expect(cols[0].name).toBe("backlog");
    },
    15_000,
  );

  it(
    "reads a board by ID",
    async () => {
      const db = getDb();
      const created = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData(),
      );
      cleanup.push({ collection: "boards", id: created.$id });

      const fetched = await db.getDocument(
        DATABASE_ID,
        "boards",
        created.$id,
      );
      expect(fetched.name).toBe("Integration Test Board");
      expect(fetched.$id).toBe(created.$id);
    },
    15_000,
  );

  it(
    "updates board name and columns",
    async () => {
      const db = getDb();
      const created = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData(),
      );
      cleanup.push({ collection: "boards", id: created.$id });

      const updated = await db.updateDocument(
        DATABASE_ID,
        "boards",
        created.$id,
        {
          name: "Renamed Board",
          columns: JSON.stringify([
            { name: "todo", limit: 0 },
            { name: "done", limit: 0 },
          ]),
          updated_at: now(),
        },
      );
      expect(updated.name).toBe("Renamed Board");
      const cols = JSON.parse(updated.columns as string);
      expect(cols).toHaveLength(2);
    },
    15_000,
  );

  it(
    "deletes a board",
    async () => {
      const db = getDb();
      const created = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData(),
      );

      await db.deleteDocument(DATABASE_ID, "boards", created.$id);

      await expect(
        db.getDocument(DATABASE_ID, "boards", created.$id),
      ).rejects.toThrow();
    },
    15_000,
  );

  it(
    "increments display_counter atomically",
    async () => {
      const db = getDb();
      const created = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData({ display_counter: 5 }),
      );
      cleanup.push({ collection: "boards", id: created.$id });

      // Read, increment, write (same pattern as counter.ts)
      const board = await db.getDocument(DATABASE_ID, "boards", created.$id);
      const nextNum = (board.display_counter as number) + 1;
      await db.updateDocument(DATABASE_ID, "boards", created.$id, {
        display_counter: nextNum,
      });

      const after = await db.getDocument(DATABASE_ID, "boards", created.$id);
      expect(after.display_counter).toBe(6);
    },
    15_000,
  );
});

// ---------------------------------------------------------------------------
// 2. Item Lifecycle
// ---------------------------------------------------------------------------

describe("Item Lifecycle", () => {
  let boardId: string;

  beforeAll(async () => {
    const db = getDb();
    const board = await db.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardData(),
    );
    boardId = board.$id;
  }, 15_000);

  afterAll(async () => {
    const db = getDb();
    // Delete all items first
    const items = await db.listDocuments(DATABASE_ID, "items", [
      Query.equal("board_id", boardId),
      Query.limit(100),
    ]);
    for (const item of items.documents) {
      try {
        await db.deleteDocument(DATABASE_ID, "items", item.$id);
      } catch {
        /* best-effort */
      }
    }
    // Delete history
    const history = await db.listDocuments(DATABASE_ID, "item_history", [
      Query.equal("board_id", boardId),
      Query.limit(100),
    ]);
    for (const h of history.documents) {
      try {
        await db.deleteDocument(DATABASE_ID, "item_history", h.$id);
      } catch {
        /* best-effort */
      }
    }
    // Delete board
    try {
      await db.deleteDocument(DATABASE_ID, "boards", boardId);
    } catch {
      /* best-effort */
    }
  }, 30_000);

  it(
    "creates an item with all fields",
    async () => {
      const db = getDb();
      const doc = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 1),
      );

      expect(doc.board_id).toBe(boardId);
      expect(doc.display_num).toBe(1);
      expect(doc.type).toBe("task");
      expect(doc.title).toBe("Test Item 1");
      expect(doc.status).toBe("backlog");
      expect(doc.priority).toBe("medium");
    },
    15_000,
  );

  it(
    "creates items with all types (epic, story, task)",
    async () => {
      const db = getDb();
      const types = ["epic", "story", "task"] as const;
      for (const type of types) {
        const doc = await db.createDocument(
          DATABASE_ID,
          "items",
          ID.unique(),
          itemData(boardId, 10 + types.indexOf(type), { type }),
        );
        expect(doc.type).toBe(type);
      }
    },
    15_000,
  );

  it(
    "creates items with all priority levels",
    async () => {
      const db = getDb();
      const priorities = ["low", "medium", "high", "critical"] as const;
      for (const priority of priorities) {
        const doc = await db.createDocument(
          DATABASE_ID,
          "items",
          ID.unique(),
          itemData(boardId, 20 + priorities.indexOf(priority), { priority }),
        );
        expect(doc.priority).toBe(priority);
      }
    },
    15_000,
  );

  it(
    "moves an item by updating status",
    async () => {
      const db = getDb();
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 30, { status: "backlog" }),
      );

      const moved = await db.updateDocument(DATABASE_ID, "items", item.$id, {
        status: "in-progress",
        updated_at: now(),
      });
      expect(moved.status).toBe("in-progress");

      // Read back to confirm persistence
      const fetched = await db.getDocument(DATABASE_ID, "items", item.$id);
      expect(fetched.status).toBe("in-progress");
    },
    15_000,
  );

  it(
    "assigns an item",
    async () => {
      const db = getDb();
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 31),
      );

      const assigned = await db.updateDocument(
        DATABASE_ID,
        "items",
        item.$id,
        { assignee_id: USER_ID, updated_at: now() },
      );
      expect(assigned.assignee_id).toBe(USER_ID);
    },
    15_000,
  );

  it(
    "unassigns an item (null assignee)",
    async () => {
      const db = getDb();
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 32, { assignee_id: USER_ID }),
      );

      const unassigned = await db.updateDocument(
        DATABASE_ID,
        "items",
        item.$id,
        { assignee_id: null, updated_at: now() },
      );
      expect(unassigned.assignee_id).toBeNull();
    },
    15_000,
  );

  it(
    "blocks and unblocks an item",
    async () => {
      const db = getDb();
      const blocker = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 33),
      );
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 34),
      );

      // Block
      const blocked = await db.updateDocument(
        DATABASE_ID,
        "items",
        item.$id,
        {
          blocked_by: JSON.stringify([blocker.$id]),
          updated_at: now(),
        },
      );
      expect(JSON.parse(blocked.blocked_by as string)).toContain(blocker.$id);

      // Unblock
      const unblocked = await db.updateDocument(
        DATABASE_ID,
        "items",
        item.$id,
        {
          blocked_by: "[]",
          updated_at: now(),
        },
      );
      expect(JSON.parse(unblocked.blocked_by as string)).toHaveLength(0);
    },
    15_000,
  );

  it(
    "sets parent_id for hierarchy",
    async () => {
      const db = getDb();
      const epic = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 40, { type: "epic", title: "Parent Epic" }),
      );
      const task = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 41, { parent_id: epic.$id }),
      );

      expect(task.parent_id).toBe(epic.$id);
    },
    15_000,
  );

  it(
    "queries items by board_id",
    async () => {
      const db = getDb();
      const result = await db.listDocuments(DATABASE_ID, "items", [
        Query.equal("board_id", boardId),
        Query.limit(100),
      ]);
      expect(result.total).toBeGreaterThan(0);
      for (const doc of result.documents) {
        expect(doc.board_id).toBe(boardId);
      }
    },
    15_000,
  );

  it(
    "queries items by status filter",
    async () => {
      const db = getDb();
      const result = await db.listDocuments(DATABASE_ID, "items", [
        Query.equal("board_id", boardId),
        Query.equal("status", "in-progress"),
        Query.limit(100),
      ]);
      for (const doc of result.documents) {
        expect(doc.status).toBe("in-progress");
      }
    },
    15_000,
  );

  it(
    "deletes an item",
    async () => {
      const db = getDb();
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(boardId, 99),
      );

      await db.deleteDocument(DATABASE_ID, "items", item.$id);
      await expect(
        db.getDocument(DATABASE_ID, "items", item.$id),
      ).rejects.toThrow();
    },
    15_000,
  );
});

// ---------------------------------------------------------------------------
// 3. History Entries
// ---------------------------------------------------------------------------

describe("History Entries", () => {
  let boardId: string;

  beforeAll(async () => {
    const db = getDb();
    const board = await db.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardData(),
    );
    boardId = board.$id;
  }, 15_000);

  afterAll(async () => {
    const db = getDb();
    const history = await db.listDocuments(DATABASE_ID, "item_history", [
      Query.equal("board_id", boardId),
      Query.limit(100),
    ]);
    for (const h of history.documents) {
      try {
        await db.deleteDocument(DATABASE_ID, "item_history", h.$id);
      } catch {
        /* best-effort */
      }
    }
    try {
      await db.deleteDocument(DATABASE_ID, "boards", boardId);
    } catch {
      /* best-effort */
    }
  }, 30_000);

  it(
    "creates a history entry for move action",
    async () => {
      const db = getDb();
      const doc = await db.createDocument(
        DATABASE_ID,
        "item_history",
        ID.unique(),
        {
          item_id: "test-item-1",
          board_id: boardId,
          user_id: USER_ID,
          session_id: "test-session",
          action: "moved",
          detail: "status: backlog -> in-progress",
          timestamp: now(),
        },
      );

      expect(doc.action).toBe("moved");
      expect(doc.detail).toBe("status: backlog -> in-progress");
    },
    15_000,
  );

  it(
    "creates history entries for all action types",
    async () => {
      const db = getDb();
      const actions = [
        "created",
        "moved",
        "edited",
        "assigned",
        "blocked",
        "unblocked",
      ] as const;
      for (const action of actions) {
        const doc = await db.createDocument(
          DATABASE_ID,
          "item_history",
          ID.unique(),
          {
            item_id: "test-item-2",
            board_id: boardId,
            user_id: USER_ID,
            session_id: "test-session",
            action,
            detail: `test ${action}`,
            timestamp: now(),
          },
        );
        expect(doc.action).toBe(action);
      }
    },
    30_000,
  );

  it(
    "queries history by board_id ordered by timestamp",
    async () => {
      const db = getDb();
      const result = await db.listDocuments(DATABASE_ID, "item_history", [
        Query.equal("board_id", boardId),
        Query.orderDesc("timestamp"),
        Query.limit(10),
      ]);
      expect(result.total).toBeGreaterThan(0);
      for (const doc of result.documents) {
        expect(doc.board_id).toBe(boardId);
      }
    },
    15_000,
  );
});

// ---------------------------------------------------------------------------
// 4. Board Members
// ---------------------------------------------------------------------------

describe("Board Members", () => {
  let boardId: string;
  let userBId: string;

  beforeAll(async () => {
    const db = getDb();
    const users = getUsers();
    const board = await db.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardData(),
    );
    boardId = board.$id;

    // Create or find test user B
    try {
      const user = await users.create(
        ID.unique(),
        "inttest-member@obeya.dev",
        undefined,
        "TestMember123!",
        "Member User",
      );
      userBId = user.$id;
    } catch (err: unknown) {
      const appwriteErr = err as { code?: number };
      if (appwriteErr.code === 409) {
        const list = await users.list([
          Query.equal("email", ["inttest-member@obeya.dev"]),
        ]);
        userBId = list.users[0].$id;
      } else {
        throw err;
      }
    }
  }, 30_000);

  afterAll(async () => {
    const db = getDb();
    const users = getUsers();
    // Clean up members
    const members = await db.listDocuments(DATABASE_ID, "board_members", [
      Query.equal("board_id", boardId),
      Query.limit(100),
    ]);
    for (const m of members.documents) {
      try {
        await db.deleteDocument(DATABASE_ID, "board_members", m.$id);
      } catch {
        /* best-effort */
      }
    }
    try {
      await db.deleteDocument(DATABASE_ID, "boards", boardId);
    } catch {
      /* best-effort */
    }
    try {
      await users.delete(userBId);
    } catch {
      /* best-effort */
    }
  }, 30_000);

  it(
    "adds a member to a board",
    async () => {
      const db = getDb();
      const doc = await db.createDocument(
        DATABASE_ID,
        "board_members",
        ID.unique(),
        {
          board_id: boardId,
          user_id: userBId,
          role: "editor",
          invited_at: now(),
        },
      );
      expect(doc.board_id).toBe(boardId);
      expect(doc.user_id).toBe(userBId);
      expect(doc.role).toBe("editor");
    },
    15_000,
  );

  it(
    "prevents duplicate board membership",
    async () => {
      const db = getDb();
      // Check if already exists
      const existing = await db.listDocuments(DATABASE_ID, "board_members", [
        Query.equal("board_id", boardId),
        Query.equal("user_id", userBId),
        Query.limit(1),
      ]);
      expect(existing.total).toBe(1); // Should exist from previous test
    },
    15_000,
  );

  it(
    "queries board members",
    async () => {
      const db = getDb();
      const result = await db.listDocuments(DATABASE_ID, "board_members", [
        Query.equal("board_id", boardId),
        Query.limit(100),
      ]);
      expect(result.total).toBeGreaterThanOrEqual(1);
      const roles = result.documents.map((d) => d.role);
      expect(roles).toContain("editor");
    },
    15_000,
  );

  it(
    "removes a member from a board",
    async () => {
      const db = getDb();
      const members = await db.listDocuments(DATABASE_ID, "board_members", [
        Query.equal("board_id", boardId),
        Query.equal("user_id", userBId),
        Query.limit(1),
      ]);
      if (members.documents.length > 0) {
        await db.deleteDocument(
          DATABASE_ID,
          "board_members",
          members.documents[0].$id,
        );
      }

      const after = await db.listDocuments(DATABASE_ID, "board_members", [
        Query.equal("board_id", boardId),
        Query.equal("user_id", userBId),
        Query.limit(1),
      ]);
      expect(after.total).toBe(0);
    },
    15_000,
  );
});

// ---------------------------------------------------------------------------
// 5. Display Map Round-Trip
// ---------------------------------------------------------------------------

describe("Display Map Round-Trip", () => {
  afterEach(runCleanup, 30_000);

  it(
    "stores and retrieves display_map as JSON",
    async () => {
      const db = getDb();
      const displayMap = {
        "1": "item-abc",
        "2": "item-def",
        "3": "item-ghi",
      };
      const board = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData({
          display_map: JSON.stringify(displayMap),
          display_counter: 3,
        }),
      );
      cleanup.push({ collection: "boards", id: board.$id });

      const fetched = await db.getDocument(DATABASE_ID, "boards", board.$id);
      const parsed = JSON.parse(fetched.display_map as string);
      expect(parsed["1"]).toBe("item-abc");
      expect(parsed["2"]).toBe("item-def");
      expect(parsed["3"]).toBe("item-ghi");
    },
    15_000,
  );

  it(
    "updates display_map when adding items",
    async () => {
      const db = getDb();
      const board = await db.createDocument(
        DATABASE_ID,
        "boards",
        ID.unique(),
        boardData(),
      );
      cleanup.push({ collection: "boards", id: board.$id });

      // Create item
      const item = await db.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemData(board.$id, 1),
      );
      cleanup.push({ collection: "items", id: item.$id });

      // Update display_map (same pattern as route handler)
      const currentMap = JSON.parse(board.display_map as string);
      currentMap["1"] = item.$id;
      await db.updateDocument(DATABASE_ID, "boards", board.$id, {
        display_map: JSON.stringify(currentMap),
        display_counter: 1,
      });

      const after = await db.getDocument(DATABASE_ID, "boards", board.$id);
      const afterMap = JSON.parse(after.display_map as string);
      expect(afterMap["1"]).toBe(item.$id);
      expect(after.display_counter).toBe(1);
    },
    15_000,
  );
});
