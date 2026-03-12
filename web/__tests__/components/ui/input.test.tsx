import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Input } from "@/components/ui/input";

describe("Input", () => {
  it("renders with label", () => {
    render(<Input label="Email" name="email" />);
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("associates label with input via htmlFor/id", () => {
    render(<Input label="Email" name="email" />);
    const input = screen.getByLabelText("Email");
    expect(input).toHaveAttribute("id", "email");
  });

  it("renders placeholder", () => {
    render(<Input label="Email" name="email" placeholder="Enter email" />);
    expect(screen.getByPlaceholderText("Enter email")).toBeInTheDocument();
  });

  it("renders error message", () => {
    render(<Input label="Email" name="email" error="Invalid email" />);
    expect(screen.getByText("Invalid email")).toBeInTheDocument();
  });

  it("renders password type", () => {
    render(<Input label="Password" name="password" type="password" />);
    expect(screen.getByLabelText("Password")).toHaveAttribute(
      "type",
      "password"
    );
  });

  it("renders text type by default", () => {
    render(<Input label="Name" name="name" />);
    expect(screen.getByLabelText("Name")).toHaveAttribute("type", "text");
  });
});
