import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useReconnectingSubscription } from "@/hooks/use-reconnecting-subscription";

describe("useReconnectingSubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("calls onRefresh when online event fires after being offline", () => {
    const onRefresh = vi.fn();
    const onEvent = vi.fn();

    renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent,
        onRefresh,
      })
    );

    act(() => {
      window.dispatchEvent(new Event("offline"));
    });
    act(() => {
      window.dispatchEvent(new Event("online"));
    });

    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("subscribes to the correct channel", () => {
    const onEvent = vi.fn();
    const onRefresh = vi.fn();

    renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent,
        onRefresh,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);
    expect(mockSubscribe.mock.calls[0][0]).toBe(
      "databases.obeya.collections.items.documents"
    );
  });

  it("unsubscribes and removes listeners on unmount", () => {
    const removeListenerSpy = vi.spyOn(window, "removeEventListener");

    const { unmount } = renderHook(() =>
      useReconnectingSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent: vi.fn(),
        onRefresh: vi.fn(),
      })
    );

    unmount();

    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
    expect(removeListenerSpy).toHaveBeenCalled();
  });

  it("does not subscribe when boardId is empty", () => {
    renderHook(() =>
      useReconnectingSubscription({
        boardId: "",
        databaseId: "obeya",
        channel: "databases.obeya.collections.items.documents",
        onEvent: vi.fn(),
        onRefresh: vi.fn(),
      })
    );

    expect(mockSubscribe).not.toHaveBeenCalled();
  });
});
