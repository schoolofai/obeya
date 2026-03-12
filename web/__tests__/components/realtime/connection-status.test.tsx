import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";

describe("ConnectionStatusIndicator", () => {
  it("renders connected state with green indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connected" />
    );
    const indicator = container.querySelector("[data-status='connected']");
    expect(indicator).not.toBeNull();
    const text = container.textContent;
    expect(text).toContain("Connected");
  });

  it("renders connecting state with yellow indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connecting" />
    );
    const indicator = container.querySelector("[data-status='connecting']");
    expect(indicator).not.toBeNull();
    const text = container.textContent;
    expect(text).toContain("Connecting");
  });

  it("renders disconnected state with gray indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="disconnected" />
    );
    const indicator = container.querySelector("[data-status='disconnected']");
    expect(indicator).not.toBeNull();
    const text = container.textContent;
    expect(text).toContain("Offline");
  });

  it("renders error state with red indicator", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="error" />
    );
    const indicator = container.querySelector("[data-status='error']");
    expect(indicator).not.toBeNull();
    const text = container.textContent;
    expect(text).toContain("Error");
  });

  it("renders compact mode without text", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connected" compact />
    );
    const text = container.textContent;
    expect(text).not.toContain("Connected");
    const dot = container.querySelector("[data-status='connected']");
    expect(dot).not.toBeNull();
  });
});
