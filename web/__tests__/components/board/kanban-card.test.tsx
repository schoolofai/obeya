import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KanbanCard } from "@/components/board/kanban-card";

const mockItem = {
  $id: "item1",
  display_num: 42,
  type: "task" as const,
  title: "Fix the login bug",
  priority: "high" as const,
  assignee_id: "user1",
  status: "in-progress",
  blocked_by: [],
  board_id: "board1",
  description: "",
  parent_id: null,
  tags: [],
  project: null,
  created_at: "2026-03-10T10:00:00Z",
  updated_at: "2026-03-11T14:00:00Z",
};

describe("KanbanCard", () => {
  it("renders display number and title", () => {
    render(<KanbanCard item={mockItem} onClick={() => {}} />);
    expect(screen.getByText("#42")).toBeDefined();
    expect(screen.getByText("Fix the login bug")).toBeDefined();
  });

  it("shows priority badge", () => {
    render(<KanbanCard item={mockItem} onClick={() => {}} />);
    expect(screen.getByText("high")).toBeDefined();
  });

  it("shows type icon text", () => {
    render(<KanbanCard item={mockItem} onClick={() => {}} />);
    expect(screen.getByTestId("type-icon")).toBeDefined();
  });
});
