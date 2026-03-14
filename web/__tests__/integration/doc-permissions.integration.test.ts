// @vitest-environment node

import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import {
  Client,
  Databases,
  Users,
  ID,
  Permission,
  Role,
  Query,
} from "node-appwrite";

const ENDPOINT = "https://cloud.appwrite.io/v1";
const PROJECT_ID = "69b2c740003cbab398cc";
const API_KEY =
  "standard_037d12fa61a503bf8df49d8aa19ee526ee1e6f7313bdb5dfe04cbe9558095c2ccaefcf433b7316f63e996bedcef8a921d4465a62398320a07276e39dfcf5dac9c375bd655cbff66b45c4d80daffd646d05580d30f9116a83932984f553b171409ec049383b456340e0cdcfd1a295c84dff92793eefd52c21a9b806dde52595eb";
const DATABASE_ID = "obeya";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Admin client — bypasses document-level permissions via API key. */
function adminClient(): { db: Databases; users: Users } {
  const client = new Client()
    .setEndpoint(ENDPOINT)
    .setProject(PROJECT_ID)
    .setKey(API_KEY);
  return { db: new Databases(client), users: new Users(client) };
}

/**
 * Create a user-scoped client that respects document permissions.
 * Uses the admin SDK to mint a JWT for the given user, then sets it
 * on a fresh Client via `setJWT`. This avoids session/cookie issues
 * that arise when using the server SDK with `setSession`.
 */
async function userClientForId(userId: string): Promise<Databases> {
  const { users } = adminClient();
  const { jwt } = await users.createJWT(userId);
  const client = new Client()
    .setEndpoint(ENDPOINT)
    .setProject(PROJECT_ID)
    .setJWT(jwt);
  return new Databases(client);
}

/** Minimal valid payload for a board document. */
function boardPayload(overrides: Record<string, unknown> = {}) {
  const now = new Date().toISOString();
  return {
    name: "test-board",
    owner_id: USER_A_ID,
    org_id: null,
    display_counter: 0,
    columns: "[]",
    display_map: "{}",
    users: "{}",
    projects: "{}",
    agent_role: "worker",
    version: 1,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

/** Minimal valid payload for an item document. */
function itemPayload(
  boardId: string,
  displayNum: number,
  overrides: Record<string, unknown> = {},
) {
  const now = new Date().toISOString();
  return {
    board_id: boardId,
    display_num: displayNum,
    type: "task",
    title: `Test item ${displayNum}`,
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
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const USER_A_ID = "testuser001";

const USER_B_EMAIL = "testb@obeya.dev";
const USER_B_PASS = "TestPasswordB123!";

// ---------------------------------------------------------------------------
// Cleanup tracker
// ---------------------------------------------------------------------------

const cleanup: { collection: string; id: string }[] = [];

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("Document Permissions Integration", () => {
  let userBId: string;

  beforeAll(async () => {
    const { users } = adminClient();
    try {
      const user = await users.create(
        ID.unique(),
        USER_B_EMAIL,
        undefined,
        USER_B_PASS,
        "Test User B",
      );
      userBId = user.$id;
    } catch (err: unknown) {
      const appwriteErr = err as { code?: number };
      if (appwriteErr.code === 409) {
        const list = await users.list([Query.equal("email", [USER_B_EMAIL])]);
        userBId = list.users[0].$id;
      } else {
        throw err;
      }
    }
  }, 30_000);

  afterAll(async () => {
    const { users } = adminClient();
    try {
      await users.delete(userBId);
    } catch {
      /* best-effort */
    }
  }, 30_000);

  afterEach(async () => {
    const { db } = adminClient();
    for (const item of cleanup.reverse()) {
      try {
        await db.deleteDocument(DATABASE_ID, item.collection, item.id);
      } catch {
        /* already deleted or never created */
      }
    }
    cleanup.length = 0;
  }, 30_000);

  // -----------------------------------------------------------------------
  // 1. Permissions are stored correctly on creation
  // -----------------------------------------------------------------------
  it("board created with owner permissions has correct $permissions", async () => {
    const { db } = adminClient();
    const perms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
      Permission.delete(Role.user(USER_A_ID)),
    ];

    const doc = await db.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-perms-board" }),
      perms,
    );
    cleanup.push({ collection: "boards", id: doc.$id });

    expect(doc.$permissions).toContain(`read("user:${USER_A_ID}")`);
    expect(doc.$permissions).toContain(`update("user:${USER_A_ID}")`);
    expect(doc.$permissions).toContain(`delete("user:${USER_A_ID}")`);
  }, 15_000);

  // -----------------------------------------------------------------------
  // 2. Owner can read their own board via user JWT
  // -----------------------------------------------------------------------
  it("owner can read their board via user JWT", async () => {
    const { db: adminDb } = adminClient();
    const perms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
    ];

    const doc = await adminDb.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-user-read" }),
      perms,
    );
    cleanup.push({ collection: "boards", id: doc.$id });

    const userDb = await userClientForId(USER_A_ID);
    const fetched = await userDb.getDocument(DATABASE_ID, "boards", doc.$id);
    expect(fetched.name).toBe("test-user-read");
  }, 15_000);

  // -----------------------------------------------------------------------
  // 3. User without permission CANNOT read another user's board
  // -----------------------------------------------------------------------
  it("user without permission CANNOT read another user's board", async () => {
    const { db: adminDb } = adminClient();
    const perms = [Permission.read(Role.user(USER_A_ID))];

    const doc = await adminDb.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-no-access" }),
      perms,
    );
    cleanup.push({ collection: "boards", id: doc.$id });

    const userBDb = await userClientForId(userBId);
    await expect(
      userBDb.getDocument(DATABASE_ID, "boards", doc.$id),
    ).rejects.toThrow();
  }, 15_000);

  // -----------------------------------------------------------------------
  // 4. Adding read permission allows previously unauthorized user to read
  // -----------------------------------------------------------------------
  it("adding read permission allows previously unauthorized user to read", async () => {
    const { db: adminDb } = adminClient();
    const perms = [Permission.read(Role.user(USER_A_ID))];

    const doc = await adminDb.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-add-perm" }),
      perms,
    );
    cleanup.push({ collection: "boards", id: doc.$id });

    // User B cannot read yet
    const userBDb = await userClientForId(userBId);
    await expect(
      userBDb.getDocument(DATABASE_ID, "boards", doc.$id),
    ).rejects.toThrow();

    // Grant read to User B via admin
    const updatedPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.read(Role.user(userBId)),
    ];
    await adminDb.updateDocument(
      DATABASE_ID,
      "boards",
      doc.$id,
      {},
      updatedPerms,
    );

    // User B CAN now read (fresh JWT to avoid any caching)
    const userBDb2 = await userClientForId(userBId);
    const fetched = await userBDb2.getDocument(
      DATABASE_ID,
      "boards",
      doc.$id,
    );
    expect(fetched.name).toBe("test-add-perm");
  }, 30_000);

  // -----------------------------------------------------------------------
  // 5. Full sync flow: board + items permission propagation
  // -----------------------------------------------------------------------
  it("item permissions sync: all items get updated when board permissions change", async () => {
    const { db: adminDb } = adminClient();
    const boardPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
    ];

    const board = await adminDb.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-sync", display_counter: 2 }),
      boardPerms,
    );
    cleanup.push({ collection: "boards", id: board.$id });

    // Create 2 items owned only by User A
    const itemPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
    ];
    for (let i = 1; i <= 2; i++) {
      const item = await adminDb.createDocument(
        DATABASE_ID,
        "items",
        ID.unique(),
        itemPayload(board.$id, i),
        itemPerms,
      );
      cleanup.push({ collection: "items", id: item.$id });
    }

    // User B cannot see items yet
    const userBDb = await userClientForId(userBId);
    const beforeList = await userBDb.listDocuments(DATABASE_ID, "items", [
      Query.equal("board_id", board.$id),
    ]);
    expect(beforeList.total).toBe(0);

    // Sync: grant User B read on board + every item
    const newPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
      Permission.read(Role.user(userBId)),
    ];
    await adminDb.updateDocument(
      DATABASE_ID,
      "boards",
      board.$id,
      {},
      newPerms,
    );

    const items = await adminDb.listDocuments(DATABASE_ID, "items", [
      Query.equal("board_id", board.$id),
      Query.limit(100),
    ]);
    for (const item of items.documents) {
      await adminDb.updateDocument(
        DATABASE_ID,
        "items",
        item.$id,
        {},
        newPerms,
      );
    }

    // User B CAN now see all items
    const userBDb2 = await userClientForId(userBId);
    const afterList = await userBDb2.listDocuments(DATABASE_ID, "items", [
      Query.equal("board_id", board.$id),
    ]);
    expect(afterList.total).toBe(2);
  }, 30_000);

  // -----------------------------------------------------------------------
  // 6. Editor can update items, viewer cannot
  // -----------------------------------------------------------------------
  it("editor can update items, viewer cannot", async () => {
    const { db: adminDb } = adminClient();

    // Board readable by both users
    const boardPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.read(Role.user(userBId)),
    ];
    const board = await adminDb.createDocument(
      DATABASE_ID,
      "boards",
      ID.unique(),
      boardPayload({ name: "test-editor-viewer", display_counter: 1 }),
      boardPerms,
    );
    cleanup.push({ collection: "boards", id: board.$id });

    // Item: User A can read+update, User B can only read
    const itemPerms = [
      Permission.read(Role.user(USER_A_ID)),
      Permission.update(Role.user(USER_A_ID)),
      Permission.read(Role.user(userBId)),
    ];
    const item = await adminDb.createDocument(
      DATABASE_ID,
      "items",
      ID.unique(),
      itemPayload(board.$id, 1, { title: "Viewer cant edit" }),
      itemPerms,
    );
    cleanup.push({ collection: "items", id: item.$id });

    // User A (editor) CAN update
    const userADb = await userClientForId(USER_A_ID);
    const updated = await userADb.updateDocument(
      DATABASE_ID,
      "items",
      item.$id,
      { title: "Updated by owner" },
    );
    expect(updated.title).toBe("Updated by owner");

    // User B (viewer) CANNOT update
    const userBDb = await userClientForId(userBId);
    await expect(
      userBDb.updateDocument(DATABASE_ID, "items", item.$id, {
        title: "Should fail",
      }),
    ).rejects.toThrow();
  }, 30_000);
});
