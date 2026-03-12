import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Badge } from "@/components/ui/badge";

describe("Badge variants", () => {
  it("renders default variant", () => {
    render(<Badge>Default</Badge>);
    expect(screen.getByText("Default")).toHaveClass("bg-gray-100");
  });

  it("renders success variant", () => {
    render(<Badge variant="success">Done</Badge>);
    expect(screen.getByText("Done")).toHaveClass("bg-green-100");
  });

  it("renders warning variant", () => {
    render(<Badge variant="warning">Blocked</Badge>);
    expect(screen.getByText("Blocked")).toHaveClass("bg-yellow-100");
  });

  it("renders danger variant", () => {
    render(<Badge variant="danger">Critical</Badge>);
    expect(screen.getByText("Critical")).toHaveClass("bg-red-100");
  });

  it("renders info variant", () => {
    render(<Badge variant="info">Info</Badge>);
    expect(screen.getByText("Info")).toHaveClass("bg-blue-100");
  });
});
