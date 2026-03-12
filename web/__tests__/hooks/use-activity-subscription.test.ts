import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useActivitySubscription } from "@/hooks/use-activity-subscription";
import type { ActivityEvent } from "@/hooks/use-activity-subscription";

describe("useActivitySubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("subscribes to the item_history collection channel", () => {
    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);
    const channelArg = mockSubscribe.mock.calls[0][0];
    expect(channelArg).toBe("databases.obeya.collections.item_history.documents");
  });

  it("unsubscribes on unmount", () => {
    const onActivity = vi.fn();
    const { unmount } = renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );
    unmount();
    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
  });

  it("does not subscribe when boardId is empty", () => {
    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "",
        databaseId: "obeya",
        onActivity,
      })
    );
    expect(mockSubscribe).not.toHaveBeenCalled();
  });

  it("filters events by board_id and calls onActivity", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    expect(capturedCallback).not.toBeNull();

    act(() => {
      capturedCallback!({
        events: ["databases.obeya.collections.item_history.documents.hist-1.create"],
        payload: {
          $id: "hist-1",
          item_id: "item-42",
          board_id: "board-123",
          user_id: "user-1",
          action: "moved",
          detail: "status: todo -> in-progress",
          timestamp: "2026-03-12T10:05:00Z",
        },
      });
    });

    expect(onActivity).toHaveBeenCalledTimes(1);
    const activityArg: ActivityEvent = onActivity.mock.calls[0][0];
    expect(activityArg.entry.item_id).toBe("item-42");
    expect(activityArg.entry.action).toBe("moved");
  });

  it("ignores events for a different board_id", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onActivity = vi.fn();
    renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );

    act(() => {
      capturedCallback!({
        events: ["databases.obeya.collections.item_history.documents.hist-99.create"],
        payload: {
          $id: "hist-99",
          board_id: "other-board",
          action: "created",
          detail: "wrong board",
        },
      });
    });

    expect(onActivity).not.toHaveBeenCalled();
  });

  it("reports connection status", () => {
    const onActivity = vi.fn();
    const { result } = renderHook(() =>
      useActivitySubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onActivity,
      })
    );
    expect(result.current.status).toBe("connected");
  });
});
