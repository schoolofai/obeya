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

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));

vi.mock("node-appwrite", () => ({
  ID: { unique: vi.fn().mockReturnValue("plan-1") },
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    orderDesc: vi.fn((field: string) => `orderDesc=${field}`),
    limit: vi.fn((n: number) => `limit=${n}`),
    offset: vi.fn((n: number) => `offset=${n}`),
  },
}));

import { GET, POST } from "@/app/api/boards/[id]/plans/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards/:id/plans", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns list of plans for a board", async () => {
    mockListDocs.mockResolvedValue({
      total: 1,
      documents: [
        { $id: "plan-1", title: "Sprint Plan", display_num: 5, board_id: "board-1" },
      ],
    });

    const request = new Request("http://localhost/api/boards/board-1/plans");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].title).toBe("Sprint Plan");
  });
});

describe("POST /api/boards/:id/plans", () => {
  const mockGetDoc = vi.fn();
  const mockCreateDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      createDocument: mockCreateDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("creates plan with incremented display counter", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "board-1", display_counter: 10,
    });
    mockUpdateDoc.mockResolvedValue({ $id: "board-1", display_counter: 11 });
    mockCreateDoc.mockResolvedValue({
      $id: "plan-1", title: "Release Plan", display_num: 11, board_id: "board-1",
    });

    const request = new Request("http://localhost/api/boards/board-1/plans", {
      method: "POST",
      body: JSON.stringify({
        title: "Release Plan",
        source_path: "docs/plans/release.md",
        content: "# Release Plan\n\nSteps...",
      }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "boards", "board-1",
      expect.objectContaining({ display_counter: 11 })
    );
    expect(mockCreateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        title: "Release Plan",
        board_id: "board-1",
        display_num: 11,
        linked_items: "[]",
      })
    );
  });

  it("rejects plan creation with missing title", async () => {
    const request = new Request("http://localhost/api/boards/board-1/plans", {
      method: "POST",
      body: JSON.stringify({ content: "# no title" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
  });
});
