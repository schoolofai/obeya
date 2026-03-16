import { describe, it, expect } from "vitest";
import { serializeItem, deserializeItem } from "@/lib/items/serialize";

describe("serializeItem — review fields", () => {
  it("serializes review_context as JSON string", () => {
    const result = serializeItem({
      review_context: {
        purpose: "Fix auth",
        files_changed: [{ path: "auth.go", added: 10, removed: 5 }],
      },
    });
    expect(typeof result.review_context).toBe("string");
    const parsed = JSON.parse(result.review_context as string);
    expect(parsed.purpose).toBe("Fix auth");
    expect(parsed.files_changed[0].path).toBe("auth.go");
  });

  it("serializes human_review as JSON string", () => {
    const result = serializeItem({
      human_review: { status: "reviewed", reviewed_by: "niladri" },
    });
    expect(typeof result.human_review).toBe("string");
    const parsed = JSON.parse(result.human_review as string);
    expect(parsed.status).toBe("reviewed");
  });

  it("skips null review fields (does not store 'null' string)", () => {
    const result = serializeItem({
      review_context: null,
      human_review: null,
    });
    expect(result.review_context).toBeNull();
    expect(result.human_review).toBeNull();
  });
});

describe("deserializeItem — review fields", () => {
  const baseDoc = {
    $id: "item-1",
    board_id: "board-1",
    display_num: 1,
    type: "task",
    title: "Test",
    description: "",
    status: "done",
    priority: "medium",
    parent_id: null,
    assignee_id: null,
    blocked_by: "[]",
    tags: "[]",
    project: null,
    created_at: "2026-03-10T00:00:00Z",
    updated_at: "2026-03-10T00:00:00Z",
  };

  it("deserializes review_context from JSON string", () => {
    const doc = {
      ...baseDoc,
      review_context: JSON.stringify({ purpose: "Fix auth", files_changed: [] }),
    };
    const item = deserializeItem(doc);
    expect(item.review_context).not.toBeNull();
    expect(item.review_context!.purpose).toBe("Fix auth");
  });

  it("deserializes human_review from JSON string", () => {
    const doc = {
      ...baseDoc,
      human_review: JSON.stringify({ status: "reviewed", reviewed_by: "niladri" }),
    };
    const item = deserializeItem(doc);
    expect(item.human_review).not.toBeNull();
    expect(item.human_review!.status).toBe("reviewed");
  });

  it("returns null for missing review fields", () => {
    const item = deserializeItem(baseDoc);
    expect(item.review_context).toBeNull();
    expect(item.human_review).toBeNull();
    expect(item.confidence).toBeNull();
  });

  it("deserializes confidence as number", () => {
    const doc = { ...baseDoc, confidence: 75 };
    const item = deserializeItem(doc);
    expect(item.confidence).toBe(75);
  });

  it("deserializes sponsor", () => {
    const doc = { ...baseDoc, sponsor: "niladri" };
    const item = deserializeItem(doc);
    expect(item.sponsor).toBe("niladri");
  });

  it("handles review_context already parsed as object", () => {
    const doc = {
      ...baseDoc,
      review_context: { purpose: "Already parsed" },
    };
    const item = deserializeItem(doc);
    expect(item.review_context!.purpose).toBe("Already parsed");
  });
});
