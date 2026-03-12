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

import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";

describe("incrementDisplayCounter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads current counter, increments, and returns the new value", async () => {
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        display_counter: 5,
      }),
      updateDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        display_counter: 6,
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await incrementDisplayCounter("board-1");

    expect(result).toBe(6);
    expect(mockDb.getDocument).toHaveBeenCalledTimes(1);
    expect(mockDb.updateDocument).toHaveBeenCalledWith(
      "obeya",
      "boards",
      "board-1",
      { display_counter: 6 }
    );
  });

  it("retries on conflict (409) and succeeds", async () => {
    const conflict = new Error("Conflict");
    (conflict as any).code = 409;

    const mockDb = {
      getDocument: vi
        .fn()
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 5 })
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 6 }),
      updateDocument: vi
        .fn()
        .mockRejectedValueOnce(conflict)
        .mockResolvedValueOnce({ $id: "board-1", display_counter: 7 }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await incrementDisplayCounter("board-1");

    expect(result).toBe(7);
    expect(mockDb.getDocument).toHaveBeenCalledTimes(2);
    expect(mockDb.updateDocument).toHaveBeenCalledTimes(2);
  });

  it("throws COUNTER_CONFLICT after max retries exhausted", async () => {
    const conflict = new Error("Conflict");
    (conflict as any).code = 409;

    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({ $id: "board-1", display_counter: 5 }),
      updateDocument: vi.fn().mockRejectedValue(conflict),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(incrementDisplayCounter("board-1")).rejects.toThrow(
      "Failed to increment display counter after 3 retries"
    );
  });
});
