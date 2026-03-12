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

import { GET } from "@/app/api/items/[id]/history/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/items/:id/history", () => {
  const mockListDocs = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      listDocuments: mockListDocs,
    } as any);
  });

  it("returns history entries sorted by timestamp desc", async () => {
    mockListDocs.mockResolvedValue({
      total: 2,
      documents: [
        { $id: "h2", action: "moved", timestamp: "2026-03-12T11:00:00Z" },
        { $id: "h1", action: "created", timestamp: "2026-03-12T10:00:00Z" },
      ],
    });

    const request = new Request("http://localhost/api/items/item-1/history");
    const response = await GET(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(2);
    expect(body.meta.total).toBe(2);
  });

  it("returns empty array when no history exists", async () => {
    mockListDocs.mockResolvedValue({ total: 0, documents: [] });

    const request = new Request("http://localhost/api/items/item-1/history");
    const response = await GET(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toEqual([]);
    expect(body.meta.total).toBe(0);
  });
});
