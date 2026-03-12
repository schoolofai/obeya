"use client";

import { useState } from "react";
import type { HistoryEntry } from "@/lib/api-client";

const ACTION_COLORS: Record<string, string> = {
  created: "bg-green-100 text-green-800",
  moved: "bg-blue-100 text-blue-800",
  edited: "bg-yellow-100 text-yellow-800",
  assigned: "bg-purple-100 text-purple-800",
  blocked: "bg-red-100 text-red-800",
  unblocked: "bg-gray-100 text-gray-800",
};

function formatTimestamp(ts: string): string {
  const date = new Date(ts);
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface FilterButtonProps {
  label: string;
  active: boolean;
  onClick: () => void;
}

function FilterButton({ label, active, onClick }: FilterButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
        active
          ? "bg-indigo-600 text-white"
          : "bg-gray-100 text-gray-600 hover:bg-gray-200"
      }`}
    >
      {label}
    </button>
  );
}

interface ActivityEntryProps {
  entry: HistoryEntry;
}

function ActivityEntry({ entry }: ActivityEntryProps) {
  const colorClass = ACTION_COLORS[entry.action] ?? "bg-gray-100 text-gray-800";
  return (
    <li className="flex items-start gap-3 py-3 border-b border-gray-100 last:border-0">
      <span className={`mt-0.5 px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
        {entry.action}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-800">{entry.detail}</p>
        <p className="text-xs text-gray-400 mt-0.5">{formatTimestamp(entry.timestamp)}</p>
      </div>
    </li>
  );
}

interface ActivityFeedProps {
  entries: HistoryEntry[];
}

export function ActivityFeed({ entries }: ActivityFeedProps) {
  const [activeFilter, setActiveFilter] = useState<string | null>(null);

  const uniqueActions = Array.from(new Set(entries.map((e) => e.action)));

  const filtered =
    activeFilter === null
      ? entries
      : entries.filter((e) => e.action === activeFilter);

  return (
    <div>
      <div className="flex flex-wrap gap-2 mb-4">
        <FilterButton
          label="All"
          active={activeFilter === null}
          onClick={() => setActiveFilter(null)}
        />
        {uniqueActions.map((action) => (
          <FilterButton
            key={action}
            label={action.charAt(0).toUpperCase() + action.slice(1)}
            active={activeFilter === action}
            onClick={() => setActiveFilter(action)}
          />
        ))}
      </div>

      {filtered.length === 0 ? (
        <p className="text-sm text-gray-500">No activity yet.</p>
      ) : (
        <ul>
          {filtered.map((entry) => (
            <ActivityEntry key={entry.$id} entry={entry} />
          ))}
        </ul>
      )}
    </div>
  );
}
