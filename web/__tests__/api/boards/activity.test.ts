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
  Query: {
    equal: vi.fn((field: string, value: string) => `${field}=${value}`),
    orderDesc: vi.fn((field: string) => `orderDesc=${field}`),
    limit: vi.fn((n: number) => `limit=${n}`),
    offset: vi.fn((n: number) => `offset=${n}`),
  },
}));

import { GET } from "@/app/api/boards/[id]/activity/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards/:id/activity", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns board-wide activity feed paginated", async () => {
    mockListDocs.mockResolvedValue({
      total: 50,
      documents: [
        { $id: "h3", item_id: "item-2", action: "assigned", timestamp: "2026-03-12T12:00:00Z" },
        { $id: "h2", item_id: "item-1", action: "moved", timestamp: "2026-03-12T11:00:00Z" },
      ],
    });

    const request = new Request("http://localhost/api/boards/board-1/activity?limit=2&offset=0");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.meta.total).toBe(50);
    expect(body.meta.limit).toBe(2);
  });

  it("uses default pagination when no query params", async () => {
    mockListDocs.mockResolvedValue({ total: 0, documents: [] });

    const request = new Request("http://localhost/api/boards/board-1/activity");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.meta.limit).toBe(25);
    expect(body.meta.offset).toBe(0);
  });
});
