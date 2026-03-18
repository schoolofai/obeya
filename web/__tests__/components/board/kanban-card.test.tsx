import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { KanbanCard } from "@/components/board/kanban-card";
import type { BoardItem } from "@/lib/api-client";

function makeItem(overrides: Partial<BoardItem> = {}): BoardItem {
  return {
    $id: "item1",
    board_id: "board1",
    display_num: 42,
    type: "task",
    title: "Fix the login bug",
    priority: "high",
    assignee_id: "user1",
    status: "in-progress",
    blocked_by: [],
    description: "",
    parent_id: null,
    tags: [],
    project: null,
    created_at: "2026-03-10T10:00:00Z",
    updated_at: "2026-03-11T14:00:00Z",
    ...overrides,
  };
}

const defaultProps = {
  allItems: {} as Record<string, BoardItem>,
  collapsed: {},
  onToggleCollapse: () => {},
  onClick: () => {},
};

describe("KanbanCard", () => {
  it("renders display number and title", () => {
    const item = makeItem();
    render(<KanbanCard item={item} {...defaultProps} />);
    expect(screen.getByText("#42")).toBeDefined();
    expect(screen.getByText("Fix the login bug")).toBeDefined();
  });

  it("shows priority badge", () => {
    const item = makeItem();
    render(<KanbanCard item={item} {...defaultProps} />);
    expect(screen.getByText("high")).toBeDefined();
  });

  it("shows type icon text", () => {
    const item = makeItem();
    render(<KanbanCard item={item} {...defaultProps} />);
    expect(screen.getByTestId("type-icon")).toBeDefined();
  });

  it("renders breadcrumb when item has parent", () => {
    const epic = makeItem({
      $id: "epic-1",
      title: "Auth Rewrite",
      type: "epic",
      status: "backlog",
    });
    const task = makeItem({
      $id: "task-1",
      title: "Fix bug",
      parent_id: "epic-1",
      status: "backlog",
    });
    const allItems = { "epic-1": epic, "task-1": task };
    render(
      <KanbanCard
        item={task}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    expect(screen.getByTestId("breadcrumb").textContent).toBe("Auth Rewrite");
  });

  it("does not render breadcrumb for root item", () => {
    const epic = makeItem({
      $id: "epic-1",
      title: "Auth Rewrite",
      type: "epic",
    });
    const allItems = { "epic-1": epic };
    render(
      <KanbanCard
        item={epic}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    expect(screen.queryByTestId("breadcrumb")).toBeNull();
  });

  it("shows collapse indicator and child badge for parent items", () => {
    const epic = makeItem({
      $id: "epic-1",
      title: "Auth Rewrite",
      type: "epic",
      status: "backlog",
    });
    const child = makeItem({
      $id: "task-1",
      title: "Child",
      parent_id: "epic-1",
      status: "backlog",
    });
    const allItems = { "epic-1": epic, "task-1": child };
    render(
      <KanbanCard
        item={epic}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    expect(screen.getByTestId("collapse-indicator").textContent).toBe("▼");
    expect(screen.getByTestId("child-badge").textContent).toBe("1 item");
    expect(screen.getByTestId("progress-indicator").textContent).toBe(
      "0/1 done"
    );
  });

  it("shows collapsed indicator when collapsed", () => {
    const epic = makeItem({
      $id: "epic-1",
      title: "Auth Rewrite",
      type: "epic",
    });
    const child = makeItem({
      $id: "task-1",
      parent_id: "epic-1",
    });
    const allItems = { "epic-1": epic, "task-1": child };
    render(
      <KanbanCard
        item={epic}
        allItems={allItems}
        collapsed={{ "epic-1": true }}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    expect(screen.getByTestId("collapse-indicator").textContent).toBe("▶");
  });

  it("calls onToggleCollapse when collapse indicator is clicked", () => {
    let toggledId = "";
    const epic = makeItem({
      $id: "epic-1",
      title: "Auth Rewrite",
      type: "epic",
    });
    const child = makeItem({
      $id: "task-1",
      parent_id: "epic-1",
    });
    const allItems = { "epic-1": epic, "task-1": child };
    render(
      <KanbanCard
        item={epic}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={(id) => {
          toggledId = id;
        }}
        onClick={() => {}}
      />
    );
    fireEvent.click(screen.getByTestId("collapse-indicator"));
    expect(toggledId).toBe("epic-1");
  });

  it("does not show collapse indicator for leaf items", () => {
    const task = makeItem({ $id: "task-1" });
    const allItems = { "task-1": task };
    render(
      <KanbanCard
        item={task}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    expect(screen.queryByTestId("collapse-indicator")).toBeNull();
    expect(screen.queryByTestId("child-badge")).toBeNull();
  });

  it("applies left border class for epics", () => {
    const epic = makeItem({
      $id: "epic-1",
      type: "epic",
    });
    const allItems = { "epic-1": epic };
    const { container } = render(
      <KanbanCard
        item={epic}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    const button = container.querySelector("button");
    expect(button?.className).toContain("border-l-fuchsia-500");
  });

  it("applies left border class for stories", () => {
    const story = makeItem({
      $id: "story-1",
      type: "story",
    });
    const allItems = { "story-1": story };
    const { container } = render(
      <KanbanCard
        item={story}
        allItems={allItems}
        collapsed={{}}
        onToggleCollapse={() => {}}
        onClick={() => {}}
      />
    );
    const button = container.querySelector("button");
    expect(button?.className).toContain("border-l-blue-500");
  });
});
