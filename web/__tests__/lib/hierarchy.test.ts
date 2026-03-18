import { describe, it, expect } from "vitest";
import {
  breadcrumbPath,
  childCount,
  doneCount,
  hasChildren,
  isHiddenByCollapse,
  orderItemsHierarchically,
} from "@/lib/hierarchy";
import type { BoardItem } from "@/lib/api-client";

function makeItem(overrides: Partial<BoardItem>): BoardItem {
  return {
    $id: "id",
    board_id: "board1",
    display_num: 1,
    title: "test",
    type: "task",
    status: "backlog",
    priority: "medium",
    parent_id: null,
    assignee_id: null,
    blocked_by: [],
    tags: [],
    project: null,
    description: "",
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

const testItems: Record<string, BoardItem> = {
  "epic-1": makeItem({
    $id: "epic-1",
    display_num: 1,
    title: "Auth Rewrite",
    type: "epic",
    status: "backlog",
  }),
  "story-4": makeItem({
    $id: "story-4",
    display_num: 4,
    title: "Session Mgmt",
    type: "story",
    status: "backlog",
    parent_id: "epic-1",
  }),
  "task-2": makeItem({
    $id: "task-2",
    display_num: 2,
    title: "Refactor",
    type: "task",
    status: "backlog",
    parent_id: "story-4",
  }),
  "task-3": makeItem({
    $id: "task-3",
    display_num: 3,
    title: "JWT",
    type: "task",
    status: "in-progress",
    parent_id: "story-4",
  }),
  "task-6": makeItem({
    $id: "task-6",
    display_num: 6,
    title: "Update store",
    type: "task",
    status: "done",
    parent_id: "story-4",
  }),
};

describe("breadcrumbPath", () => {
  it("builds ancestry path for nested task", () => {
    expect(breadcrumbPath(testItems, testItems["task-2"])).toBe(
      "Auth Rewrite › Session Mgmt"
    );
  });

  it("builds single-level path for story under epic", () => {
    expect(breadcrumbPath(testItems, testItems["story-4"])).toBe(
      "Auth Rewrite"
    );
  });

  it("returns empty string for root item", () => {
    expect(breadcrumbPath(testItems, testItems["epic-1"])).toBe("");
  });

  it("handles cycle without infinite loop", () => {
    const cycleItems: Record<string, BoardItem> = {
      a: makeItem({ $id: "a", title: "A", parent_id: "b" }),
      b: makeItem({ $id: "b", title: "B", parent_id: "a" }),
    };
    // Should not hang; result is not empty because B is a's parent
    const result = breadcrumbPath(cycleItems, cycleItems["a"]);
    expect(result).toBe("B");
  });
});

describe("childCount", () => {
  it("counts all descendants of epic", () => {
    expect(childCount(testItems, "epic-1")).toBe(4);
  });

  it("counts direct+transitive children of story", () => {
    expect(childCount(testItems, "story-4")).toBe(3);
  });

  it("returns 0 for leaf task", () => {
    expect(childCount(testItems, "task-2")).toBe(0);
  });
});

describe("doneCount", () => {
  it("counts done descendants of epic", () => {
    expect(doneCount(testItems, "epic-1")).toBe(1);
  });

  it("returns 0 when no descendants are done", () => {
    expect(doneCount(testItems, "task-2")).toBe(0);
  });
});

describe("hasChildren", () => {
  it("returns true for epic with children", () => {
    expect(hasChildren(testItems, "epic-1")).toBe(true);
  });

  it("returns true for story with children", () => {
    expect(hasChildren(testItems, "story-4")).toBe(true);
  });

  it("returns false for leaf task", () => {
    expect(hasChildren(testItems, "task-2")).toBe(false);
  });
});

describe("isHiddenByCollapse", () => {
  it("hides same-column child when parent is collapsed", () => {
    expect(
      isHiddenByCollapse(testItems, testItems["story-4"], { "epic-1": true })
    ).toBe(true);
  });

  it("hides grandchild in same column", () => {
    expect(
      isHiddenByCollapse(testItems, testItems["task-2"], { "epic-1": true })
    ).toBe(true);
  });

  it("does NOT hide cross-column child", () => {
    expect(
      isHiddenByCollapse(testItems, testItems["task-3"], { "epic-1": true })
    ).toBe(false);
  });

  it("does NOT hide parent when child is collapsed", () => {
    expect(
      isHiddenByCollapse(testItems, testItems["epic-1"], { "story-4": true })
    ).toBe(false);
  });

  it("handles cycle without infinite loop", () => {
    const cycleItems: Record<string, BoardItem> = {
      a: makeItem({ $id: "a", status: "backlog", parent_id: "b" }),
      b: makeItem({ $id: "b", status: "backlog", parent_id: "a" }),
    };
    // Should not hang
    isHiddenByCollapse(cycleItems, cycleItems["b"], { a: true });
  });
});

describe("orderItemsHierarchically", () => {
  it("orders parents before children", () => {
    const items = [
      testItems["task-2"],
      testItems["story-4"],
      testItems["epic-1"],
    ];
    const ordered = orderItemsHierarchically(testItems, items);
    expect(ordered.map((i) => i.$id)).toEqual([
      "epic-1",
      "story-4",
      "task-2",
    ]);
  });

  it("handles orphan items as roots", () => {
    const orphan = makeItem({
      $id: "task-20",
      display_num: 20,
      title: "Fix README",
      status: "backlog",
    });
    const items = [orphan, testItems["epic-1"]];
    const allWithOrphan = { ...testItems, "task-20": orphan };
    const ordered = orderItemsHierarchically(allWithOrphan, items);
    expect(ordered.map((i) => i.$id)).toEqual(["epic-1", "task-20"]);
  });

  it("sorts multiple root trees by display_num", () => {
    const epic10 = makeItem({
      $id: "epic-10",
      display_num: 10,
      title: "Rate Limiting",
      type: "epic",
      status: "backlog",
    });
    const task11 = makeItem({
      $id: "task-11",
      display_num: 11,
      title: "Throttle",
      status: "backlog",
      parent_id: "epic-10",
    });
    const allItems = { ...testItems, "epic-10": epic10, "task-11": task11 };
    const items = [
      task11,
      epic10,
      testItems["task-2"],
      testItems["story-4"],
      testItems["epic-1"],
    ];
    const ordered = orderItemsHierarchically(allItems, items);
    expect(ordered.map((i) => i.$id)).toEqual([
      "epic-1",
      "story-4",
      "task-2",
      "epic-10",
      "task-11",
    ]);
  });
});
