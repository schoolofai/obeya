import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { BoardList } from "@/components/dashboard/board-list";
import type { Board, Org } from "@/lib/types";

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: React.PropsWithChildren<{ href: string; [key: string]: unknown }>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

const personalBoard: Board = {
  id: "board-1",
  name: "Personal Board",
  owner_id: "user-1",
  org_id: null,
  display_counter: 5,
  columns: [],
  item_count: 3,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const org: Org = {
  id: "org-1",
  name: "My Company",
  slug: "my-company",
  owner_id: "user-1",
  plan: "pro",
  created_at: "2026-01-01T00:00:00Z",
};

const orgBoard: Board = {
  id: "board-2",
  name: "Team Board",
  owner_id: "user-1",
  org_id: "org-1",
  display_counter: 8,
  columns: [],
  item_count: 10,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("BoardList", () => {
  it("shows empty state when no boards", () => {
    render(<BoardList boards={[]} orgs={[]} />);
    expect(screen.getByText("No boards yet.")).toBeInTheDocument();
  });

  it("renders personal section for boards without org", () => {
    render(<BoardList boards={[personalBoard]} orgs={[]} />);
    expect(screen.getByText("Personal")).toBeInTheDocument();
    expect(screen.getByText("Personal Board")).toBeInTheDocument();
  });

  it("renders org section for boards with org_id", () => {
    render(<BoardList boards={[orgBoard]} orgs={[org]} />);
    expect(screen.getByText("My Company")).toBeInTheDocument();
    expect(screen.getByText("Team Board")).toBeInTheDocument();
  });

  it("renders both personal and org sections together", () => {
    render(<BoardList boards={[personalBoard, orgBoard]} orgs={[org]} />);
    expect(screen.getByText("Personal")).toBeInTheDocument();
    expect(screen.getByText("My Company")).toBeInTheDocument();
    expect(screen.getByText("Personal Board")).toBeInTheDocument();
    expect(screen.getByText("Team Board")).toBeInTheDocument();
  });
});
