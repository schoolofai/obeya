import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ApiTokenManager } from "@/components/settings/api-token-manager";

const mockTokens = [
  {
    $id: "tok1",
    name: "My laptop",
    scopes: ["*"],
    last_used_at: "2026-03-11T10:00:00Z",
    expires_at: null,
  },
  {
    $id: "tok2",
    name: "CI server",
    scopes: ["boards:read"],
    last_used_at: null,
    expires_at: null,
  },
];

describe("ApiTokenManager", () => {
  it("renders existing tokens", () => {
    render(<ApiTokenManager tokens={mockTokens} />);
    expect(screen.getByText("My laptop")).toBeDefined();
    expect(screen.getByText("CI server")).toBeDefined();
  });

  it("shows revoke button for each token", () => {
    render(<ApiTokenManager tokens={mockTokens} />);
    const revokeButtons = screen.getAllByText("Revoke");
    expect(revokeButtons.length).toBe(2);
  });

  it("renders create token form", () => {
    render(<ApiTokenManager tokens={mockTokens} />);
    expect(screen.getByPlaceholderText("Token name")).toBeDefined();
    expect(screen.getByText("Create Token")).toBeDefined();
  });
});
