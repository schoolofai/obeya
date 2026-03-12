"use client";

import Link from "next/link";
import type { Board } from "@/lib/types";

interface BoardCardProps {
  board: Board;
}

export function BoardCard({ board }: BoardCardProps) {
  return (
    <Link
      href={`/boards/${board.id}`}
      className="block rounded-lg border border-gray-200 bg-white p-4 transition-shadow hover:shadow-md"
    >
      <h3 className="font-semibold text-gray-900">{board.name}</h3>
      <div className="mt-2 flex items-center justify-between text-sm text-gray-500">
        <span>{formatItemCount(board.item_count)}</span>
        <span>{formatRelativeTime(board.updated_at)}</span>
      </div>
    </Link>
  );
}

function formatItemCount(count: number): string {
  return count === 1 ? "1 item" : `${count} items`;
}

function formatRelativeTime(updatedAt: string): string {
  const diffMs = Date.now() - new Date(updatedAt).getTime();
  const diffMins = Math.floor(diffMs / 60_000);

  if (diffMins < 60) return `Updated ${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `Updated ${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `Updated ${diffDays}d ago`;
}
