import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { RealtimeActivityFeed } from "@/components/activity/realtime-activity-feed";
import type { HistoryEntry } from "@/hooks/use-activity-subscription";

vi.mock("@/hooks/use-activity-subscription", () => ({
  useActivitySubscription: vi.fn().mockReturnValue({ status: "connected" }),
}));

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: vi.fn().mockReturnValue(vi.fn()),
  })),
}));

describe("RealtimeActivityFeed", () => {
  const mockEntries: HistoryEntry[] = [
    {
      $id: "hist-1",
      item_id: "item-1",
      board_id: "board-123",
      user_id: "user-1",
      action: "created",
      detail: "Created task #42 'Build login page'",
      timestamp: "2026-03-12T10:00:00Z",
    },
    {
      $id: "hist-2",
      item_id: "item-2",
      board_id: "board-123",
      user_id: "user-2",
      action: "moved",
      detail: "status: todo -> in-progress",
      timestamp: "2026-03-12T10:05:00Z",
    },
    {
      $id: "hist-3",
      item_id: "item-1",
      board_id: "board-123",
      user_id: "user-1",
      action: "edited",
      detail: "Updated description",
      timestamp: "2026-03-12T10:10:00Z",
    },
  ];

  it("renders a list of activity entries", () => {
    const { container } = render(
      <RealtimeActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={mockEntries}
      />
    );
    expect(container.textContent).toContain("created");
    expect(container.textContent).toContain("moved");
    expect(container.textContent).toContain("edited");
  });

  it("shows entries in reverse chronological order (newest first)", () => {
    const { container } = render(
      <RealtimeActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={mockEntries}
      />
    );
    const items = container.querySelectorAll("[data-activity-entry]");
    expect(items.length).toBe(3);
    expect(items[0].textContent).toContain("edited");
  });

  it("renders empty state when no entries", () => {
    const { container } = render(
      <RealtimeActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={[]}
      />
    );
    expect(container.textContent).toContain("No activity");
  });

  it("displays the action type with an appropriate label", () => {
    const { container } = render(
      <RealtimeActivityFeed
        boardId="board-123"
        databaseId="obeya"
        initialEntries={[mockEntries[1]]}
      />
    );
    expect(container.textContent).toContain("moved");
    expect(container.textContent).toContain("status: todo -> in-progress");
  });
});
