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

import { GET } from "@/app/api/plans/[id]/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/plans/:id", () => {
  const mockGetDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
    } as any);
  });

  it("returns plan with linked items resolved", async () => {
    mockGetDoc
      .mockResolvedValueOnce({
        $id: "plan-1",
        title: "Release Plan",
        board_id: "board-1",
        linked_items: JSON.stringify(["item-1", "item-2"]),
        content: "# Plan",
      })
      .mockResolvedValueOnce({
        $id: "item-1", title: "Task A", status: "done",
      })
      .mockResolvedValueOnce({
        $id: "item-2", title: "Task B", status: "in-progress",
      });

    const request = new Request("http://localhost/api/plans/plan-1");
    const response = await GET(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.plan.$id).toBe("plan-1");
    expect(body.data.linked_items).toHaveLength(2);
    expect(body.data.linked_items[0].title).toBe("Task A");
  });

  it("returns plan with empty linked items when none linked", async () => {
    mockGetDoc.mockResolvedValueOnce({
      $id: "plan-1",
      title: "Empty Plan",
      board_id: "board-1",
      linked_items: "[]",
      content: "# Empty",
    });

    const request = new Request("http://localhost/api/plans/plan-1");
    const response = await GET(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.linked_items).toEqual([]);
  });
});
