import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Header } from "@/components/layout/header";
import type { User } from "@/lib/types";

const mockUser: User = {
  id: "user-1",
  email: "alice@example.com",
  name: "Alice Smith",
};

describe("Header", () => {
  it("renders user name when user is provided", () => {
    render(<Header user={mockUser} />);
    expect(screen.getByText("Alice Smith")).toBeInTheDocument();
  });

  it("renders avatar initials from user name", () => {
    render(<Header user={mockUser} />);
    expect(screen.getByText("AS")).toBeInTheDocument();
  });

  it("renders nothing problematic when user is null", () => {
    const { container } = render(<Header user={null} />);
    expect(container).toBeTruthy();
  });
});
