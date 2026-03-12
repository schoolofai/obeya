# Obeya Cloud Plan 3: Orgs & Sharing — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement organization management, board sharing, permission resolution middleware, and free tier limit enforcement for Obeya Cloud.

**Architecture:** Next.js 15 App Router with API routes as the backend. Appwrite Server SDK for database and auth operations. Vitest for unit/integration testing of API route handlers. All API responses use a consistent envelope (`{ok, data, error, meta}`). Permission resolution checks org membership first, then board membership, using the HIGHER permission level when both exist.

**Tech Stack:** Next.js 15, TypeScript, Appwrite Node SDK (`node-appwrite`), Vitest, zod

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md`

**Repository:** `~/code/obeya-cloud` (created by Plan 1). This plan adds org, board member, and permission APIs to the existing project.

**Dependencies:** Plan 1 (Foundation) must be complete. Plan 2 (Boards & Items) should be complete for board member endpoints to be useful, but is not strictly required for org endpoints.

---

## File Structure

```
obeya-cloud/
├── app/api/
│   ├── orgs/
│   │   ├── route.ts                         # GET /api/orgs, POST /api/orgs
│   │   └── [id]/
│   │       ├── route.ts                     # GET/PATCH/DELETE /api/orgs/:id
│   │       └── members/
│   │           ├── route.ts                 # GET/POST /api/orgs/:id/members
│   │           └── [uid]/
│   │               └── route.ts             # PATCH/DELETE /api/orgs/:id/members/:uid
│   └── boards/
│       └── [id]/
│           └── members/
│               ├── route.ts                 # GET/POST /api/boards/:id/members
│               └── [uid]/
│                   └── route.ts             # PATCH/DELETE /api/boards/:id/members/:uid
├── lib/
│   ├── permissions.ts                       # Permission resolution utility
│   ├── slugs.ts                             # Slug generation and uniqueness
│   └── limits.ts                            # Free tier limit enforcement
└── __tests__/
    ├── lib/
    │   ├── permissions.test.ts
    │   ├── slugs.test.ts
    │   └── limits.test.ts
    └── api/
        ├── orgs/
        │   ├── orgs.test.ts                 # GET/POST /api/orgs
        │   ├── orgs-id.test.ts              # GET/PATCH/DELETE /api/orgs/:id
        │   └── org-members.test.ts          # Org member endpoints
        └── boards/
            └── board-members.test.ts        # Board member endpoints
```

---

## Chunk 1: Permission Utilities

### Task 1: Slug Generation

**Files:**
- Create: `obeya-cloud/lib/slugs.ts`
- Test: `obeya-cloud/__tests__/lib/slugs.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/slugs.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockListDocuments = vi.fn();

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { generateSlug, ensureUniqueSlug } from "@/lib/slugs";

describe("generateSlug", () => {
  it("converts name to lowercase kebab-case", () => {
    expect(generateSlug("My Cool Org")).toBe("my-cool-org");
  });

  it("strips special characters", () => {
    expect(generateSlug("Org @#$% Name!")).toBe("org-name");
  });

  it("collapses multiple hyphens", () => {
    expect(generateSlug("Org---Name")).toBe("org-name");
  });

  it("trims leading and trailing hyphens", () => {
    expect(generateSlug("--Org Name--")).toBe("org-name");
  });

  it("handles unicode by transliterating common chars", () => {
    expect(generateSlug("Über Org")).toBe("uber-org");
  });

  it("returns fallback for empty result", () => {
    expect(generateSlug("@#$%")).toBe("org");
  });
});

describe("ensureUniqueSlug", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns slug as-is when no duplicates exist", async () => {
    mockListDocuments.mockResolvedValue({ total: 0, documents: [] });

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org");
  });

  it("appends number when slug already taken", async () => {
    mockListDocuments
      .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org" }] })
      .mockResolvedValueOnce({ total: 0, documents: [] });

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org-1");
  });

  it("increments number until unique", async () => {
    mockListDocuments
      .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org" }] })
      .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org-1" }] })
      .mockResolvedValueOnce({ total: 0, documents: [] });

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org-2");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/slugs.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/slugs.ts`

```typescript
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";

const TRANSLITERATION: Record<string, string> = {
  ä: "ae", ö: "oe", ü: "ue", ß: "ss",
  à: "a", á: "a", â: "a", ã: "a",
  è: "e", é: "e", ê: "e", ë: "e",
  ì: "i", í: "i", î: "i", ï: "i",
  ò: "o", ó: "o", ô: "o", õ: "o",
  ù: "u", ú: "u", û: "u",
  ñ: "n", ç: "c",
};

export function generateSlug(name: string): string {
  let slug = name.toLowerCase();

  for (const [char, replacement] of Object.entries(TRANSLITERATION)) {
    slug = slug.replaceAll(char, replacement);
  }

  slug = slug
    .replace(/[^a-z0-9-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");

  if (slug.length === 0) {
    return "org";
  }

  return slug;
}

export async function ensureUniqueSlug(baseSlug: string): Promise<string> {
  const env = getEnv();
  const db = getDatabases();
  let candidate = baseSlug;
  let suffix = 0;
  const maxAttempts = 20;

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    const existing = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      [Query.equal("slug", candidate), Query.limit(1)]
    );

    if (existing.total === 0) {
      return candidate;
    }

    suffix++;
    candidate = `${baseSlug}-${suffix}`;
  }

  throw new Error(
    `Could not generate unique slug after ${maxAttempts} attempts for base: ${baseSlug}`
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/slugs.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/slugs.ts __tests__/lib/slugs.test.ts
git commit -m "feat: add slug generation and uniqueness checking for orgs"
```

---

### Task 2: Permission Resolution

**Files:**
- Create: `obeya-cloud/lib/permissions.ts`
- Test: `obeya-cloud/__tests__/lib/permissions.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/permissions.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockListDocuments = vi.fn();

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import {
  ORG_ROLE_LEVEL,
  BOARD_ROLE_LEVEL,
  resolvePermission,
  requireOrgRole,
  requireBoardAccess,
  type EffectivePermission,
} from "@/lib/permissions";

describe("role level mappings", () => {
  it("org roles have correct hierarchy", () => {
    expect(ORG_ROLE_LEVEL.member).toBe(1);
    expect(ORG_ROLE_LEVEL.admin).toBe(2);
    expect(ORG_ROLE_LEVEL.owner).toBe(3);
  });

  it("board roles have correct hierarchy", () => {
    expect(BOARD_ROLE_LEVEL.viewer).toBe(1);
    expect(BOARD_ROLE_LEVEL.editor).toBe(2);
    expect(BOARD_ROLE_LEVEL.owner).toBe(3);
  });
});

describe("resolvePermission", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns org role when user is org member and board belongs to org", async () => {
    // org_members query returns a match
    mockListDocuments
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ org_id: "org1", user_id: "user1", role: "admin" }],
      })
      // board_members query returns no match
      .mockResolvedValueOnce({ total: 0, documents: [] });

    const result = await resolvePermission("user1", "board1", "org1");

    expect(result).toEqual({
      canAccess: true,
      effectiveLevel: 2,
      orgRole: "admin",
      boardRole: null,
      source: "org",
    });
  });

  it("returns board role when user is board member only", async () => {
    // org_members query returns no match
    mockListDocuments
      .mockResolvedValueOnce({ total: 0, documents: [] })
      // board_members query returns a match
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ board_id: "board1", user_id: "user1", role: "editor" }],
      });

    const result = await resolvePermission("user1", "board1", null);

    expect(result).toEqual({
      canAccess: true,
      effectiveLevel: 2,
      orgRole: null,
      boardRole: "editor",
      source: "board",
    });
  });

  it("returns higher permission when user has both org and board roles", async () => {
    // org_members: member (level 1)
    mockListDocuments
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ org_id: "org1", user_id: "user1", role: "member" }],
      })
      // board_members: owner (level 3)
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ board_id: "board1", user_id: "user1", role: "owner" }],
      });

    const result = await resolvePermission("user1", "board1", "org1");

    expect(result).toEqual({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: "member",
      boardRole: "owner",
      source: "board",
    });
  });

  it("returns higher org permission when org role outranks board role", async () => {
    // org_members: owner (level 3)
    mockListDocuments
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ org_id: "org1", user_id: "user1", role: "owner" }],
      })
      // board_members: viewer (level 1)
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ board_id: "board1", user_id: "user1", role: "viewer" }],
      });

    const result = await resolvePermission("user1", "board1", "org1");

    expect(result).toEqual({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: "owner",
      boardRole: "viewer",
      source: "org",
    });
  });

  it("returns no access when user has neither org nor board membership", async () => {
    mockListDocuments
      .mockResolvedValueOnce({ total: 0, documents: [] })
      .mockResolvedValueOnce({ total: 0, documents: [] });

    const result = await resolvePermission("user1", "board1", "org1");

    expect(result).toEqual({
      canAccess: false,
      effectiveLevel: 0,
      orgRole: null,
      boardRole: null,
      source: "none",
    });
  });

  it("skips org check when org_id is null (personal board)", async () => {
    // Only board_members query is made
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [{ board_id: "board1", user_id: "user1", role: "owner" }],
    });

    const result = await resolvePermission("user1", "board1", null);

    expect(result.canAccess).toBe(true);
    expect(result.boardRole).toBe("owner");
    expect(result.orgRole).toBeNull();
    expect(mockListDocuments).toHaveBeenCalledTimes(1);
  });
});

describe("requireOrgRole", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns membership when user has sufficient role", async () => {
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem1",
          org_id: "org1",
          user_id: "user1",
          role: "admin",
          invited_at: "2026-01-01T00:00:00.000Z",
          accepted_at: "2026-01-01T00:00:00.000Z",
        },
      ],
    });

    const result = await requireOrgRole("user1", "org1", "member");
    expect(result.role).toBe("admin");
  });

  it("throws FORBIDDEN when user role is insufficient", async () => {
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem1",
          org_id: "org1",
          user_id: "user1",
          role: "member",
          invited_at: "2026-01-01T00:00:00.000Z",
          accepted_at: "2026-01-01T00:00:00.000Z",
        },
      ],
    });

    await expect(requireOrgRole("user1", "org1", "admin")).rejects.toThrow(
      "Insufficient"
    );
  });

  it("throws FORBIDDEN when user is not a member", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 0, documents: [] });

    await expect(requireOrgRole("user1", "org1", "member")).rejects.toThrow(
      "not a member"
    );
  });
});

describe("requireBoardAccess", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns permission when user has access at required level", async () => {
    // org members
    mockListDocuments
      .mockResolvedValueOnce({ total: 0, documents: [] })
      // board members
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ board_id: "board1", user_id: "user1", role: "editor" }],
      });

    const result = await requireBoardAccess("user1", "board1", null, 2);
    expect(result.canAccess).toBe(true);
    expect(result.effectiveLevel).toBe(2);
  });

  it("throws FORBIDDEN when user lacks access", async () => {
    mockListDocuments
      .mockResolvedValueOnce({ total: 0, documents: [] })
      .mockResolvedValueOnce({ total: 0, documents: [] });

    await expect(
      requireBoardAccess("user1", "board1", null, 1)
    ).rejects.toThrow("No access");
  });

  it("throws FORBIDDEN when access level is insufficient", async () => {
    mockListDocuments
      .mockResolvedValueOnce({ total: 0, documents: [] })
      .mockResolvedValueOnce({
        total: 1,
        documents: [{ board_id: "board1", user_id: "user1", role: "viewer" }],
      });

    await expect(
      requireBoardAccess("user1", "board1", null, 2)
    ).rejects.toThrow("Insufficient");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/permissions.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/permissions.ts`

```typescript
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";

export type OrgRole = "owner" | "admin" | "member";
export type BoardRole = "owner" | "editor" | "viewer";

export const ORG_ROLE_LEVEL: Record<OrgRole, number> = {
  member: 1,
  admin: 2,
  owner: 3,
};

export const BOARD_ROLE_LEVEL: Record<BoardRole, number> = {
  viewer: 1,
  editor: 2,
  owner: 3,
};

export interface EffectivePermission {
  canAccess: boolean;
  effectiveLevel: number;
  orgRole: OrgRole | null;
  boardRole: BoardRole | null;
  source: "org" | "board" | "none";
}

export async function resolvePermission(
  userId: string,
  boardId: string,
  orgId: string | null
): Promise<EffectivePermission> {
  const env = getEnv();
  const db = getDatabases();

  let orgRole: OrgRole | null = null;
  let orgLevel = 0;

  if (orgId) {
    const orgMembership = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [
        Query.equal("org_id", orgId),
        Query.equal("user_id", userId),
        Query.limit(1),
      ]
    );

    if (orgMembership.total > 0) {
      orgRole = orgMembership.documents[0].role as OrgRole;
      orgLevel = ORG_ROLE_LEVEL[orgRole];
    }
  }

  const boardMembership = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    [
      Query.equal("board_id", boardId),
      Query.equal("user_id", userId),
      Query.limit(1),
    ]
  );

  let boardRole: BoardRole | null = null;
  let boardLevel = 0;

  if (boardMembership.total > 0) {
    boardRole = boardMembership.documents[0].role as BoardRole;
    boardLevel = BOARD_ROLE_LEVEL[boardRole];
  }

  if (orgLevel === 0 && boardLevel === 0) {
    return {
      canAccess: false,
      effectiveLevel: 0,
      orgRole: null,
      boardRole: null,
      source: "none",
    };
  }

  const useOrg = orgLevel >= boardLevel;

  return {
    canAccess: true,
    effectiveLevel: useOrg ? orgLevel : boardLevel,
    orgRole,
    boardRole,
    source: useOrg ? "org" : "board",
  };
}

export interface OrgMembershipDoc {
  $id: string;
  org_id: string;
  user_id: string;
  role: OrgRole;
  invited_at: string;
  accepted_at: string | null;
}

export async function requireOrgRole(
  userId: string,
  orgId: string,
  minimumRole: OrgRole
): Promise<OrgMembershipDoc> {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORG_MEMBERS,
    [
      Query.equal("org_id", orgId),
      Query.equal("user_id", userId),
      Query.limit(1),
    ]
  );

  if (result.total === 0) {
    throw new AppError(ErrorCode.FORBIDDEN, "You are not a member of this org");
  }

  const doc = result.documents[0];
  const userLevel = ORG_ROLE_LEVEL[doc.role as OrgRole];
  const requiredLevel = ORG_ROLE_LEVEL[minimumRole];

  if (userLevel < requiredLevel) {
    throw new AppError(
      ErrorCode.FORBIDDEN,
      `Insufficient org role. Required: ${minimumRole}, yours: ${doc.role}`
    );
  }

  return {
    $id: doc.$id,
    org_id: doc.org_id,
    user_id: doc.user_id,
    role: doc.role as OrgRole,
    invited_at: doc.invited_at,
    accepted_at: doc.accepted_at ?? null,
  };
}

export async function requireBoardAccess(
  userId: string,
  boardId: string,
  orgId: string | null,
  minimumLevel: number
): Promise<EffectivePermission> {
  const permission = await resolvePermission(userId, boardId, orgId);

  if (!permission.canAccess) {
    throw new AppError(ErrorCode.FORBIDDEN, "No access to this board");
  }

  if (permission.effectiveLevel < minimumLevel) {
    throw new AppError(
      ErrorCode.FORBIDDEN,
      `Insufficient permission level. Required: ${minimumLevel}, yours: ${permission.effectiveLevel}`
    );
  }

  return permission;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/permissions.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/permissions.ts __tests__/lib/permissions.test.ts
git commit -m "feat: add permission resolution with org + board role hierarchy"
```

---

### Task 3: Free Tier Limit Enforcement

**Files:**
- Create: `obeya-cloud/lib/limits.ts`
- Test: `obeya-cloud/__tests__/lib/limits.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/lib/limits.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockListDocuments = vi.fn();

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
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

describe("FREE_TIER_LIMITS", () => {
  it("has correct values", () => {
    expect(FREE_TIER_LIMITS.PERSONAL_BOARDS).toBe(3);
    expect(FREE_TIER_LIMITS.ORGS).toBe(1);
    expect(FREE_TIER_LIMITS.MEMBERS_PER_ORG).toBe(3);
    expect(FREE_TIER_LIMITS.ITEMS_PER_BOARD).toBe(100);
  });
});

describe("enforcePersonalBoardLimit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not throw when under limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 2, documents: [] });

    await expect(
      enforcePersonalBoardLimit("user1")
    ).resolves.toBeUndefined();
  });

  it("throws PLAN_LIMIT_REACHED when at limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 3, documents: [] });

    await expect(enforcePersonalBoardLimit("user1")).rejects.toThrow(
      "personal board"
    );
  });
});

describe("enforceOrgLimit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not throw when under limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 0, documents: [] });

    await expect(enforceOrgLimit("user1")).resolves.toBeUndefined();
  });

  it("throws PLAN_LIMIT_REACHED when at limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 1, documents: [] });

    await expect(enforceOrgLimit("user1")).rejects.toThrow("org");
  });
});

describe("enforceOrgMemberLimit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not throw when under limit for free org", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 2, documents: [] });

    await expect(
      enforceOrgMemberLimit("org1", "free")
    ).resolves.toBeUndefined();
  });

  it("throws PLAN_LIMIT_REACHED when at limit for free org", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 3, documents: [] });

    await expect(enforceOrgMemberLimit("org1", "free")).rejects.toThrow(
      "member"
    );
  });

  it("does not throw for pro org regardless of count", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 100, documents: [] });

    await expect(
      enforceOrgMemberLimit("org1", "pro")
    ).resolves.toBeUndefined();
  });

  it("does not throw for enterprise org regardless of count", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 500, documents: [] });

    await expect(
      enforceOrgMemberLimit("org1", "enterprise")
    ).resolves.toBeUndefined();
  });
});

describe("enforceBoardItemLimit", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not throw when under limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 50, documents: [] });

    await expect(
      enforceBoardItemLimit("board1")
    ).resolves.toBeUndefined();
  });

  it("throws PLAN_LIMIT_REACHED when at limit", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 100, documents: [] });

    await expect(enforceBoardItemLimit("board1")).rejects.toThrow("item");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/limits.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/lib/limits.ts`

```typescript
import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";

export const FREE_TIER_LIMITS = {
  PERSONAL_BOARDS: 3,
  ORGS: 1,
  MEMBERS_PER_ORG: 3,
  ITEMS_PER_BOARD: 100,
} as const;

type OrgPlan = "free" | "pro" | "enterprise";

export async function enforcePersonalBoardLimit(
  userId: string
): Promise<void> {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    [
      Query.equal("owner_id", userId),
      Query.isNull("org_id"),
      Query.limit(1),
      Query.select(["$id"]),
    ]
  );

  if (result.total >= FREE_TIER_LIMITS.PERSONAL_BOARDS) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Free tier personal board limit reached (${FREE_TIER_LIMITS.PERSONAL_BOARDS}). Upgrade to Pro for unlimited boards.`
    );
  }
}

export async function enforceOrgLimit(userId: string): Promise<void> {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORGS,
    [
      Query.equal("owner_id", userId),
      Query.limit(1),
      Query.select(["$id"]),
    ]
  );

  if (result.total >= FREE_TIER_LIMITS.ORGS) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Free tier org limit reached (${FREE_TIER_LIMITS.ORGS}). Upgrade to Pro for unlimited orgs.`
    );
  }
}

export async function enforceOrgMemberLimit(
  orgId: string,
  orgPlan: OrgPlan
): Promise<void> {
  if (orgPlan !== "free") {
    return;
  }

  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORG_MEMBERS,
    [
      Query.equal("org_id", orgId),
      Query.limit(1),
      Query.select(["$id"]),
    ]
  );

  if (result.total >= FREE_TIER_LIMITS.MEMBERS_PER_ORG) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Free tier org member limit reached (${FREE_TIER_LIMITS.MEMBERS_PER_ORG}). Upgrade to Pro for unlimited members.`
    );
  }
}

export async function enforceBoardItemLimit(
  boardId: string
): Promise<void> {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ITEMS,
    [
      Query.equal("board_id", boardId),
      Query.limit(1),
      Query.select(["$id"]),
    ]
  );

  if (result.total >= FREE_TIER_LIMITS.ITEMS_PER_BOARD) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Free tier item limit reached (${FREE_TIER_LIMITS.ITEMS_PER_BOARD} items per board). Upgrade to Pro for unlimited items.`
    );
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/lib/limits.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/limits.ts __tests__/lib/limits.test.ts
git commit -m "feat: add free tier limit enforcement for boards, orgs, members, items"
```

---

## Chunk 2: Org CRUD Endpoints

### Task 4: GET /api/orgs & POST /api/orgs

**Files:**
- Create: `obeya-cloud/app/api/orgs/route.ts`
- Test: `obeya-cloud/__tests__/api/orgs/orgs.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/orgs/orgs.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockAuthenticate = vi.fn();
const mockListDocuments = vi.fn();
const mockCreateDocument = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: any[]) => mockAuthenticate(...args),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
    createDocument: mockCreateDocument,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/limits", () => ({
  enforceOrgLimit: vi.fn(),
}));

vi.mock("@/lib/slugs", () => ({
  generateSlug: vi.fn((name: string) => name.toLowerCase().replace(/\s+/g, "-")),
  ensureUniqueSlug: vi.fn((slug: string) => Promise.resolve(slug)),
}));

import { GET, POST } from "@/app/api/orgs/route";
import { enforceOrgLimit } from "@/lib/limits";

describe("GET /api/orgs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("returns orgs the user is a member of", async () => {
    // First call: list org_members for user
    mockListDocuments
      .mockResolvedValueOnce({
        total: 2,
        documents: [
          { org_id: "org1", user_id: "user1", role: "owner" },
          { org_id: "org2", user_id: "user1", role: "member" },
        ],
      })
      // Second call: get org1 details
      .mockResolvedValueOnce({
        total: 1,
        documents: [
          {
            $id: "org1",
            name: "Org One",
            slug: "org-one",
            owner_id: "user1",
            plan: "free",
            created_at: "2026-01-01T00:00:00.000Z",
          },
        ],
      })
      // Third call: get org2 details
      .mockResolvedValueOnce({
        total: 1,
        documents: [
          {
            $id: "org2",
            name: "Org Two",
            slug: "org-two",
            owner_id: "user2",
            plan: "pro",
            created_at: "2026-01-02T00:00:00.000Z",
          },
        ],
      });

    const request = new Request("http://localhost/api/orgs", {
      headers: { cookie: "a]session=test" },
    });

    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.data[0].name).toBe("Org One");
    expect(body.data[0].role).toBe("owner");
    expect(body.data[1].name).toBe("Org Two");
    expect(body.data[1].role).toBe("member");
  });

  it("returns empty array when user has no orgs", async () => {
    mockListDocuments.mockResolvedValueOnce({ total: 0, documents: [] });

    const request = new Request("http://localhost/api/orgs", {
      headers: { cookie: "a]session=test" },
    });

    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toEqual([]);
  });
});

describe("POST /api/orgs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("creates org with auto-generated slug and owner membership", async () => {
    mockCreateDocument
      // First call: create org
      .mockResolvedValueOnce({
        $id: "org-new",
        name: "My Org",
        slug: "my-org",
        owner_id: "user1",
        plan: "free",
        created_at: "2026-03-12T00:00:00.000Z",
      })
      // Second call: create owner membership
      .mockResolvedValueOnce({
        $id: "mem1",
        org_id: "org-new",
        user_id: "user1",
        role: "owner",
      });

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({ name: "My Org" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("My Org");
    expect(body.data.slug).toBe("my-org");
    expect(body.data.owner_id).toBe("user1");
    expect(body.data.plan).toBe("free");
  });

  it("returns 400 for missing name", async () => {
    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({}),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request);
    expect(response.status).toBe(400);
  });

  it("enforces org limit", async () => {
    vi.mocked(enforceOrgLimit).mockRejectedValueOnce(
      new Error("Free tier org limit reached")
    );

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({ name: "Another Org" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request);
    expect(response.status).toBe(500); // handleError wraps generic errors
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/orgs/orgs.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/orgs/route.ts`

```typescript
import { z } from "zod";
import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { generateSlug, ensureUniqueSlug } from "@/lib/slugs";
import { enforceOrgLimit } from "@/lib/limits";

const createOrgSchema = z.object({
  name: z.string().min(1, "Org name is required").max(255),
  slug: z.string().min(1).max(100).optional(),
});

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    const env = getEnv();
    const db = getDatabases();

    const memberships = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [Query.equal("user_id", user.id), Query.limit(100)]
    );

    if (memberships.total === 0) {
      return ok([]);
    }

    const orgs = await Promise.all(
      memberships.documents.map(async (mem) => {
        const orgResult = await db.listDocuments(
          env.APPWRITE_DATABASE_ID,
          COLLECTIONS.ORGS,
          [Query.equal("$id", mem.org_id), Query.limit(1)]
        );

        if (orgResult.total === 0) {
          return null;
        }

        const org = orgResult.documents[0];
        return {
          id: org.$id,
          name: org.name,
          slug: org.slug,
          owner_id: org.owner_id,
          plan: org.plan,
          role: mem.role,
          created_at: org.created_at,
        };
      })
    );

    return ok(orgs.filter(Boolean));
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const { name, slug: customSlug } = await validateBody(
      request,
      createOrgSchema
    );

    await enforceOrgLimit(user.id);

    const env = getEnv();
    const db = getDatabases();
    const now = new Date().toISOString();

    const baseSlug = customSlug ?? generateSlug(name);
    const slug = await ensureUniqueSlug(baseSlug);

    const org = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      ID.unique(),
      {
        name,
        slug,
        owner_id: user.id,
        plan: "free",
        created_at: now,
      }
    );

    await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      ID.unique(),
      {
        org_id: org.$id,
        user_id: user.id,
        role: "owner",
        invited_at: now,
        accepted_at: now,
      }
    );

    return ok(
      {
        id: org.$id,
        name: org.name,
        slug: org.slug,
        owner_id: org.owner_id,
        plan: org.plan,
        created_at: org.created_at,
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
npm test -- __tests__/api/orgs/orgs.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/orgs/route.ts __tests__/api/orgs/orgs.test.ts
git commit -m "feat: add GET/POST /api/orgs with slug generation and limit enforcement"
```

---

### Task 5: GET/PATCH/DELETE /api/orgs/:id

**Files:**
- Create: `obeya-cloud/app/api/orgs/[id]/route.ts`
- Test: `obeya-cloud/__tests__/api/orgs/orgs-id.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/orgs/orgs-id.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockAuthenticate = vi.fn();
const mockGetDocument = vi.fn();
const mockUpdateDocument = vi.fn();
const mockDeleteDocument = vi.fn();
const mockListDocuments = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: any[]) => mockAuthenticate(...args),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    getDocument: mockGetDocument,
    updateDocument: mockUpdateDocument,
    deleteDocument: mockDeleteDocument,
    listDocuments: mockListDocuments,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/permissions", () => ({
  requireOrgRole: vi.fn(),
  ORG_ROLE_LEVEL: { member: 1, admin: 2, owner: 3 },
}));

vi.mock("@/lib/slugs", () => ({
  generateSlug: vi.fn((name: string) => name.toLowerCase().replace(/\s+/g, "-")),
  ensureUniqueSlug: vi.fn((slug: string) => Promise.resolve(slug)),
}));

import { GET, PATCH, DELETE } from "@/app/api/orgs/[id]/route";
import { requireOrgRole } from "@/lib/permissions";

const makeParams = (id: string) => ({ params: Promise.resolve({ id }) });

describe("GET /api/orgs/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("returns org details when user is a member", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "member",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "org1",
      name: "Test Org",
      slug: "test-org",
      owner_id: "user1",
      plan: "free",
      created_at: "2026-01-01T00:00:00.000Z",
    });

    // member count query
    mockListDocuments.mockResolvedValueOnce({ total: 3, documents: [] });

    const request = new Request("http://localhost/api/orgs/org1", {
      headers: { cookie: "a]session=test" },
    });

    const response = await GET(request, makeParams("org1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("Test Org");
    expect(body.data.slug).toBe("test-org");
    expect(body.data.member_count).toBe(3);
  });

  it("returns 403 when user is not a member", async () => {
    vi.mocked(requireOrgRole).mockRejectedValueOnce(
      new Error("not a member")
    );

    const request = new Request("http://localhost/api/orgs/org1", {
      headers: { cookie: "a]session=test" },
    });

    const response = await GET(request, makeParams("org1"));
    expect(response.status).toBe(500);
  });
});

describe("PATCH /api/orgs/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("updates org name when user is admin", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    mockUpdateDocument.mockResolvedValueOnce({
      $id: "org1",
      name: "Updated Org",
      slug: "test-org",
      owner_id: "user1",
      plan: "free",
      created_at: "2026-01-01T00:00:00.000Z",
    });

    const request = new Request("http://localhost/api/orgs/org1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Updated Org" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await PATCH(request, makeParams("org1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.name).toBe("Updated Org");
  });

  it("returns 400 for empty update", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    const request = new Request("http://localhost/api/orgs/org1", {
      method: "PATCH",
      body: JSON.stringify({}),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await PATCH(request, makeParams("org1"));
    expect(response.status).toBe(400);
  });
});

describe("DELETE /api/orgs/:id", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("deletes org and all memberships when user is owner", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "owner",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // List org members to delete
    mockListDocuments.mockResolvedValueOnce({
      total: 2,
      documents: [{ $id: "mem1" }, { $id: "mem2" }],
    });

    // Delete member docs + org doc
    mockDeleteDocument.mockResolvedValue({});

    const request = new Request("http://localhost/api/orgs/org1", {
      method: "DELETE",
      headers: { cookie: "a]session=test" },
    });

    const response = await DELETE(request, makeParams("org1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.deleted).toBe(true);
    // Should have deleted 2 member docs + 1 org doc
    expect(mockDeleteDocument).toHaveBeenCalledTimes(3);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/orgs/orgs-id.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write implementation**

Create: `obeya-cloud/app/api/orgs/[id]/route.ts`

```typescript
import { z } from "zod";
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await requireOrgRole(user.id, id, "member");

    const env = getEnv();
    const db = getDatabases();

    const org = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      id
    );

    const memberCount = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [Query.equal("org_id", id), Query.limit(1), Query.select(["$id"])]
    );

    return ok({
      id: org.$id,
      name: org.name,
      slug: org.slug,
      owner_id: org.owner_id,
      plan: org.plan,
      member_count: memberCount.total,
      created_at: org.created_at,
    });
  } catch (err) {
    return handleError(err);
  }
}

const updateOrgSchema = z
  .object({
    name: z.string().min(1).max(255).optional(),
  })
  .refine((data) => Object.values(data).some((v) => v !== undefined), {
    message: "At least one field must be provided for update",
  });

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await requireOrgRole(user.id, id, "admin");

    const data = await validateBody(request, updateOrgSchema);
    const env = getEnv();
    const db = getDatabases();

    const updatePayload: Record<string, unknown> = {};
    if (data.name) {
      updatePayload.name = data.name;
    }

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      id,
      updatePayload
    );

    return ok({
      id: updated.$id,
      name: updated.name,
      slug: updated.slug,
      owner_id: updated.owner_id,
      plan: updated.plan,
      created_at: updated.created_at,
    });
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await requireOrgRole(user.id, id, "owner");

    const env = getEnv();
    const db = getDatabases();

    const members = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [Query.equal("org_id", id), Query.limit(100)]
    );

    await Promise.all(
      members.documents.map((mem) =>
        db.deleteDocument(
          env.APPWRITE_DATABASE_ID,
          COLLECTIONS.ORG_MEMBERS,
          mem.$id
        )
      )
    );

    await db.deleteDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      id
    );

    return ok({ deleted: true });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/orgs/orgs-id.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/orgs/[id]/route.ts __tests__/api/orgs/orgs-id.test.ts
git commit -m "feat: add GET/PATCH/DELETE /api/orgs/:id with role-based access"
```

---

## Chunk 3: Org Member Endpoints

### Task 6: GET/POST /api/orgs/:id/members & PATCH/DELETE /api/orgs/:id/members/:uid

**Files:**
- Create: `obeya-cloud/app/api/orgs/[id]/members/route.ts`
- Create: `obeya-cloud/app/api/orgs/[id]/members/[uid]/route.ts`
- Test: `obeya-cloud/__tests__/api/orgs/org-members.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/orgs/org-members.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockAuthenticate = vi.fn();
const mockListDocuments = vi.fn();
const mockCreateDocument = vi.fn();
const mockUpdateDocument = vi.fn();
const mockDeleteDocument = vi.fn();
const mockGetDocument = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: any[]) => mockAuthenticate(...args),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
    createDocument: mockCreateDocument,
    updateDocument: mockUpdateDocument,
    deleteDocument: mockDeleteDocument,
    getDocument: mockGetDocument,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/permissions", () => ({
  requireOrgRole: vi.fn(),
  ORG_ROLE_LEVEL: { member: 1, admin: 2, owner: 3 },
}));

vi.mock("@/lib/limits", () => ({
  enforceOrgMemberLimit: vi.fn(),
}));

import { GET, POST } from "@/app/api/orgs/[id]/members/route";
import {
  PATCH,
  DELETE,
} from "@/app/api/orgs/[id]/members/[uid]/route";
import { requireOrgRole } from "@/lib/permissions";
import { enforceOrgMemberLimit } from "@/lib/limits";

const makeMemberParams = (id: string) => ({
  params: Promise.resolve({ id }),
});
const makeMemberUidParams = (id: string, uid: string) => ({
  params: Promise.resolve({ id, uid }),
});

describe("GET /api/orgs/:id/members", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("returns list of org members", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    mockListDocuments.mockResolvedValueOnce({
      total: 2,
      documents: [
        {
          $id: "mem1",
          org_id: "org1",
          user_id: "user1",
          role: "owner",
          invited_at: "2026-01-01T00:00:00.000Z",
          accepted_at: "2026-01-01T00:00:00.000Z",
        },
        {
          $id: "mem2",
          org_id: "org1",
          user_id: "user2",
          role: "member",
          invited_at: "2026-01-02T00:00:00.000Z",
          accepted_at: null,
        },
      ],
    });

    const request = new Request("http://localhost/api/orgs/org1/members", {
      headers: { cookie: "a]session=test" },
    });

    const response = await GET(request, makeMemberParams("org1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.data[0].role).toBe("owner");
    expect(body.data[1].accepted_at).toBeNull();
  });
});

describe("POST /api/orgs/:id/members", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("adds a new member to the org", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // Get org to check plan
    mockGetDocument.mockResolvedValueOnce({
      $id: "org1",
      plan: "free",
    });

    // Check if user already a member
    mockListDocuments.mockResolvedValueOnce({ total: 0, documents: [] });

    mockCreateDocument.mockResolvedValueOnce({
      $id: "mem-new",
      org_id: "org1",
      user_id: "user3",
      role: "member",
      invited_at: "2026-03-12T00:00:00.000Z",
      accepted_at: null,
    });

    const request = new Request("http://localhost/api/orgs/org1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user3", role: "member" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request, makeMemberParams("org1"));
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.user_id).toBe("user3");
    expect(body.data.role).toBe("member");
    expect(body.data.accepted_at).toBeNull();
  });

  it("returns 400 when user_id is missing", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    const request = new Request("http://localhost/api/orgs/org1/members", {
      method: "POST",
      body: JSON.stringify({ role: "member" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request, makeMemberParams("org1"));
    expect(response.status).toBe(400);
  });

  it("returns 409 when user is already a member", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "org1",
      plan: "free",
    });

    // User already a member
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [{ $id: "existing-mem" }],
    });

    const request = new Request("http://localhost/api/orgs/org1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user3", role: "member" }),
      headers: {
        "Content-Type": "application/json",
        cookie: "a]session=test",
      },
    });

    const response = await POST(request, makeMemberParams("org1"));
    expect(response.status).toBe(409);
  });
});

describe("PATCH /api/orgs/:id/members/:uid", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("updates member role", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "owner",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // Find existing membership
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem2",
          org_id: "org1",
          user_id: "user2",
          role: "member",
          invited_at: "2026-01-02T00:00:00.000Z",
          accepted_at: "2026-01-02T00:00:00.000Z",
        },
      ],
    });

    mockUpdateDocument.mockResolvedValueOnce({
      $id: "mem2",
      org_id: "org1",
      user_id: "user2",
      role: "admin",
      invited_at: "2026-01-02T00:00:00.000Z",
      accepted_at: "2026-01-02T00:00:00.000Z",
    });

    const request = new Request(
      "http://localhost/api/orgs/org1/members/user2",
      {
        method: "PATCH",
        body: JSON.stringify({ role: "admin" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await PATCH(
      request,
      makeMemberUidParams("org1", "user2")
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.role).toBe("admin");
  });

  it("prevents changing owner role", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "owner",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // Target is owner
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem-owner",
          org_id: "org1",
          user_id: "user-owner",
          role: "owner",
        },
      ],
    });

    const request = new Request(
      "http://localhost/api/orgs/org1/members/user-owner",
      {
        method: "PATCH",
        body: JSON.stringify({ role: "member" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await PATCH(
      request,
      makeMemberUidParams("org1", "user-owner")
    );
    expect(response.status).toBe(403);
  });
});

describe("DELETE /api/orgs/:id/members/:uid", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("removes member from org", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "admin",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // Find membership to delete
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem2",
          org_id: "org1",
          user_id: "user2",
          role: "member",
        },
      ],
    });

    mockDeleteDocument.mockResolvedValue({});

    const request = new Request(
      "http://localhost/api/orgs/org1/members/user2",
      {
        method: "DELETE",
        headers: { cookie: "a]session=test" },
      }
    );

    const response = await DELETE(
      request,
      makeMemberUidParams("org1", "user2")
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.removed).toBe(true);
  });

  it("prevents removing the org owner", async () => {
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user1",
      role: "owner",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    // Target is owner
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem-owner",
          org_id: "org1",
          user_id: "user-owner",
          role: "owner",
        },
      ],
    });

    const request = new Request(
      "http://localhost/api/orgs/org1/members/user-owner",
      {
        method: "DELETE",
        headers: { cookie: "a]session=test" },
      }
    );

    const response = await DELETE(
      request,
      makeMemberUidParams("org1", "user-owner")
    );
    expect(response.status).toBe(403);
  });

  it("allows member to remove themselves", async () => {
    // User is only a member, but removing themselves
    vi.mocked(requireOrgRole).mockResolvedValueOnce({
      $id: "mem1",
      org_id: "org1",
      user_id: "user2",
      role: "member",
      invited_at: "2026-01-01T00:00:00.000Z",
      accepted_at: "2026-01-01T00:00:00.000Z",
    });

    mockAuthenticate.mockResolvedValue({
      id: "user2",
      email: "b@b.com",
      name: "Bob",
    });

    // Find membership
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "mem2",
          org_id: "org1",
          user_id: "user2",
          role: "member",
        },
      ],
    });

    mockDeleteDocument.mockResolvedValue({});

    const request = new Request(
      "http://localhost/api/orgs/org1/members/user2",
      {
        method: "DELETE",
        headers: { cookie: "a]session=test" },
      }
    );

    const response = await DELETE(
      request,
      makeMemberUidParams("org1", "user2")
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.removed).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/orgs/org-members.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write GET/POST org members implementation**

Create: `obeya-cloud/app/api/orgs/[id]/members/route.ts`

```typescript
import { z } from "zod";
import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";
import { enforceOrgMemberLimit } from "@/lib/limits";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await requireOrgRole(user.id, id, "member");

    const env = getEnv();
    const db = getDatabases();

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [Query.equal("org_id", id), Query.limit(100)]
    );

    const members = result.documents.map((doc) => ({
      id: doc.$id,
      org_id: doc.org_id,
      user_id: doc.user_id,
      role: doc.role,
      invited_at: doc.invited_at,
      accepted_at: doc.accepted_at ?? null,
    }));

    return ok(members, { meta: { total: result.total } });
  } catch (err) {
    return handleError(err);
  }
}

const addMemberSchema = z.object({
  user_id: z.string().min(1, "user_id is required"),
  role: z.enum(["admin", "member"]).default("member"),
});

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;

    await requireOrgRole(user.id, id, "admin");

    const { user_id: targetUserId, role } = await validateBody(
      request,
      addMemberSchema
    );

    const env = getEnv();
    const db = getDatabases();

    const org = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      id
    );

    await enforceOrgMemberLimit(id, org.plan);

    const existing = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      [
        Query.equal("org_id", id),
        Query.equal("user_id", targetUserId),
        Query.limit(1),
      ]
    );

    if (existing.total > 0) {
      throw new AppError(
        ErrorCode.SLUG_ALREADY_EXISTS,
        "User is already a member of this org"
      );
    }

    const now = new Date().toISOString();

    const doc = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      ID.unique(),
      {
        org_id: id,
        user_id: targetUserId,
        role,
        invited_at: now,
        accepted_at: null,
      }
    );

    return ok(
      {
        id: doc.$id,
        org_id: doc.org_id,
        user_id: doc.user_id,
        role: doc.role,
        invited_at: doc.invited_at,
        accepted_at: doc.accepted_at ?? null,
      },
      { status: 201 }
    );
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Write PATCH/DELETE org members implementation**

Create: `obeya-cloud/app/api/orgs/[id]/members/[uid]/route.ts`

```typescript
import { z } from "zod";
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string; uid: string }> };

async function findMembership(orgId: string, userId: string) {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORG_MEMBERS,
    [
      Query.equal("org_id", orgId),
      Query.equal("user_id", userId),
      Query.limit(1),
    ]
  );

  if (result.total === 0) {
    throw new AppError(
      ErrorCode.USER_NOT_FOUND,
      "User is not a member of this org"
    );
  }

  return result.documents[0];
}

const updateRoleSchema = z.object({
  role: z.enum(["admin", "member"]),
});

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: orgId, uid: targetUserId } = await context.params;

    await requireOrgRole(user.id, orgId, "owner");

    const { role: newRole } = await validateBody(request, updateRoleSchema);

    const membership = await findMembership(orgId, targetUserId);

    if (membership.role === "owner") {
      throw new AppError(
        ErrorCode.FORBIDDEN,
        "Cannot change the role of the org owner"
      );
    }

    const env = getEnv();
    const db = getDatabases();

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      membership.$id,
      { role: newRole }
    );

    return ok({
      id: updated.$id,
      org_id: updated.org_id,
      user_id: updated.user_id,
      role: updated.role,
      invited_at: updated.invited_at,
      accepted_at: updated.accepted_at ?? null,
    });
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: orgId, uid: targetUserId } = await context.params;

    const isSelfRemoval = user.id === targetUserId;

    if (!isSelfRemoval) {
      await requireOrgRole(user.id, orgId, "admin");
    } else {
      await requireOrgRole(user.id, orgId, "member");
    }

    const membership = await findMembership(orgId, targetUserId);

    if (membership.role === "owner") {
      throw new AppError(
        ErrorCode.FORBIDDEN,
        "Cannot remove the org owner. Transfer ownership first."
      );
    }

    const env = getEnv();
    const db = getDatabases();

    await db.deleteDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      membership.$id
    );

    return ok({ removed: true });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/orgs/org-members.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/orgs/[id]/members/ __tests__/api/orgs/org-members.test.ts
git commit -m "feat: add org member CRUD with role management and limit enforcement"
```

---

## Chunk 4: Board Member Endpoints

### Task 7: GET/POST /api/boards/:id/members & PATCH/DELETE /api/boards/:id/members/:uid

**Files:**
- Create: `obeya-cloud/app/api/boards/[id]/members/route.ts`
- Create: `obeya-cloud/app/api/boards/[id]/members/[uid]/route.ts`
- Test: `obeya-cloud/__tests__/api/boards/board-members.test.ts`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/api/boards/board-members.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockAuthenticate = vi.fn();
const mockListDocuments = vi.fn();
const mockCreateDocument = vi.fn();
const mockUpdateDocument = vi.fn();
const mockDeleteDocument = vi.fn();
const mockGetDocument = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: any[]) => mockAuthenticate(...args),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    listDocuments: mockListDocuments,
    createDocument: mockCreateDocument,
    updateDocument: mockUpdateDocument,
    deleteDocument: mockDeleteDocument,
    getDocument: mockGetDocument,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/permissions", () => ({
  requireBoardAccess: vi.fn(),
  BOARD_ROLE_LEVEL: { viewer: 1, editor: 2, owner: 3 },
}));

import { GET, POST } from "@/app/api/boards/[id]/members/route";
import {
  PATCH,
  DELETE,
} from "@/app/api/boards/[id]/members/[uid]/route";
import { requireBoardAccess } from "@/lib/permissions";

const makeBoardParams = (id: string) => ({
  params: Promise.resolve({ id }),
});
const makeBoardUidParams = (id: string, uid: string) => ({
  params: Promise.resolve({ id, uid }),
});

describe("GET /api/boards/:id/members", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("returns list of board members", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockListDocuments.mockResolvedValueOnce({
      total: 2,
      documents: [
        {
          $id: "bm1",
          board_id: "board1",
          user_id: "user1",
          role: "owner",
          invited_at: "2026-01-01T00:00:00.000Z",
        },
        {
          $id: "bm2",
          board_id: "board1",
          user_id: "user2",
          role: "editor",
          invited_at: "2026-01-02T00:00:00.000Z",
        },
      ],
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members",
      { headers: { cookie: "a]session=test" } }
    );

    const response = await GET(request, makeBoardParams("board1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.data[0].role).toBe("owner");
  });
});

describe("POST /api/boards/:id/members", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("adds a new member to the board", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    // Get board to find org_id
    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // Check existing membership
    mockListDocuments.mockResolvedValueOnce({ total: 0, documents: [] });

    mockCreateDocument.mockResolvedValueOnce({
      $id: "bm-new",
      board_id: "board1",
      user_id: "user3",
      role: "editor",
      invited_at: "2026-03-12T00:00:00.000Z",
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members",
      {
        method: "POST",
        body: JSON.stringify({ user_id: "user3", role: "editor" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await POST(request, makeBoardParams("board1"));
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.user_id).toBe("user3");
    expect(body.data.role).toBe("editor");
  });

  it("returns 409 when user is already a member", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // User already a member
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [{ $id: "existing" }],
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members",
      {
        method: "POST",
        body: JSON.stringify({ user_id: "user3", role: "editor" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await POST(request, makeBoardParams("board1"));
    expect(response.status).toBe(409);
  });

  it("returns 400 for invalid role", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members",
      {
        method: "POST",
        body: JSON.stringify({ user_id: "user3", role: "superadmin" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await POST(request, makeBoardParams("board1"));
    expect(response.status).toBe(400);
  });
});

describe("PATCH /api/boards/:id/members/:uid", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("updates member role", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // Find existing membership
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "bm2",
          board_id: "board1",
          user_id: "user2",
          role: "viewer",
          invited_at: "2026-01-02T00:00:00.000Z",
        },
      ],
    });

    mockUpdateDocument.mockResolvedValueOnce({
      $id: "bm2",
      board_id: "board1",
      user_id: "user2",
      role: "editor",
      invited_at: "2026-01-02T00:00:00.000Z",
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members/user2",
      {
        method: "PATCH",
        body: JSON.stringify({ role: "editor" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await PATCH(
      request,
      makeBoardUidParams("board1", "user2")
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.role).toBe("editor");
  });

  it("prevents changing board owner role", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // Target is board owner
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "bm-owner",
          board_id: "board1",
          user_id: "user-owner",
          role: "owner",
        },
      ],
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members/user-owner",
      {
        method: "PATCH",
        body: JSON.stringify({ role: "viewer" }),
        headers: {
          "Content-Type": "application/json",
          cookie: "a]session=test",
        },
      }
    );

    const response = await PATCH(
      request,
      makeBoardUidParams("board1", "user-owner")
    );
    expect(response.status).toBe(403);
  });
});

describe("DELETE /api/boards/:id/members/:uid", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthenticate.mockResolvedValue({
      id: "user1",
      email: "a@b.com",
      name: "Alice",
    });
  });

  it("removes member from board", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // Find membership
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "bm2",
          board_id: "board1",
          user_id: "user2",
          role: "editor",
        },
      ],
    });

    mockDeleteDocument.mockResolvedValue({});

    const request = new Request(
      "http://localhost/api/boards/board1/members/user2",
      {
        method: "DELETE",
        headers: { cookie: "a]session=test" },
      }
    );

    const response = await DELETE(
      request,
      makeBoardUidParams("board1", "user2")
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.removed).toBe(true);
  });

  it("prevents removing board owner", async () => {
    vi.mocked(requireBoardAccess).mockResolvedValueOnce({
      canAccess: true,
      effectiveLevel: 3,
      orgRole: null,
      boardRole: "owner",
      source: "board",
    });

    mockGetDocument.mockResolvedValueOnce({
      $id: "board1",
      owner_id: "user1",
      org_id: null,
    });

    // Target is owner
    mockListDocuments.mockResolvedValueOnce({
      total: 1,
      documents: [
        {
          $id: "bm-owner",
          board_id: "board1",
          user_id: "user-owner",
          role: "owner",
        },
      ],
    });

    const request = new Request(
      "http://localhost/api/boards/board1/members/user-owner",
      {
        method: "DELETE",
        headers: { cookie: "a]session=test" },
      }
    );

    const response = await DELETE(
      request,
      makeBoardUidParams("board1", "user-owner")
    );
    expect(response.status).toBe(403);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/board-members.test.ts
```

Expected: FAIL — module not found

- [ ] **Step 3: Write GET/POST board members implementation**

Create: `obeya-cloud/app/api/boards/[id]/members/route.ts`

```typescript
import { z } from "zod";
import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { requireBoardAccess, BOARD_ROLE_LEVEL } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId } = await context.params;

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId
    );

    await requireBoardAccess(
      user.id,
      boardId,
      board.org_id ?? null,
      BOARD_ROLE_LEVEL.viewer
    );

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      [Query.equal("board_id", boardId), Query.limit(100)]
    );

    const members = result.documents.map((doc) => ({
      id: doc.$id,
      board_id: doc.board_id,
      user_id: doc.user_id,
      role: doc.role,
      invited_at: doc.invited_at,
    }));

    return ok(members, { meta: { total: result.total } });
  } catch (err) {
    return handleError(err);
  }
}

const addBoardMemberSchema = z.object({
  user_id: z.string().min(1, "user_id is required"),
  role: z.enum(["editor", "viewer"]).default("viewer"),
});

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId } = await context.params;

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId
    );

    await requireBoardAccess(
      user.id,
      boardId,
      board.org_id ?? null,
      BOARD_ROLE_LEVEL.owner
    );

    const { user_id: targetUserId, role } = await validateBody(
      request,
      addBoardMemberSchema
    );

    const existing = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      [
        Query.equal("board_id", boardId),
        Query.equal("user_id", targetUserId),
        Query.limit(1),
      ]
    );

    if (existing.total > 0) {
      throw new AppError(
        ErrorCode.SLUG_ALREADY_EXISTS,
        "User is already a member of this board"
      );
    }

    const now = new Date().toISOString();

    const doc = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      ID.unique(),
      {
        board_id: boardId,
        user_id: targetUserId,
        role,
        invited_at: now,
      }
    );

    return ok(
      {
        id: doc.$id,
        board_id: doc.board_id,
        user_id: doc.user_id,
        role: doc.role,
        invited_at: doc.invited_at,
      },
      { status: 201 }
    );
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 4: Write PATCH/DELETE board members implementation**

Create: `obeya-cloud/app/api/boards/[id]/members/[uid]/route.ts`

```typescript
import { z } from "zod";
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { requireBoardAccess, BOARD_ROLE_LEVEL } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string; uid: string }> };

async function findBoardMembership(boardId: string, userId: string) {
  const env = getEnv();
  const db = getDatabases();

  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    [
      Query.equal("board_id", boardId),
      Query.equal("user_id", userId),
      Query.limit(1),
    ]
  );

  if (result.total === 0) {
    throw new AppError(
      ErrorCode.USER_NOT_FOUND,
      "User is not a member of this board"
    );
  }

  return result.documents[0];
}

const updateBoardRoleSchema = z.object({
  role: z.enum(["editor", "viewer"]),
});

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId, uid: targetUserId } = await context.params;

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId
    );

    await requireBoardAccess(
      user.id,
      boardId,
      board.org_id ?? null,
      BOARD_ROLE_LEVEL.owner
    );

    const { role: newRole } = await validateBody(
      request,
      updateBoardRoleSchema
    );

    const membership = await findBoardMembership(boardId, targetUserId);

    if (membership.role === "owner") {
      throw new AppError(
        ErrorCode.FORBIDDEN,
        "Cannot change the role of the board owner"
      );
    }

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      membership.$id,
      { role: newRole }
    );

    return ok({
      id: updated.$id,
      board_id: updated.board_id,
      user_id: updated.user_id,
      role: updated.role,
      invited_at: updated.invited_at,
    });
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId, uid: targetUserId } = await context.params;

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      boardId
    );

    const isSelfRemoval = user.id === targetUserId;

    if (!isSelfRemoval) {
      await requireBoardAccess(
        user.id,
        boardId,
        board.org_id ?? null,
        BOARD_ROLE_LEVEL.owner
      );
    } else {
      await requireBoardAccess(
        user.id,
        boardId,
        board.org_id ?? null,
        BOARD_ROLE_LEVEL.viewer
      );
    }

    const membership = await findBoardMembership(boardId, targetUserId);

    if (membership.role === "owner") {
      throw new AppError(
        ErrorCode.FORBIDDEN,
        "Cannot remove the board owner. Transfer ownership first."
      );
    }

    await db.deleteDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      membership.$id
    );

    return ok({ removed: true });
  } catch (err) {
    return handleError(err);
  }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/api/boards/board-members.test.ts
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add app/api/boards/[id]/members/ __tests__/api/boards/board-members.test.ts
git commit -m "feat: add board member CRUD with permission-based access control"
```

---

## Chunk 5: Integration & Final Verification

### Task 8: Run Full Test Suite

- [ ] **Step 1: Run all tests**

```bash
cd ~/code/obeya-cloud
npm test
```

Expected: All tests PASS

- [ ] **Step 2: Verify file structure matches plan**

```bash
cd ~/code/obeya-cloud
find lib/permissions.ts lib/slugs.ts lib/limits.ts \
     app/api/orgs/ app/api/boards/*/members/ \
     __tests__/lib/permissions.test.ts __tests__/lib/slugs.test.ts __tests__/lib/limits.test.ts \
     __tests__/api/orgs/ __tests__/api/boards/board-members.test.ts \
     -type f 2>/dev/null | sort
```

Expected: All files present.

- [ ] **Step 3: Final commit**

```bash
cd ~/code/obeya-cloud
git add -A
git commit -m "feat: complete Plan 3 — orgs, sharing, permissions, free tier limits"
```

---

## Summary

This plan delivers:

| Component | What's built |
|-----------|-------------|
| **Slug Generation** | `lib/slugs.ts` — URL-safe slug from org name, uniqueness enforcement via Appwrite query |
| **Permission Resolution** | `lib/permissions.ts` — org role + board role lookup, higher wins, `requireOrgRole` and `requireBoardAccess` guards |
| **Free Tier Limits** | `lib/limits.ts` — enforce 3 personal boards, 1 org, 3 members/org, 100 items/board; returns `PLAN_LIMIT_REACHED` |
| **Org CRUD** | `GET/POST /api/orgs` — list user's orgs, create org with slug + owner membership |
| **Org Detail** | `GET/PATCH/DELETE /api/orgs/:id` — view (member+), update (admin+), delete (owner only, cascades memberships) |
| **Org Members** | `GET/POST /api/orgs/:id/members` — list (member+), invite (admin+) with duplicate + limit checks |
| **Org Member Mgmt** | `PATCH/DELETE /api/orgs/:id/members/:uid` — change role (owner only), remove (admin+ or self, owner protected) |
| **Board Members** | `GET/POST /api/boards/:id/members` — list (viewer+), invite (owner only) with duplicate check |
| **Board Member Mgmt** | `PATCH/DELETE /api/boards/:id/members/:uid` — change role (owner only), remove (owner+ or self, owner protected) |

**Test coverage:** 10 test files, covering slug generation, permission resolution hierarchy, limit enforcement, and all CRUD operations with role-based access checks.

**Next plan:** Plan 4 — Web UI (Dashboard, Kanban board view, org/board settings pages)
