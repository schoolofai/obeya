"use client";

import { useState, useCallback, useMemo } from "react";
import { DragDropContext, type DropResult } from "@hello-pangea/dnd";
import { KanbanColumn } from "./kanban-column";
import { ItemDetailPanel } from "./item-detail-panel";
import { ReviewQueuePanel } from "./review-queue-panel";
import {
  isHiddenByCollapse,
  orderItemsHierarchically,
} from "@/lib/hierarchy";
import type { BoardItem, BoardColumn, Board } from "@/lib/api-client";

interface KanbanBoardProps {
  board: Board;
  items: BoardItem[];
}

export function KanbanBoard({ board, items: initialItems }: KanbanBoardProps) {
  const [items, setItems] = useState<BoardItem[]>(initialItems);
  const [selectedItem, setSelectedItem] = useState<BoardItem | null>(null);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const columns: BoardColumn[] = JSON.parse(board.columns);

  const allItemsMap = useMemo(() => {
    const map: Record<string, BoardItem> = {};
    for (const item of items) {
      map[item.$id] = item;
    }
    return map;
  }, [items]);

  const visibleColumnItems = useCallback(
    (columnName: string) => {
      const colItems = items.filter((item) => item.status === columnName);
      const visible = colItems.filter(
        (item) => !isHiddenByCollapse(allItemsMap, item, collapsed)
      );
      return orderItemsHierarchically(allItemsMap, visible);
    },
    [items, allItemsMap, collapsed]
  );

  const handleToggleCollapse = useCallback((itemId: string) => {
    setCollapsed((prev) => ({ ...prev, [itemId]: !prev[itemId] }));
  }, []);

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

  const handleMarkReviewed = useCallback(
    async (item: BoardItem) => {
      const response = await fetch(
        `/api/boards/${board.$id}/items/${item.display_num}/review`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status: "reviewed" }),
        }
      );
      if (!response.ok) return;
      const body = await response.json();
      if (body.ok) {
        const updated: BoardItem = { ...item, human_review: body.data.human_review };
        handleItemUpdate(updated);
      }
    },
    [board.$id, handleItemUpdate]
  );

  const handleHide = useCallback(
    async (item: BoardItem) => {
      const response = await fetch(
        `/api/boards/${board.$id}/items/${item.display_num}/review`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status: "hidden" }),
        }
      );
      if (!response.ok) return;
      const body = await response.json();
      if (body.ok) {
        const updated: BoardItem = { ...item, human_review: body.data.human_review };
        handleItemUpdate(updated);
      }
    },
    [board.$id, handleItemUpdate]
  );

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 overflow-x-auto">
        <DragDropContext onDragEnd={handleDragEnd}>
          <div className="flex gap-4 p-6 h-full">
            {columns.map((column) => (
              <KanbanColumn
                key={column.name}
                column={column}
                items={visibleColumnItems(column.name)}
                allItems={allItemsMap}
                collapsed={collapsed}
                onToggleCollapse={handleToggleCollapse}
                onCardClick={handleCardClick}
              />
            ))}
            <ReviewQueuePanel
              items={items}
              onCardClick={handleCardClick}
              onMarkReviewed={handleMarkReviewed}
              onHide={handleHide}
            />
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
