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
  FREE_TIER_LIMITS,
  enforcePersonalBoardLimit,
  enforceOrgLimit,
  enforceOrgMemberLimit,
  enforceBoardItemLimit,
} from "@/lib/limits";
import { getDatabases } from "@/lib/appwrite/server";
import { AppError, ErrorCode } from "@/lib/errors";

describe("FREE_TIER_LIMITS", () => {
  it("defines correct personal board limit", () => {
    expect(FREE_TIER_LIMITS.PERSONAL_BOARDS).toBe(3);
  });

  it("defines correct org limit", () => {
    expect(FREE_TIER_LIMITS.ORGS).toBe(1);
  });

  it("defines correct members per org limit", () => {
    expect(FREE_TIER_LIMITS.MEMBERS_PER_ORG).toBe(3);
  });

  it("defines correct items per board limit", () => {
    expect(FREE_TIER_LIMITS.ITEMS_PER_BOARD).toBe(100);
  });
});

describe("enforcePersonalBoardLimit", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when user is under the limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 2, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforcePersonalBoardLimit("user-1")).resolves.not.toThrow();
  });

  it("throws PLAN_LIMIT_REACHED when at the limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 3, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforcePersonalBoardLimit("user-1")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });

  it("throws PLAN_LIMIT_REACHED when over the limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 5, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforcePersonalBoardLimit("user-1")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });
});

describe("enforceOrgLimit", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when user has no orgs", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgLimit("user-1")).resolves.not.toThrow();
  });

  it("throws PLAN_LIMIT_REACHED when at the org limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 1, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgLimit("user-1")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });
});

describe("enforceOrgMemberLimit", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when org has fewer members than limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 2, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgMemberLimit("org-1", "free")).resolves.not.toThrow();
  });

  it("throws PLAN_LIMIT_REACHED when at the member limit for free plan", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 3, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgMemberLimit("org-1", "free")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });

  it("does not throw for pro plan regardless of member count", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 100, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgMemberLimit("org-1", "pro")).resolves.not.toThrow();
  });

  it("does not throw for enterprise plan regardless of member count", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 500, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceOrgMemberLimit("org-1", "enterprise")).resolves.not.toThrow();
  });
});

describe("enforceBoardItemLimit", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not throw when board is under the item limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 50, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceBoardItemLimit("board-1")).resolves.not.toThrow();
  });

  it("throws PLAN_LIMIT_REACHED when board is at the item limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 100, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceBoardItemLimit("board-1")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });

  it("throws PLAN_LIMIT_REACHED when board is over the item limit", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 150, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(enforceBoardItemLimit("board-1")).rejects.toMatchObject({
      code: ErrorCode.PLAN_LIMIT_REACHED,
    });
  });
});
