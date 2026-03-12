import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Avatar } from "@/components/ui/avatar";

describe("Avatar initials", () => {
  it("renders initials from name", () => {
    render(<Avatar name="John Doe" />);
    expect(screen.getByText("JD")).toBeInTheDocument();
  });

  it("renders single initial for single-word name", () => {
    render(<Avatar name="Alice" />);
    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("renders first+last initials ignoring middle names", () => {
    render(<Avatar name="John Michael Doe" />);
    expect(screen.getByText("JD")).toBeInTheDocument();
  });
});

describe("Avatar with image", () => {
  it("renders img with alt=name when src is provided", () => {
    render(<Avatar name="John Doe" src="/avatar.jpg" />);
    const img = screen.getByRole("img", { name: "John Doe" });
    expect(img).toHaveAttribute("src", "/avatar.jpg");
    expect(img).toHaveAttribute("alt", "John Doe");
  });
});

describe("Avatar sizes", () => {
  it("renders sm size", () => {
    const { container } = render(<Avatar name="JD" size="sm" />);
    expect(container.firstChild).toHaveClass("h-8");
  });

  it("renders md size by default", () => {
    const { container } = render(<Avatar name="JD" />);
    expect(container.firstChild).toHaveClass("h-10");
  });

  it("renders lg size", () => {
    const { container } = render(<Avatar name="JD" size="lg" />);
    expect(container.firstChild).toHaveClass("h-12");
  });
});
