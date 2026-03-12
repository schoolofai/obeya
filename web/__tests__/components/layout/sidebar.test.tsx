import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Sidebar } from "@/components/layout/sidebar";

vi.mock("next/navigation", () => ({
  usePathname: () => "/dashboard",
}));

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: React.PropsWithChildren<{ href: string; [key: string]: unknown }>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("Sidebar", () => {
  it("renders Obeya logo/brand", () => {
    render(<Sidebar />);
    expect(screen.getByText("Obeya")).toBeInTheDocument();
  });

  it("renders Dashboard link", () => {
    render(<Sidebar />);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("renders Settings link", () => {
    render(<Sidebar />);
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("Dashboard link points to /dashboard", () => {
    render(<Sidebar />);
    const link = screen.getByRole("link", { name: /dashboard/i });
    expect(link).toHaveAttribute("href", "/dashboard");
  });

  it("Settings link points to /settings", () => {
    render(<Sidebar />);
    const link = screen.getByRole("link", { name: /settings/i });
    expect(link).toHaveAttribute("href", "/settings");
  });

  it("marks active link for current pathname", () => {
    render(<Sidebar />);
    const dashboardLink = screen.getByRole("link", { name: /dashboard/i });
    expect(dashboardLink).toHaveClass("bg-gray-100");
  });
});
