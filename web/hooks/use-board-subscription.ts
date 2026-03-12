"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";

export type ConnectionStatus = "connected" | "connecting" | "disconnected" | "error";
export type ItemAction = "create" | "update" | "delete";

export interface BoardItemEvent {
  action: ItemAction;
  item: Record<string, unknown>;
}

interface UseBoardSubscriptionOptions {
  boardId: string;
  databaseId: string;
  onEvent: (event: BoardItemEvent) => void;
}

interface UseBoardSubscriptionResult {
  status: ConnectionStatus;
}

function parseAction(eventString: string): ItemAction | null {
  const parts = eventString.split(".");
  const lastPart = parts[parts.length - 1];

  if (lastPart === "create" || lastPart === "update" || lastPart === "delete") {
    return lastPart;
  }
  return null;
}

export function useBoardSubscription(
  options: UseBoardSubscriptionOptions
): UseBoardSubscriptionResult {
  const { boardId, databaseId, onEvent } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onEventRef = useRef(onEvent);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();
    const channel = `databases.${databaseId}.collections.items.documents`;

    const unsubscribe = client.subscribe(channel, (event: any) => {
      const payload = event.payload;

      if (payload.board_id !== boardId) {
        return;
      }

      const events: string[] = event.events || [];
      let action: ItemAction | null = null;

      for (const eventStr of events) {
        action = parseAction(eventStr);
        if (action) break;
      }

      if (!action) return;

      onEventRef.current({ action, item: payload });
    });

    setStatus("connected");

    return () => {
      unsubscribe();
      setStatus("disconnected");
    };
  }, [boardId, databaseId]);

  return { status };
}
