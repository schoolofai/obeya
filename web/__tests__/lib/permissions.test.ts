import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import {
  ORG_ROLE_LEVEL,
  BOARD_ROLE_LEVEL,
  resolvePermission,
  requireOrgRole,
  requireBoardAccess,
} from "@/lib/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { AppError, ErrorCode } from "@/lib/errors";

describe("ORG_ROLE_LEVEL and BOARD_ROLE_LEVEL", () => {
  it("defines org role levels correctly", () => {
    expect(ORG_ROLE_LEVEL.member).toBe(1);
    expect(ORG_ROLE_LEVEL.admin).toBe(2);
    expect(ORG_ROLE_LEVEL.owner).toBe(3);
  });

  it("defines board role levels correctly", () => {
    expect(BOARD_ROLE_LEVEL.viewer).toBe(1);
    expect(BOARD_ROLE_LEVEL.editor).toBe(2);
    expect(BOARD_ROLE_LEVEL.owner).toBe(3);
  });
});

describe("resolvePermission", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns org member level when org membership exists and no board membership", async () => {
    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({
          documents: [{ user_id: "user-1", org_id: "org-1", role: "admin" }],
        })
        .mockResolvedValueOnce({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await resolvePermission("user-1", "board-1", "org-1");
    expect(result.level).toBe(BOARD_ROLE_LEVEL.editor);
    expect(result.source).toBe("org");
  });

  it("returns board member level when board membership exists and no org", async () => {
    // When orgId is null, only board members are queried (one listDocuments call)
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", board_id: "board-1", role: "editor" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await resolvePermission("user-1", "board-1", null);
    expect(result.level).toBe(BOARD_ROLE_LEVEL.editor);
    expect(result.source).toBe("board");
  });

  it("returns no access when no memberships", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await resolvePermission("user-1", "board-1", null);
    expect(result.level).toBe(0);
    expect(result.source).toBe("none");
  });

  it("returns higher permission when both org and board memberships exist", async () => {
    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({
          documents: [{ user_id: "user-1", org_id: "org-1", role: "member" }],
        })
        .mockResolvedValueOnce({
          documents: [{ user_id: "user-1", board_id: "board-1", role: "owner" }],
        }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await resolvePermission("user-1", "board-1", "org-1");
    expect(result.level).toBe(BOARD_ROLE_LEVEL.owner);
    expect(result.source).toBe("board");
  });
});

describe("requireOrgRole", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when user has sufficient org role", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", org_id: "org-1", role: "admin" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(requireOrgRole("user-1", "org-1", "admin")).resolves.not.toThrow();
  });

  it("does not throw when user has higher org role", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", org_id: "org-1", role: "owner" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(requireOrgRole("user-1", "org-1", "member")).resolves.not.toThrow();
  });

  it("throws FORBIDDEN when user is not a member", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(requireOrgRole("user-1", "org-1", "member")).rejects.toThrow(AppError);
    await expect(requireOrgRole("user-1", "org-1", "member")).rejects.toMatchObject({
      code: ErrorCode.FORBIDDEN,
    });
  });

  it("throws FORBIDDEN when user has insufficient role", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", org_id: "org-1", role: "member" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(requireOrgRole("user-1", "org-1", "admin")).rejects.toMatchObject({
      code: ErrorCode.FORBIDDEN,
    });
  });
});

describe("requireBoardAccess", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when user has board access", async () => {
    // orgId is null, so only board membership is queried
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", board_id: "board-1", role: "editor" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(
      requireBoardAccess("user-1", "board-1", null, "viewer")
    ).resolves.not.toThrow();
  });

  it("throws FORBIDDEN when user has no access", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(
      requireBoardAccess("user-1", "board-1", null, "viewer")
    ).rejects.toMatchObject({ code: ErrorCode.FORBIDDEN });
  });

  it("throws FORBIDDEN when user has insufficient level", async () => {
    // orgId is null, so only board membership is queried
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ user_id: "user-1", board_id: "board-1", role: "viewer" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(
      requireBoardAccess("user-1", "board-1", null, "owner")
    ).rejects.toMatchObject({ code: ErrorCode.FORBIDDEN });
  });
});
