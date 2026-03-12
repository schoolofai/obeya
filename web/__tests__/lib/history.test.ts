import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
    APPWRITE_ENDPOINT: "https://cloud.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "proj-1",
    APPWRITE_API_KEY: "key-1",
  }),
}));
vi.mock("node-appwrite", () => ({
  ID: { unique: vi.fn().mockReturnValue("hist-id-1") },
}));

import { createHistoryEntry } from "@/lib/history";
import { getDatabases } from "@/lib/appwrite/server";

describe("createHistoryEntry", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("calls createDocument with correct fields", async () => {
    const mockDb = { createDocument: vi.fn().mockResolvedValue({}) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await createHistoryEntry({
      itemId: "item-1",
      boardId: "board-1",
      userId: "user-1",
      action: "moved",
      detail: "status: todo -> in-progress",
    });

    expect(mockDb.createDocument).toHaveBeenCalledOnce();
    const [dbId, collection, docId, payload] = mockDb.createDocument.mock.calls[0];
    expect(dbId).toBe("obeya");
    expect(collection).toBe("item_history");
    expect(docId).toBe("hist-id-1");
    expect(payload.item_id).toBe("item-1");
    expect(payload.board_id).toBe("board-1");
    expect(payload.user_id).toBe("user-1");
    expect(payload.action).toBe("moved");
    expect(payload.detail).toBe("status: todo -> in-progress");
  });

  it("includes an ISO timestamp in the payload", async () => {
    const mockDb = { createDocument: vi.fn().mockResolvedValue({}) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const before = new Date().toISOString();
    await createHistoryEntry({
      itemId: "item-2",
      boardId: "board-2",
      userId: "user-2",
      action: "created",
      detail: "item created",
    });
    const after = new Date().toISOString();

    const payload = mockDb.createDocument.mock.calls[0][3];
    expect(payload.timestamp).toBeDefined();
    expect(payload.timestamp >= before).toBe(true);
    expect(payload.timestamp <= after).toBe(true);
  });
});
