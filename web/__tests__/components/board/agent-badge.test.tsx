import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentBadge } from "@/components/board/agent-badge";

describe("AgentBadge", () => {
  it("renders with Agent text", () => {
    render(<AgentBadge />);
    expect(screen.getByText("Agent")).toBeDefined();
  });

  it("has correct test id", () => {
    render(<AgentBadge />);
    expect(screen.getByTestId("agent-badge")).toBeDefined();
  });

  it("applies custom className", () => {
    render(<AgentBadge className="ml-2" />);
    const badge = screen.getByTestId("agent-badge");
    expect(badge.className).toContain("ml-2");
  });
});
