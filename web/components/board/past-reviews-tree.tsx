"use client";

import { useState, useCallback, useMemo } from "react";
import type { HumanReview } from "@/lib/types";
import { ConfidenceGauge } from "./confidence-gauge";

export interface ReviewTreeItem {
  id: string;
  display_num: number;
  title: string;
  type: "epic" | "story" | "task";
  parent_id: string | null;
  confidence: number | null;
  human_review: HumanReview | null;
}

export interface TreeNode {
  item: ReviewTreeItem;
  children: TreeNode[];
  isStructural: boolean;
}

export function buildReviewTree(items: ReviewTreeItem[]): TreeNode[] {
  const reviewed = items.filter((i) => i.human_review !== null);
  if (reviewed.length === 0) return [];

  const itemMap = new Map(items.map((i) => [i.id, i]));
  const reviewedIds = new Set(reviewed.map((i) => i.id));
  const nodeMap = new Map<string, TreeNode>();
  const needed = new Set<string>();

  for (const item of reviewed) {
    needed.add(item.id);
    let current = item;
    while (current.parent_id && itemMap.has(current.parent_id)) {
      needed.add(current.parent_id);
      current = itemMap.get(current.parent_id)!;
    }
  }

  for (const id of needed) {
    const item = itemMap.get(id)!;
    nodeMap.set(id, {
      item,
      children: [],
      isStructural: !reviewedIds.has(id),
    });
  }

  const roots: TreeNode[] = [];
  for (const node of nodeMap.values()) {
    if (node.item.parent_id && nodeMap.has(node.item.parent_id)) {
      nodeMap.get(node.item.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => a.item.display_num - b.item.display_num);
    nodes.forEach((n) => sortNodes(n.children));
  };
  sortNodes(roots);

  return roots;
}

interface TreeNodeRowProps {
  node: TreeNode;
  depth: number;
  onSelect: (id: string) => void;
}

function TreeNodeRow({ node, depth, onSelect }: TreeNodeRowProps) {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = node.children.length > 0;

  const handleToggle = useCallback(() => {
    if (hasChildren) setExpanded(!expanded);
  }, [hasChildren, expanded]);

  const reviewStatus = node.item.human_review?.status;
  const statusIcon = reviewStatus === "reviewed" ? "\u2713" : reviewStatus === "hidden" ? "\u2014" : "";

  return (
    <>
      <div
        data-testid={`tree-node-${node.item.display_num}`}
        className={`flex items-center gap-2 py-1 px-2 rounded cursor-pointer
          hover:bg-[var(--bg-tertiary)] transition-colors
          ${node.isStructural ? "opacity-50" : ""}`}
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
      >
        {hasChildren ? (
          <button
            onClick={handleToggle}
            className="w-4 text-xs text-[var(--text-secondary)]"
            aria-label={expanded ? "Collapse" : "Expand"}
          >
            {expanded ? "\u25BC" : "\u25B6"}
          </button>
        ) : (
          <span className="w-4" />
        )}

        {!node.isStructural && statusIcon && (
          <span className={`text-xs ${reviewStatus === "reviewed" ? "text-green-400" : "text-[var(--text-faint)]"}`}>
            {statusIcon}
          </span>
        )}

        <button
          onClick={() => onSelect(node.item.id)}
          className="flex-1 text-left flex items-center gap-2 min-w-0"
        >
          <span className="text-xs font-mono text-[var(--text-faint)]">
            #{node.item.display_num}
          </span>
          <span className="text-xs text-[var(--text-primary)] truncate">
            {node.item.title}
          </span>
        </button>

        {node.item.confidence !== null && !node.isStructural && (
          <ConfidenceGauge confidence={node.item.confidence} />
        )}

        {node.item.human_review?.reviewed_at && !node.isStructural && (
          <span className="text-xs text-[var(--text-faint)] shrink-0">
            {new Date(node.item.human_review.reviewed_at).toLocaleDateString()}
          </span>
        )}
      </div>

      {expanded && node.children.map((child) => (
        <TreeNodeRow
          key={child.item.id}
          node={child}
          depth={depth + 1}
          onSelect={onSelect}
        />
      ))}
    </>
  );
}

interface PastReviewsTreeProps {
  items: ReviewTreeItem[];
  onSelect: (id: string) => void;
  onClose: () => void;
  className?: string;
}

export function PastReviewsTree({ items, onSelect, onClose, className = "" }: PastReviewsTreeProps) {
  const tree = useMemo(() => buildReviewTree(items), [items]);

  return (
    <div
      data-testid="past-reviews-tree"
      className={`flex flex-col bg-[var(--bg-secondary)] border border-[var(--border-default)]
        rounded-lg overflow-hidden ${className}`}
    >
      <div className="flex items-center justify-between px-4 py-3
        border-b border-[var(--border-default)]">
        <h3 className="text-sm font-semibold text-[var(--text-primary)]">
          Past Reviews
        </h3>
        <button
          onClick={onClose}
          aria-label="Close past reviews"
          className="text-[var(--text-secondary)] hover:text-[var(--text-primary)]
            transition-colors text-sm"
        >
          Esc
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-2">
        {tree.length === 0 ? (
          <p className="text-xs text-[var(--text-secondary)] p-4 text-center">
            No reviewed items yet
          </p>
        ) : (
          tree.map((node) => (
            <TreeNodeRow
              key={node.item.id}
              node={node}
              depth={0}
              onSelect={onSelect}
            />
          ))
        )}
      </div>
    </div>
  );
}
