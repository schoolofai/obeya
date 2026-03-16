import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReviewContextPanel } from "@/components/board/review-context-panel";
import type { ReviewContext } from "@/lib/types";

const fullContext: ReviewContext = {
  purpose: "Replace cookie-based sessions with JWT tokens",
  files_changed: [
    { path: "auth/middleware.go", added: 82, removed: 41 },
    { path: "auth/session.go", added: 15, removed: 8 },
  ],
  tests_written: [
    { name: "TestJWTValidation", passed: true },
    { name: "TestSessionMigration", passed: true },
    { name: "TestEdgeCase", passed: false },
  ],
  proof: [
    { check: "go vet clean", status: "pass" },
    { check: "edge case tests", status: "fail", detail: "No concurrent session tests" },
    { check: "token refresh", status: "warn" },
  ],
  reasoning: "JWT chosen for debuggability",
  reproduce: ["go test ./internal/auth/ -run TestJWT", "go test ./internal/auth/ -v"],
};

describe("ReviewContextPanel", () => {
  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it("renders collapsed by default", () => {
    render(<ReviewContextPanel context={fullContext} />);
    expect(screen.getByText("review context")).toBeDefined();
    expect(screen.queryByText("Purpose")).toBeNull();
  });

  it("expands on click to show purpose", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByText("Purpose")).toBeDefined();
    expect(screen.getByText("Replace cookie-based sessions with JWT tokens")).toBeDefined();
  });

  it("shows files changed when expanded", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByTestId("files-list")).toBeDefined();
    expect(screen.getByText("auth/middleware.go")).toBeDefined();
    expect(screen.getByText("+82")).toBeDefined();
    expect(screen.getByText("-41")).toBeDefined();
  });

  it("shows test summary when expanded", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByTestId("tests-list")).toBeDefined();
    expect(screen.getByText("2/3 passing")).toBeDefined();
  });

  it("shows proof items when expanded", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByTestId("proof-list")).toBeDefined();
    expect(screen.getByText("go vet clean")).toBeDefined();
  });

  it("shows reasoning when expanded", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByText("JWT chosen for debuggability")).toBeDefined();
  });

  it("shows reproduce commands when expanded", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByTestId("reproduce-commands")).toBeDefined();
  });

  it("collapses back when clicked again", () => {
    render(<ReviewContextPanel context={fullContext} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByText("Purpose")).toBeDefined();
    fireEvent.click(screen.getByText("review context"));
    expect(screen.queryByText("Purpose")).toBeNull();
  });

  it("handles minimal context (purpose only)", () => {
    const minimal: ReviewContext = { purpose: "Quick fix" };
    render(<ReviewContextPanel context={minimal} />);
    fireEvent.click(screen.getByText("review context"));
    expect(screen.getByText("Quick fix")).toBeDefined();
    expect(screen.queryByTestId("files-list")).toBeNull();
    expect(screen.queryByTestId("tests-list")).toBeNull();
    expect(screen.queryByTestId("proof-list")).toBeNull();
  });
});
