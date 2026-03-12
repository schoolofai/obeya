import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const mockUnsubscribe = vi.fn();
const mockSubscribe = vi.fn().mockReturnValue(mockUnsubscribe);

vi.mock("@/lib/appwrite/browser-client", () => ({
  getBrowserClient: vi.fn(() => ({
    subscribe: mockSubscribe,
  })),
}));

import { useBoardSubscription } from "@/hooks/use-board-subscription";
import type { BoardItemEvent } from "@/hooks/use-board-subscription";

describe("useBoardSubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("subscribes to the correct Appwrite channel on mount", () => {
    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(mockSubscribe).toHaveBeenCalledTimes(1);
    const channelArg = mockSubscribe.mock.calls[0][0];
    expect(channelArg).toBe("databases.obeya.collections.items.documents");
  });

  it("unsubscribes on unmount", () => {
    const onEvent = vi.fn();
    const { unmount } = renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );
    unmount();
    expect(mockUnsubscribe).toHaveBeenCalledTimes(1);
  });

  it("does not subscribe when boardId is empty", () => {
    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "",
        databaseId: "obeya",
        onEvent,
      })
    );
    expect(mockSubscribe).not.toHaveBeenCalled();
  });

  it("filters events by board_id and calls onEvent with parsed data", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    expect(capturedCallback).not.toBeNull();

    act(() => {
      capturedCallback!({
        events: ["databases.obeya.collections.items.documents.item-1.create"],
        payload: {
          $id: "item-1",
          board_id: "board-123",
          display_num: 42,
          type: "task",
          title: "New task",
          status: "todo",
          priority: "medium",
          created_at: "2026-03-12T10:00:00Z",
          updated_at: "2026-03-12T10:00:00Z",
        },
      });
    });

    expect(onEvent).toHaveBeenCalledTimes(1);
    const eventArg: BoardItemEvent = onEvent.mock.calls[0][0];
    expect(eventArg.action).toBe("create");
    expect(eventArg.item.$id).toBe("item-1");
  });

  it("ignores events for a different board_id", () => {
    let capturedCallback: ((event: any) => void) | null = null;
    mockSubscribe.mockImplementation(
      (_channel: string, callback: (event: any) => void) => {
        capturedCallback = callback;
        return mockUnsubscribe;
      }
    );

    const onEvent = vi.fn();
    renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );

    act(() => {
      capturedCallback!({
        events: ["databases.obeya.collections.items.documents.item-99.create"],
        payload: {
          $id: "item-99",
          board_id: "other-board",
          title: "Wrong board",
        },
      });
    });

    expect(onEvent).not.toHaveBeenCalled();
  });

  it("tracks connection status", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() =>
      useBoardSubscription({
        boardId: "board-123",
        databaseId: "obeya",
        onEvent,
      })
    );
    expect(result.current.status).toBe("connected");
  });

  it("reports disconnected status when boardId is empty", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() =>
      useBoardSubscription({
        boardId: "",
        databaseId: "obeya",
        onEvent,
      })
    );
    expect(result.current.status).toBe("disconnected");
  });
});
