import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }),
}));

import { syncBoardPermissions } from "@/lib/appwrite/sync-permissions";
import { getDatabases } from "@/lib/appwrite/server";

describe("syncBoardPermissions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("updates board and all items with correct permissions", async () => {
    const mockDb = {
      getDocument: vi
        .fn()
        .mockResolvedValue({ $id: "board-1", owner_id: "owner-1" }),
      listDocuments: vi
        .fn()
        .mockResolvedValueOnce({
          // board_members query (from fetchBoardMemberList)
          total: 1,
          documents: [{ user_id: "editor-1", role: "editor" }],
        })
        .mockResolvedValueOnce({
          // items query
          total: 2,
          documents: [{ $id: "item-1" }, { $id: "item-2" }],
        }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const count = await syncBoardPermissions("board-1");

    expect(count).toBe(3); // board + 2 items
    expect(mockDb.updateDocument).toHaveBeenCalledTimes(3);

    // First call: board permissions
    const boardPermsCall = mockDb.updateDocument.mock.calls[0];
    expect(boardPermsCall[2]).toBe("board-1");
    const boardPerms = boardPermsCall[4];
    expect(boardPerms).toContain('read("user:owner-1")');
    expect(boardPerms).toContain('read("user:editor-1")');

    // Item calls: item permissions
    const itemPermsCall = mockDb.updateDocument.mock.calls[1];
    const itemPerms = itemPermsCall[4];
    expect(itemPerms).toContain('read("user:owner-1")');
    expect(itemPerms).toContain('read("user:editor-1")');
  });

  it("handles board with no members", async () => {
    const mockDb = {
      getDocument: vi
        .fn()
        .mockResolvedValue({ $id: "board-1", owner_id: "owner-1" }),
      listDocuments: vi
        .fn()
        .mockResolvedValueOnce({ total: 0, documents: [] }) // no members
        .mockResolvedValueOnce({ total: 0, documents: [] }), // no items
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const count = await syncBoardPermissions("board-1");

    expect(count).toBe(1); // just the board
    expect(mockDb.updateDocument).toHaveBeenCalledTimes(1);
  });
});
