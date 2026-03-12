"use client";
import { useState } from "react";
import type { Board } from "@/lib/api-client";

interface OrgBoardListProps {
  orgId: string;
  boards: Board[];
}

export function OrgBoardList({ orgId, boards: initial }: OrgBoardListProps) {
  const [boards, setBoards] = useState<Board[]>(initial);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");

  async function handleCreateBoard() {
    if (!newName.trim()) return;
    setCreating(true);
    const res = await fetch("/api/boards", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: newName.trim(), org_id: orgId }),
    });
    const body = await res.json();
    if (!body.ok) throw new Error(body.error?.message ?? "Failed to create board");
    setBoards((prev) => [...prev, body.data]);
    setNewName("");
    setCreating(false);
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-gray-900">Boards</h3>
      </div>
      {boards.length === 0 ? (
        <p className="text-sm text-gray-400 py-4">No boards yet.</p>
      ) : (
        <ul className="space-y-2 mb-4">
          {boards.map((board) => (
            <li key={board.$id}>
              <a
                href={`/boards/${board.$id}`}
                className="flex items-center justify-between p-3 rounded-lg border border-gray-200 hover:border-indigo-300 hover:bg-indigo-50 transition-colors"
              >
                <span className="text-sm font-medium text-gray-900">{board.name}</span>
                <span className="text-xs text-gray-400">{new Date(board.updated_at).toLocaleDateString()}</span>
              </a>
            </li>
          ))}
        </ul>
      )}
      <div className="flex gap-2 mt-4">
        <input
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="New board name"
          className="border rounded px-3 py-1.5 text-sm flex-1"
        />
        <button
          onClick={handleCreateBoard}
          disabled={creating || !newName.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          {creating ? "Creating..." : "New Board"}
        </button>
      </div>
    </div>
  );
}
