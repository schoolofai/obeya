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

import { POST } from "@/app/api/plans/[id]/link/route";
import { DELETE } from "@/app/api/plans/[id]/link/[iid]/route";
import { getDatabases } from "@/lib/appwrite/server";

describe("POST /api/plans/:id/link", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("links item IDs to plan", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });
    mockUpdateDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1", "item-2", "item-3"]),
    });

    const request = new Request("http://localhost/api/plans/plan-1/link", {
      method: "POST",
      body: JSON.stringify({ item_ids: ["item-2", "item-3"] }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "plan-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-1", "item-2", "item-3"]),
      })
    );
  });

  it("skips duplicate item IDs when linking", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });
    mockUpdateDoc.mockResolvedValue({ $id: "plan-1" });

    const request = new Request("http://localhost/api/plans/plan-1/link", {
      method: "POST",
      body: JSON.stringify({ item_ids: ["item-1", "item-2"] }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "plan-1" }) });
    expect(response.status).toBe(200);

    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-1", "item-2"]),
      })
    );
  });
});

describe("DELETE /api/plans/:id/link/:iid", () => {
  const mockGetDoc = vi.fn();
  const mockUpdateDoc = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDatabases).mockReturnValue({
      getDocument: mockGetDoc,
      updateDocument: mockUpdateDoc,
    } as any);
  });

  it("removes item ID from linked_items", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1", "item-2"]),
    });
    mockUpdateDoc.mockResolvedValue({ $id: "plan-1" });

    const request = new Request("http://localhost/api/plans/plan-1/link/item-1", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "plan-1", iid: "item-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(mockUpdateDoc).toHaveBeenCalledWith(
      "obeya", "plans", "plan-1",
      expect.objectContaining({
        linked_items: JSON.stringify(["item-2"]),
      })
    );
  });

  it("returns error when item ID not in linked_items", async () => {
    mockGetDoc.mockResolvedValue({
      $id: "plan-1", linked_items: JSON.stringify(["item-1"]),
    });

    const request = new Request("http://localhost/api/plans/plan-1/link/nonexistent", {
      method: "DELETE",
    });

    const response = await DELETE(request, {
      params: Promise.resolve({ id: "plan-1", iid: "nonexistent" }),
    });
    const body = await response.json();

    expect(response.status).toBe(404);
    expect(body.ok).toBe(false);
  });
});
