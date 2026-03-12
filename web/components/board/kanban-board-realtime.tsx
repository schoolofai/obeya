"use client";

import { useState, useCallback } from "react";
import {
  useBoardRealtimeSync,
  type BoardItem,
} from "@/hooks/use-board-realtime-sync";
import { useActivitySubscription } from "@/hooks/use-activity-subscription";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";
import type { ActivityEvent } from "@/hooks/use-activity-subscription";
import type { HistoryEntry } from "@/hooks/use-activity-subscription";
import type { ConnectionStatus } from "@/hooks/use-board-subscription";

interface KanbanBoardRealtimeProps {
  boardId: string;
  databaseId: string;
  initialItems: BoardItem[];
  columns: { name: string; limit: number }[];
}

export function KanbanBoardRealtime({
  boardId,
  databaseId,
  initialItems,
  columns,
}: KanbanBoardRealtimeProps) {
  const [items, setItems] = useState<BoardItem[]>(initialItems);
  const [activities, setActivities] = useState<HistoryEntry[]>([]);

  const handleItemCreated = useCallback((item: BoardItem) => {
    setItems((prev) => {
      if (prev.some((existing) => existing.$id === item.$id)) {
        return prev;
      }
      return [...prev, item];
    });
  }, []);

  const handleItemUpdated = useCallback((item: BoardItem) => {
    setItems((prev) =>
      prev.map((existing) =>
        existing.$id === item.$id ? item : existing
      )
    );
  }, []);

  const handleItemDeleted = useCallback((itemId: string) => {
    setItems((prev) => prev.filter((item) => item.$id !== itemId));
  }, []);

  const handleActivity = useCallback((event: ActivityEvent) => {
    setActivities((prev) => [event.entry, ...prev].slice(0, 50));
  }, []);

  const { status: boardStatus } = useBoardRealtimeSync({
    boardId,
    databaseId,
    onItemCreated: handleItemCreated,
    onItemUpdated: handleItemUpdated,
    onItemDeleted: handleItemDeleted,
  });

  const { status: activityStatus } = useActivitySubscription({
    boardId,
    databaseId,
    onActivity: handleActivity,
  });

  const effectiveStatus = deriveEffectiveStatus(boardStatus, activityStatus);

  const itemsByColumn = buildColumnMap(columns, items);

  return (
    <div className="flex flex-col h-full">
      <BoardHeader status={effectiveStatus} />
      <BoardColumns
        columns={columns}
        itemsByColumn={itemsByColumn}
      />
    </div>
  );
}

function deriveEffectiveStatus(
  boardStatus: string,
  activityStatus: string
): ConnectionStatus {
  if (boardStatus === "error" || activityStatus === "error") return "error";
  if (boardStatus === "connecting" || activityStatus === "connecting") return "connecting";
  if (boardStatus === "connected" && activityStatus === "connected") return "connected";
  return "disconnected";
}

function buildColumnMap(
  columns: { name: string; limit: number }[],
  items: BoardItem[]
): Record<string, BoardItem[]> {
  const map: Record<string, BoardItem[]> = {};
  for (const col of columns) {
    map[col.name] = [];
  }
  for (const item of items) {
    if (map[item.status]) {
      map[item.status].push(item);
    }
  }
  return map;
}

interface BoardHeaderProps {
  status: ConnectionStatus;
}

function BoardHeader({ status }: BoardHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2 border-b">
      <h2 className="text-lg font-semibold">Board</h2>
      <ConnectionStatusIndicator status={status} />
    </div>
  );
}

interface BoardColumnsProps {
  columns: { name: string; limit: number }[];
  itemsByColumn: Record<string, BoardItem[]>;
}

function BoardColumns({ columns, itemsByColumn }: BoardColumnsProps) {
  return (
    <div className="flex flex-1 overflow-x-auto gap-4 p-4">
      {columns.map((col) => {
        const colItems = itemsByColumn[col.name] || [];
        const isOverLimit = col.limit > 0 && colItems.length > col.limit;
        return (
          <KanbanColumn
            key={col.name}
            col={col}
            items={colItems}
            isOverLimit={isOverLimit}
          />
        );
      })}
    </div>
  );
}

interface KanbanColumnProps {
  col: { name: string; limit: number };
  items: BoardItem[];
  isOverLimit: boolean;
}

function KanbanColumn({ col, items, isOverLimit }: KanbanColumnProps) {
  return (
    <div className="flex flex-col min-w-[280px] max-w-[320px] bg-gray-50 dark:bg-gray-800 rounded-lg">
      <div className="flex items-center justify-between px-3 py-2 border-b">
        <span className="font-medium text-sm">{col.name}</span>
        <span
          className={`text-xs ${isOverLimit ? "text-red-500 font-bold" : "text-gray-400"}`}
        >
          {items.length}
          {col.limit > 0 && `/${col.limit}`}
        </span>
      </div>
      <div className="flex flex-col gap-2 p-2 overflow-y-auto">
        {items.map((item) => (
          <KanbanItemCard key={item.$id} item={item} />
        ))}
      </div>
    </div>
  );
}

interface KanbanItemCardProps {
  item: BoardItem;
}

function KanbanItemCard({ item }: KanbanItemCardProps) {
  return (
    <div className="bg-white dark:bg-gray-700 rounded-md p-3 shadow-sm border border-gray-200 dark:border-gray-600 transition-all duration-200 animate-in fade-in">
      <div className="flex items-center gap-2 mb-1">
        <span className="text-xs text-gray-400">#{item.display_num}</span>
        <span className="text-xs px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-600 text-gray-600 dark:text-gray-300">
          {item.type}
        </span>
        <PriorityBadge priority={item.priority} />
      </div>
      <p className="text-sm font-medium">{item.title}</p>
      {item.assignee_id && (
        <p className="text-xs text-gray-400 mt-1">@{item.assignee_id}</p>
      )}
    </div>
  );
}

const PRIORITY_COLORS: Record<string, string> = {
  critical: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
  high: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300",
  medium: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300",
  low: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400",
};

function PriorityBadge({ priority }: { priority: string }) {
  const colorClass = PRIORITY_COLORS[priority] || PRIORITY_COLORS.medium;
  return (
    <span className={`text-xs px-1.5 py-0.5 rounded ${colorClass}`}>
      {priority}
    </span>
  );
}
