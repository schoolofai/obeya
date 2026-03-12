"use client";

import { useEffect, useRef, useState } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

export interface HistoryEntry {
  $id: string;
  item_id: string;
  board_id: string;
  user_id: string;
  session_id?: string;
  action: string;
  detail: string;
  timestamp: string;
}

export interface ActivityEvent {
  entry: HistoryEntry;
}

interface UseActivitySubscriptionOptions {
  boardId: string;
  databaseId: string;
  onActivity: (event: ActivityEvent) => void;
}

interface UseActivitySubscriptionResult {
  status: ConnectionStatus;
}

export function useActivitySubscription(
  options: UseActivitySubscriptionOptions
): UseActivitySubscriptionResult {
  const { boardId, databaseId, onActivity } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onActivityRef = useRef(onActivity);

  useEffect(() => {
    onActivityRef.current = onActivity;
  }, [onActivity]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();
    const channel = `databases.${databaseId}.collections.item_history.documents`;

    const unsubscribe = client.subscribe(channel, (event: any) => {
      const payload = event.payload;

      if (payload.board_id !== boardId) {
        return;
      }

      const events: string[] = event.events || [];
      const isCreate = events.some((e: string) => e.endsWith(".create"));
      if (!isCreate) return;

      const entry: HistoryEntry = {
        $id: payload.$id,
        item_id: payload.item_id,
        board_id: payload.board_id,
        user_id: payload.user_id,
        session_id: payload.session_id,
        action: payload.action,
        detail: payload.detail,
        timestamp: payload.timestamp,
      };

      onActivityRef.current({ entry });
    });

    setStatus("connected");

    return () => {
      unsubscribe();
      setStatus("disconnected");
    };
  }, [boardId, databaseId]);

  return { status };
}
