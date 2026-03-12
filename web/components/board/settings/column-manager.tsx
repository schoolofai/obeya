"use client";

import { useState } from "react";
import type { BoardColumn } from "@/lib/api-client";

interface ColumnManagerProps {
  boardId: string;
  columns: BoardColumn[];
  onUpdate: (columns: BoardColumn[]) => void;
}

export function ColumnManager({ boardId, columns: initial, onUpdate }: ColumnManagerProps) {
  const [columns, setColumns] = useState<BoardColumn[]>(initial);
  const [newName, setNewName] = useState("");
  const [saving, setSaving] = useState(false);

  async function handleAddColumn() {
    if (!newName.trim()) return;
    const updated = [...columns, { name: newName.trim(), limit: 0 }];
    setColumns(updated);
    setNewName("");
    await saveColumns(updated);
  }

  async function handleRemoveColumn(index: number) {
    const updated = columns.filter((_, i) => i !== index);
    setColumns(updated);
    await saveColumns(updated);
  }

  async function handleLimitChange(index: number, limit: number) {
    const updated = columns.map((col, i) => i === index ? { ...col, limit } : col);
    setColumns(updated);
    await saveColumns(updated);
  }

  async function saveColumns(updated: BoardColumn[]) {
    setSaving(true);
    try {
      const res = await fetch(`/api/boards/${boardId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ columns: JSON.stringify(updated) }),
      });
      if (!res.ok) {
        throw new Error("Failed to update columns");
      }
      onUpdate(updated);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-900 mb-3">Columns</h3>
      <ul className="space-y-2 mb-4">
        {columns.map((col, i) => (
          <li key={col.name} className="flex items-center gap-3">
            <span className="text-sm text-gray-700 flex-1">{col.name}</span>
            <label className="text-xs text-gray-500">
              WIP:
              <input type="number" min={0} value={col.limit}
                onChange={(e) => handleLimitChange(i, Number(e.target.value))}
                className="ml-1 w-16 border rounded px-2 py-1 text-sm" />
            </label>
            <button onClick={() => handleRemoveColumn(i)} className="text-xs text-red-600 hover:text-red-800">Remove</button>
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <input type="text" value={newName} onChange={(e) => setNewName(e.target.value)}
          placeholder="New column name" className="border rounded px-3 py-1.5 text-sm flex-1" />
        <button onClick={handleAddColumn} disabled={saving || !newName.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50">Add</button>
      </div>
    </div>
  );
}
