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
  $id: "board-1", name: "My Board", owner_id: "user-1", org_id: null,
  display_counter: 5,
  columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
  display_map: '{}', users: '{}', projects: '{}',
  agent_role: "worker", version: 1,
  created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
};

describe("GET /api/boards/:id", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns board with deserialized fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const request = new Request("http://localhost/api/boards/board-1");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
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
  beforeEach(() => { vi.clearAllMocks(); });

  it("updates board name and returns updated board", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const updatedDoc = { ...boardDoc, name: "Renamed Board" };
    const mockDb = { updateDocument: vi.fn().mockResolvedValue(updatedDoc) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Renamed Board" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.name).toBe("Renamed Board");
  });

  it("returns 200 for empty update body (no-op)", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const mockDb = { updateDocument: vi.fn().mockResolvedValue(boardDoc) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1", {
      method: "PATCH",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });

    const response = await PATCH(request, { params: Promise.resolve({ id: "board-1" }) });
    expect(response.status).toBe(200);
  });
});

describe("DELETE /api/boards/:id", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("deletes board and returns confirmation", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const mockDb = { deleteDocument: vi.fn().mockResolvedValue(undefined) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1", { method: "DELETE" });
    const response = await DELETE(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.deleted).toBe(true);
  });
});
