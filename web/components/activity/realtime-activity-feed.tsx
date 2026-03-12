"use client";

import { useState, useCallback } from "react";
import {
  useActivitySubscription,
  type HistoryEntry,
  type ActivityEvent,
} from "@/hooks/use-activity-subscription";
import { ConnectionStatusIndicator } from "@/components/realtime/connection-status";

interface RealtimeActivityFeedProps {
  boardId: string;
  databaseId: string;
  initialEntries: HistoryEntry[];
  maxEntries?: number;
}

const ACTION_CONFIG: Record<string, { label: string; color: string }> = {
  created: { label: "created", color: "text-green-600 dark:text-green-400" },
  moved: { label: "moved", color: "text-blue-600 dark:text-blue-400" },
  edited: { label: "edited", color: "text-yellow-600 dark:text-yellow-400" },
  assigned: { label: "assigned", color: "text-purple-600 dark:text-purple-400" },
  blocked: { label: "blocked", color: "text-red-600 dark:text-red-400" },
  unblocked: { label: "unblocked", color: "text-teal-600 dark:text-teal-400" },
  deleted: { label: "deleted", color: "text-red-600 dark:text-red-400" },
};

export function RealtimeActivityFeed({
  boardId,
  databaseId,
  initialEntries,
  maxEntries = 50,
}: RealtimeActivityFeedProps) {
  const sorted = [...initialEntries].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );
  const [entries, setEntries] = useState<HistoryEntry[]>(sorted);

  const handleActivity = useCallback(
    (event: ActivityEvent) => {
      setEntries((prev) => {
        if (prev.some((e) => e.$id === event.entry.$id)) return prev;
        return [event.entry, ...prev].slice(0, maxEntries);
      });
    },
    [maxEntries]
  );

  const { status } = useActivitySubscription({
    boardId,
    databaseId,
    onActivity: handleActivity,
  });

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-gray-400">
        <p>No activity yet</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <FeedHeader status={status} />
      <div className="flex flex-col divide-y">
        {entries.map((entry) => (
          <ActivityEntry key={entry.$id} entry={entry} />
        ))}
      </div>
    </div>
  );
}

interface FeedHeaderProps {
  status: string;
}

function FeedHeader({ status }: FeedHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2 border-b">
      <h3 className="text-sm font-semibold">Activity</h3>
      <ConnectionStatusIndicator
        status={status as "connected" | "connecting" | "disconnected" | "error"}
        compact
      />
    </div>
  );
}

function ActivityEntry({ entry }: { entry: HistoryEntry }) {
  const config = ACTION_CONFIG[entry.action] || {
    label: entry.action,
    color: "text-gray-600",
  };
  const timeAgo = formatTimeAgo(entry.timestamp);

  return (
    <div
      className="px-4 py-3 transition-all duration-200 animate-in fade-in slide-in-from-top-1"
      data-activity-entry
    >
      <div className="flex items-center gap-2 text-sm">
        <span className="text-gray-500 text-xs">{entry.user_id}</span>
        <span className={`font-medium ${config.color}`}>{config.label}</span>
        <span className="text-gray-400 text-xs ml-auto">{timeAgo}</span>
      </div>
      <p className="text-xs text-gray-500 mt-1">{entry.detail}</p>
    </div>
  );
}

function formatTimeAgo(timestamp: string): string {
  const now = new Date();
  const then = new Date(timestamp);
  const diffMs = now.getTime() - then.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}
