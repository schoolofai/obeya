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
