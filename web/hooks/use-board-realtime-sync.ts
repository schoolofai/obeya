"use client";

import { useCallback, useRef } from "react";
import {
  useBoardSubscription,
  type BoardItemEvent,
} from "@/hooks/use-board-subscription";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

export interface BoardItem {
  $id: string;
  board_id: string;
  display_num: number;
  type: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  parent_id?: string;
  assignee_id?: string;
  blocked_by?: string[];
  tags?: string[];
  project?: string;
  created_at: string;
  updated_at: string;
}

interface UseBoardRealtimeSyncOptions {
  boardId: string;
  databaseId: string;
  onItemCreated: (item: BoardItem) => void;
  onItemUpdated: (item: BoardItem) => void;
  onItemDeleted: (itemId: string) => void;
}

interface UseBoardRealtimeSyncResult {
  status: ConnectionStatus;
}

export function useBoardRealtimeSync(
  options: UseBoardRealtimeSyncOptions
): UseBoardRealtimeSyncResult {
  const {
    boardId,
    databaseId,
    onItemCreated,
    onItemUpdated,
    onItemDeleted,
  } = options;

  const onItemCreatedRef = useRef(onItemCreated);
  const onItemUpdatedRef = useRef(onItemUpdated);
  const onItemDeletedRef = useRef(onItemDeleted);

  onItemCreatedRef.current = onItemCreated;
  onItemUpdatedRef.current = onItemUpdated;
  onItemDeletedRef.current = onItemDeleted;

  const handleEvent = useCallback((event: BoardItemEvent) => {
    const item = event.item as unknown as BoardItem;

    switch (event.action) {
      case "create":
        onItemCreatedRef.current(item);
        break;
      case "update":
        onItemUpdatedRef.current(item);
        break;
      case "delete":
        onItemDeletedRef.current(item.$id);
        break;
    }
  }, []);

  const { status } = useBoardSubscription({
    boardId,
    databaseId,
    onEvent: handleEvent,
  });

  return { status };
}
