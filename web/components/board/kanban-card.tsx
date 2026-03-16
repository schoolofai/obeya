"use client";

import type { BoardItem } from "@/lib/api-client";
import { AgentBadge } from "./agent-badge";
import { ConfidenceGauge } from "./confidence-gauge";

const TYPE_ICONS: Record<string, string> = {
  epic: "E",
  story: "S",
  task: "T",
};

const TYPE_COLORS: Record<string, string> = {
  epic: "bg-purple-100 text-purple-700",
  story: "bg-blue-100 text-blue-700",
  task: "bg-gray-100 text-gray-700",
};

const PRIORITY_COLORS: Record<string, string> = {
  critical: "bg-red-600 text-white",
  high: "bg-orange-500 text-white",
  medium: "bg-yellow-400 text-gray-900",
  low: "bg-gray-300 text-gray-700",
};

interface KanbanCardProps {
  item: BoardItem;
  onClick: () => void;
}

function isAgentItem(item: BoardItem): boolean {
  return item.review_context !== null && item.review_context !== undefined;
}

export function KanbanCard({ item, onClick }: KanbanCardProps) {
  const isBlocked = item.blocked_by.length > 0;
  const isAgent = isAgentItem(item);
  const isReviewed = item.human_review?.status === "reviewed";

  return (
    <button
      onClick={onClick}
      className={`w-full text-left rounded-lg border p-3 shadow-sm
        hover:shadow-md transition-shadow bg-white
        ${isBlocked ? "border-red-300 bg-red-50" : ""}
        ${isReviewed ? "border-green-300 bg-green-50" : ""}
        ${!isBlocked && !isReviewed ? "border-gray-200" : ""}`}
    >
      <div className="flex items-center gap-2 mb-1">
        {isAgent && <AgentBadge />}
        <span
          data-testid="type-icon"
          className={`text-xs font-bold px-1.5 py-0.5 rounded ${TYPE_COLORS[item.type]}`}
        >
          {TYPE_ICONS[item.type]}
        </span>
        <span className="text-xs text-gray-500 font-mono">
          #{item.display_num}
        </span>
        {isReviewed && (
          <span data-testid="card-reviewed-check" className="text-green-600 text-xs ml-auto">
            &#x2713;
          </span>
        )}
      </div>
      <p className="text-sm font-medium text-gray-900 line-clamp-2">
        {item.title}
      </p>
      <div className="flex items-center justify-between mt-2">
        <div className="flex items-center gap-2">
          <span
            className={`text-xs px-2 py-0.5 rounded-full font-medium ${PRIORITY_COLORS[item.priority]}`}
          >
            {item.priority}
          </span>
          {item.confidence !== null && item.confidence !== undefined && (
            <ConfidenceGauge confidence={item.confidence} />
          )}
        </div>
        {item.assignee_id && (
          <div className="flex items-center gap-1.5">
            <div className="w-6 h-6 rounded-full bg-indigo-500 flex items-center justify-center">
              <span className="text-xs text-white font-medium">
                {item.assignee_id.charAt(0).toUpperCase()}
              </span>
            </div>
            {item.sponsor && (
              <span className="text-xs text-gray-400">
                sponsor: @{item.sponsor}
              </span>
            )}
          </div>
        )}
      </div>
      {isBlocked && (
        <div className="mt-2 text-xs text-red-600 font-medium">Blocked</div>
      )}
    </button>
  );
}
