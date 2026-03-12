"use client";

import { useEffect, useState, useCallback } from "react";
import type { BoardItem, HistoryEntry } from "@/lib/api-client";

interface ItemDetailPanelProps {
  item: BoardItem;
  boardId: string;
  onClose: () => void;
  onUpdate: (updated: BoardItem) => void;
}

const PRIORITY_COLORS: Record<string, string> = {
  critical: "bg-red-100 text-red-700",
  high: "bg-orange-100 text-orange-700",
  medium: "bg-yellow-100 text-yellow-700",
  low: "bg-gray-100 text-gray-600",
};

function BlockedSection({ blockedBy }: { blockedBy: string[] }) {
  if (blockedBy.length === 0) return null;
  return (
    <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
      <p className="text-sm font-semibold text-red-700 mb-1">Blocked by</p>
      <ul className="space-y-1">
        {blockedBy.map((id) => (
          <li key={id} className="text-sm text-red-600 font-mono">
            #{id}
          </li>
        ))}
      </ul>
    </div>
  );
}

function TagList({ tags }: { tags: string[] }) {
  if (tags.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-1 mt-2">
      {tags.map((tag) => (
        <span
          key={tag}
          className="text-xs px-2 py-0.5 rounded-full bg-indigo-100 text-indigo-700"
        >
          {tag}
        </span>
      ))}
    </div>
  );
}

function HistoryList({ history }: { history: HistoryEntry[] }) {
  if (history.length === 0) return null;
  return (
    <div className="mt-4">
      <h4 className="text-sm font-semibold text-gray-700 mb-2">Activity</h4>
      <ul className="space-y-2">
        {history.map((entry) => (
          <li key={entry.$id} className="text-xs text-gray-600">
            <span className="font-medium">{entry.action}</span>
            {entry.detail && <span className="ml-1">{entry.detail}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function ItemDetailPanel({
  item,
  boardId,
  onClose,
  onUpdate,
}: ItemDetailPanelProps) {
  const [history, setHistory] = useState<HistoryEntry[]>([]);

  const loadDetails = useCallback(async () => {
    const res = await fetch(`/api/items/${item.$id}/history`);
    const body = await res.json();
    if (!body.ok) throw new Error(body.error?.message ?? "Failed to load history");
    setHistory(body.data);
  }, [boardId, item.$id]);

  useEffect(() => {
    loadDetails();
  }, [loadDetails]);

  return (
    <div className="fixed inset-y-0 right-0 w-96 bg-white border-l border-gray-200 shadow-xl flex flex-col z-50">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-gray-500">
            #{item.display_num}
          </span>
          <span
            className={`text-xs px-2 py-0.5 rounded-full font-medium ${PRIORITY_COLORS[item.priority]}`}
          >
            {item.priority}
          </span>
        </div>
        <button
          aria-label="Close panel"
          onClick={onClose}
          className="text-gray-400 hover:text-gray-600 transition-colors"
        >
          ✕
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-4">
        <h2 className="text-base font-semibold text-gray-900 mb-2">
          {item.title}
        </h2>
        {item.description && (
          <p className="text-sm text-gray-600 leading-relaxed">
            {item.description}
          </p>
        )}
        <TagList tags={item.tags} />
        <BlockedSection blockedBy={item.blocked_by} />
        <HistoryList history={history} />
      </div>
    </div>
  );
}
