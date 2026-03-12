import { describe, it, expect } from "vitest";
import { serializeItem, deserializeItem } from "@/lib/items/serialize";

describe("serializeItem", () => {
  it("converts blocked_by and tags arrays to JSON strings", () => {
    const item = { blocked_by: ["item-1", "item-2"], tags: ["bug", "urgent"], title: "Fix login" };
    const result = serializeItem(item);
    expect(result.blocked_by).toBe('["item-1","item-2"]');
    expect(result.tags).toBe('["bug","urgent"]');
    expect(result.title).toBe("Fix login");
  });

  it("handles empty arrays", () => {
    const item = { blocked_by: [], tags: [] };
    const result = serializeItem(item);
    expect(result.blocked_by).toBe("[]");
    expect(result.tags).toBe("[]");
  });
});

describe("deserializeItem", () => {
  it("parses Appwrite document to item shape", () => {
    const doc = {
      $id: "item-1", board_id: "board-1", display_num: 3, type: "task",
      title: "Fix login", description: "Users can't log in",
      status: "in-progress", priority: "high", parent_id: null,
      assignee_id: "user-1", blocked_by: '["item-2"]', tags: '["bug","auth"]',
      project: "web", created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeItem(doc);
    expect(result.id).toBe("item-1");
    expect(result.blocked_by).toEqual(["item-2"]);
    expect(result.tags).toEqual(["bug", "auth"]);
    expect(result.display_num).toBe(3);
  });

  it("handles empty/null JSON strings gracefully", () => {
    const doc = {
      $id: "item-2", board_id: "board-1", display_num: 1, type: "task",
      title: "Test", description: "", status: "todo", priority: "medium",
      parent_id: null, assignee_id: null, blocked_by: "", tags: "",
      project: null, created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeItem(doc);
    expect(result.blocked_by).toEqual([]);
    expect(result.tags).toEqual([]);
  });
});
