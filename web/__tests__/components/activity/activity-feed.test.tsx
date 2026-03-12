import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ActivityFeed } from "@/components/activity/activity-feed";

const mockEntries = [
  {
    $id: "h1",
    item_id: "item1",
    board_id: "board1",
    user_id: "user1",
    action: "created",
    detail: "Created task #1: Fix login bug",
    timestamp: "2026-03-11T10:00:00Z",
  },
  {
    $id: "h2",
    item_id: "item1",
    board_id: "board1",
    user_id: "user2",
    action: "moved",
    detail: "status: todo -> in-progress",
    timestamp: "2026-03-11T11:30:00Z",
  },
];

describe("ActivityFeed", () => {
  it("renders activity entries", () => {
    render(<ActivityFeed entries={mockEntries} />);
    expect(screen.getByText("Created task #1: Fix login bug")).toBeDefined();
    expect(screen.getByText("status: todo -> in-progress")).toBeDefined();
  });

  it("shows action type badges", () => {
    render(<ActivityFeed entries={mockEntries} />);
    expect(screen.getByText("created")).toBeDefined();
    expect(screen.getByText("moved")).toBeDefined();
  });

  it("renders empty state when no entries", () => {
    render(<ActivityFeed entries={[]} />);
    expect(screen.getByText("No activity yet.")).toBeDefined();
  });
});
