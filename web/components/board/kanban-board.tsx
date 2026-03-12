"use client";

import { useState, useCallback } from "react";
import { DragDropContext, type DropResult } from "@hello-pangea/dnd";
import { KanbanColumn } from "./kanban-column";
import { ItemDetailPanel } from "./item-detail-panel";
import type { BoardItem, BoardColumn, Board } from "@/lib/api-client";

interface KanbanBoardProps {
  board: Board;
  items: BoardItem[];
}

export function KanbanBoard({ board, items: initialItems }: KanbanBoardProps) {
  const [items, setItems] = useState<BoardItem[]>(initialItems);
  const [selectedItem, setSelectedItem] = useState<BoardItem | null>(null);
  const columns: BoardColumn[] = JSON.parse(board.columns);

  const itemsByColumn = useCallback(
    (columnName: string) =>
      items.filter((item) => item.status === columnName),
    [items]
  );

  const revertMove = useCallback(
    (draggableId: string, originalStatus: string) => {
      setItems((prev) =>
        prev.map((i) =>
          i.$id === draggableId ? { ...i, status: originalStatus } : i
        )
      );
    },
    []
  );

  const applyMove = useCallback(
    (draggableId: string, newStatus: string) => {
      setItems((prev) =>
        prev.map((i) =>
          i.$id === draggableId ? { ...i, status: newStatus } : i
        )
      );
    },
    []
  );

  const handleDragEnd = useCallback(
    async (result: DropResult) => {
      const { draggableId, destination } = result;
      if (!destination) return;

      const newStatus = destination.droppableId;
      const item = items.find((i) => i.$id === draggableId);
      if (!item || item.status === newStatus) return;

      applyMove(draggableId, newStatus);

      const response = await fetch(`/api/items/${draggableId}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: newStatus }),
      });

      if (!response.ok) {
        revertMove(draggableId, item.status);
      }
    },
    [items, applyMove, revertMove]
  );

  const handleCardClick = useCallback((item: BoardItem) => {
    setSelectedItem(item);
  }, []);

  const handlePanelClose = useCallback(() => {
    setSelectedItem(null);
  }, []);

  const handleItemUpdate = useCallback((updated: BoardItem) => {
    setItems((prev) =>
      prev.map((i) => (i.$id === updated.$id ? updated : i))
    );
    setSelectedItem(updated);
  }, []);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 overflow-x-auto">
        <DragDropContext onDragEnd={handleDragEnd}>
          <div className="flex gap-4 p-6 h-full">
            {columns.map((column) => (
              <KanbanColumn
                key={column.name}
                column={column}
                items={itemsByColumn(column.name)}
                onCardClick={handleCardClick}
              />
            ))}
          </div>
        </DragDropContext>
      </div>
      {selectedItem && (
        <ItemDetailPanel
          item={selectedItem}
          boardId={board.$id}
          onClose={handlePanelClose}
          onUpdate={handleItemUpdate}
        />
      )}
    </div>
  );
}
