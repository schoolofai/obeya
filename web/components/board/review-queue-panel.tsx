"use client";

import { useMemo, useCallback } from "react";
import type { BoardItem } from "@/lib/api-client";
import { AgentBadge } from "./agent-badge";
import { ConfidenceGauge } from "./confidence-gauge";

interface ReviewQueuePanelProps {
  items: BoardItem[];
  onCardClick: (item: BoardItem) => void;
  onMarkReviewed: (item: BoardItem) => void;
  onHide: (item: BoardItem) => void;
  className?: string;
}

function sortByConfidence(items: BoardItem[]): BoardItem[] {
  return [...items].sort((a, b) => {
    const aConf = a.confidence ?? -1;
    const bConf = b.confidence ?? -1;
    if (aConf !== bConf) return aConf - bConf;
    return new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
  });
}

function filterReviewQueue(items: BoardItem[]): BoardItem[] {
  return items.filter(
    (item) =>
      item.status === "done" &&
      item.review_context !== null &&
      item.review_context !== undefined &&
      (item.human_review === null ||
        item.human_review === undefined ||
        item.human_review.status !== "hidden")
  );
}

function ReviewQueueCard({ item, onClick, onMarkReviewed, onHide }: {
  item: BoardItem;
  onClick: () => void;
  onMarkReviewed: () => void;
  onHide: () => void;
}) {
  const isReviewed = item.human_review?.status === "reviewed";

  const handleReview = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onMarkReviewed();
    },
    [onMarkReviewed]
  );

  const handleHide = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onHide();
    },
    [onHide]
  );

  return (
    <div
      data-testid={`review-card-${item.display_num}`}
      onClick={onClick}
      className={`rounded-lg border p-3 cursor-pointer transition-all
        ${isReviewed
          ? "border-green-500/30 bg-green-900/10"
          : "border-amber-500/20 bg-[var(--bg-secondary)] hover:border-amber-500/40"
        }`}
    >
      <div className="flex items-center gap-2 mb-1.5">
        <AgentBadge />
        <span className="text-xs font-mono text-[var(--text-faint)]">
          #{item.display_num}
        </span>
        {isReviewed && (
          <span data-testid="reviewed-checkmark" className="text-green-400 text-xs ml-auto">
            &#x2713;
          </span>
        )}
      </div>

      <p className="text-sm font-medium text-[var(--text-primary)] line-clamp-2 mb-2">
        {item.title}
      </p>

      {item.confidence !== null && item.confidence !== undefined && (
        <ConfidenceGauge confidence={item.confidence} className="mb-2" />
      )}

      {item.review_context?.purpose && (
        <p className="text-xs text-[var(--text-secondary)] line-clamp-2 mb-2">
          {item.review_context.purpose}
        </p>
      )}

      {item.sponsor && (
        <p className="text-xs text-[var(--text-faint)] mb-2">
          sponsor: @{item.sponsor}
        </p>
      )}

      <div className="flex items-center gap-2 mt-2">
        <button
          onClick={handleReview}
          disabled={isReviewed}
          className={`flex-1 text-xs font-medium py-1.5 px-3 rounded transition-colors
            ${isReviewed
              ? "bg-green-900/20 text-green-400 cursor-default"
              : "bg-green-600/20 text-green-400 hover:bg-green-600/30"
            }`}
        >
          {isReviewed ? "Reviewed" : "Mark Reviewed"}
        </button>
        <button
          onClick={handleHide}
          className="text-xs font-medium py-1.5 px-3 rounded
            bg-[var(--bg-tertiary)] text-[var(--text-secondary)]
            hover:text-[var(--text-primary)] hover:bg-[var(--border-default)]
            transition-colors"
        >
          Hide
        </button>
      </div>
    </div>
  );
}

export function ReviewQueuePanel({
  items,
  onCardClick,
  onMarkReviewed,
  onHide,
  className = "",
}: ReviewQueuePanelProps) {
  const queueItems = useMemo(() => {
    const filtered = filterReviewQueue(items);
    return sortByConfidence(filtered);
  }, [items]);

  if (queueItems.length === 0) return null;

  return (
    <div
      data-testid="review-queue-panel"
      className={`flex flex-col w-72 shrink-0 ${className}`}
    >
      <div className="mb-3 px-1">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-amber-400 uppercase tracking-wide">
            &#x26A1; Review Queue
          </h3>
          <span className="text-xs font-mono text-amber-400/70">
            ({queueItems.length})
          </span>
        </div>
        <p className="text-xs text-[var(--text-faint)] mt-0.5">
          sorted by confidence
        </p>
      </div>

      <div className="flex-1 space-y-2 p-2 rounded-lg min-h-[200px]
        border border-amber-500/20 bg-amber-900/5">
        {queueItems.map((item) => (
          <ReviewQueueCard
            key={item.$id}
            item={item}
            onClick={() => onCardClick(item)}
            onMarkReviewed={() => onMarkReviewed(item)}
            onHide={() => onHide(item)}
          />
        ))}
      </div>
    </div>
  );
}
