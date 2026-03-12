import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { BoardCard } from "@/components/dashboard/board-card";
import type { Board } from "@/lib/types";

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

const baseBoard: Board = {
  id: "board-1",
  name: "My Project Board",
  owner_id: "user-1",
  org_id: null,
  display_counter: 10,
  columns: [],
  item_count: 5,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-03-12T10:00:00Z",
};

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-03-12T12:00:00Z"));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("BoardCard", () => {
  it("renders board name", () => {
    render(<BoardCard board={baseBoard} />);
    expect(screen.getByText("My Project Board")).toBeInTheDocument();
  });

  it("renders plural item count", () => {
    render(<BoardCard board={baseBoard} />);
    expect(screen.getByText("5 items")).toBeInTheDocument();
  });

  it("renders singular item count for 1 item", () => {
    render(<BoardCard board={{ ...baseBoard, item_count: 1 }} />);
    expect(screen.getByText("1 item")).toBeInTheDocument();
  });

  it("renders updated time relative to now", () => {
    render(<BoardCard board={baseBoard} />);
    expect(screen.getByText(/Updated 2h ago/)).toBeInTheDocument();
  });

  it("links to /boards/:id", () => {
    render(<BoardCard board={baseBoard} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/boards/board-1");
  });
});
