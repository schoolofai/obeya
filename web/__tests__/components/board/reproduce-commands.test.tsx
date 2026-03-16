import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { ReproduceCommands } from "@/components/board/reproduce-commands";

describe("ReproduceCommands", () => {
  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it("renders nothing when commands is empty", () => {
    const { container } = render(<ReproduceCommands commands={[]} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders commands with $ prefix", () => {
    render(<ReproduceCommands commands={["go test ./..."]} />);
    expect(screen.getByText("$ go test ./...")).toBeDefined();
  });

  it("renders multiple commands", () => {
    render(
      <ReproduceCommands
        commands={["go test ./auth/", "go vet ./..."]}
      />
    );
    expect(screen.getByText("$ go test ./auth/")).toBeDefined();
    expect(screen.getByText("$ go vet ./...")).toBeDefined();
  });

  it("has copy buttons for each command", () => {
    render(
      <ReproduceCommands commands={["cmd1", "cmd2"]} />
    );
    const buttons = screen.getAllByText("Copy");
    expect(buttons.length).toBe(2);
  });

  it("copies command to clipboard on click", async () => {
    render(<ReproduceCommands commands={["go test ./..."]} />);
    const copyBtn = screen.getByText("Copy");
    await act(async () => {
      fireEvent.click(copyBtn);
    });
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("go test ./...");
  });

  it("shows Copied! feedback after clicking copy", async () => {
    render(<ReproduceCommands commands={["go test ./..."]} />);
    const copyBtn = screen.getByText("Copy");
    await act(async () => {
      fireEvent.click(copyBtn);
    });
    expect(screen.getByText("Copied!")).toBeDefined();
  });

  it("renders Reproduce header", () => {
    render(<ReproduceCommands commands={["cmd"]} />);
    expect(screen.getByText("Reproduce")).toBeDefined();
  });
});
