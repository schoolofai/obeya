"use client";

import { useEffect, useRef, useState } from "react";
import { getBrowserClient } from "@/lib/appwrite/browser-client";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

interface UseReconnectingSubscriptionOptions {
  boardId: string;
  databaseId: string;
  channel: string;
  onEvent: (event: unknown) => void;
  onRefresh: () => void;
}

interface UseReconnectingSubscriptionResult {
  status: ConnectionStatus;
}

export function useReconnectingSubscription(
  options: UseReconnectingSubscriptionOptions
): UseReconnectingSubscriptionResult {
  const { boardId, databaseId, channel, onEvent, onRefresh } = options;
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const onEventRef = useRef(onEvent);
  const onRefreshRef = useRef(onRefresh);
  const wasOfflineRef = useRef(false);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);

  useEffect(() => {
    if (!boardId) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    const client = getBrowserClient();

    const unsubscribe = client.subscribe(channel, (event: unknown) => {
      onEventRef.current(event);
    });

    setStatus("connected");

    const handleOffline = () => {
      wasOfflineRef.current = true;
      setStatus("disconnected");
    };

    const handleOnline = () => {
      if (wasOfflineRef.current) {
        wasOfflineRef.current = false;
        setStatus("connected");
        onRefreshRef.current();
      }
    };

    window.addEventListener("offline", handleOffline);
    window.addEventListener("online", handleOnline);

    return () => {
      unsubscribe();
      window.removeEventListener("offline", handleOffline);
      window.removeEventListener("online", handleOnline);
      setStatus("disconnected");
    };
  }, [boardId, databaseId, channel]);

  return { status };
}
