import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ItemDetailPanel } from "@/components/board/item-detail-panel";

const mockItem = {
  $id: "item1",
  board_id: "board1",
  display_num: 7,
  type: "story" as const,
  title: "User login flow",
  description: "Implement the full login flow with OAuth.",
  status: "in-progress",
  priority: "high" as const,
  parent_id: null,
  assignee_id: "user1",
  blocked_by: ["item2"],
  tags: ["auth"],
  project: null,
  created_at: "2026-03-10T10:00:00Z",
  updated_at: "2026-03-11T14:00:00Z",
};

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true, data: [] }),
    })
  );
});

describe("ItemDetailPanel", () => {
  it("renders item title and description", () => {
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={() => {}}
        onUpdate={() => {}}
      />
    );
    expect(screen.getByText("User login flow")).toBeDefined();
    expect(
      screen.getByText("Implement the full login flow with OAuth.")
    ).toBeDefined();
  });

  it("shows blocked status when blocked_by is non-empty", () => {
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={() => {}}
        onUpdate={() => {}}
      />
    );
    expect(screen.getByText("Blocked by")).toBeDefined();
  });

  it("renders close button", () => {
    const onClose = vi.fn();
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={onClose}
        onUpdate={() => {}}
      />
    );
    expect(screen.getByLabelText("Close panel")).toBeDefined();
  });
});
