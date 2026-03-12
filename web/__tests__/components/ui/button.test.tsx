import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "@/components/ui/button";

describe("Button variants", () => {
  it("renders primary variant with correct class", () => {
    render(<Button variant="primary">Click me</Button>);
    const btn = screen.getByRole("button", { name: "Click me" });
    expect(btn).toHaveClass("bg-blue-600");
  });

  it("renders secondary variant", () => {
    render(<Button variant="secondary">Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-gray-100");
  });

  it("renders ghost variant", () => {
    render(<Button variant="ghost">Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-transparent");
  });

  it("renders danger variant", () => {
    render(<Button variant="danger">Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-red-600");
  });
});

describe("Button sizes", () => {
  it("renders sm size", () => {
    render(<Button size="sm">Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("text-sm");
  });

  it("renders md size by default", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("text-sm");
  });

  it("renders lg size", () => {
    render(<Button size="lg">Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("text-base");
  });
});

describe("Button props", () => {
  it("renders full width", () => {
    render(<Button fullWidth>Click me</Button>);
    expect(screen.getByRole("button")).toHaveClass("w-full");
  });

  it("is disabled when disabled prop is set", () => {
    render(<Button disabled>Click me</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("calls onClick handler", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click me</Button>);
    await user.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
