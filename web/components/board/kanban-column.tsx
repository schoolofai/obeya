"use client";

import { Droppable, Draggable } from "@hello-pangea/dnd";
import { KanbanCard } from "./kanban-card";
import type { BoardItem, BoardColumn } from "@/lib/api-client";

interface KanbanColumnProps {
  column: BoardColumn;
  items: BoardItem[];
  onCardClick: (item: BoardItem) => void;
}

export function KanbanColumn({ column, items, onCardClick }: KanbanColumnProps) {
  const isOverLimit = column.limit > 0 && items.length > column.limit;
  const isAtLimit = column.limit > 0 && items.length === column.limit;

  return (
    <div className="flex flex-col w-72 shrink-0">
      <div className="flex items-center justify-between mb-3 px-1">
        <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">
          {column.name}
        </h3>
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-gray-500">{items.length}</span>
          {column.limit > 0 && (
            <span
              className={`text-xs px-1.5 py-0.5 rounded ${
                isOverLimit
                  ? "bg-red-100 text-red-700"
                  : isAtLimit
                    ? "bg-yellow-100 text-yellow-700"
                    : "bg-gray-100 text-gray-500"
              }`}
            >
              / {column.limit}
            </span>
          )}
        </div>
      </div>
      <Droppable droppableId={column.name}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={`flex-1 space-y-2 p-2 rounded-lg min-h-[200px] transition-colors ${
              snapshot.isDraggingOver ? "bg-blue-50" : "bg-gray-50"
            }`}
          >
            {items.map((item, index) => (
              <Draggable key={item.$id} draggableId={item.$id} index={index}>
                {(dragProvided) => (
                  <div
                    ref={dragProvided.innerRef}
                    {...dragProvided.draggableProps}
                    {...dragProvided.dragHandleProps}
                  >
                    <KanbanCard item={item} onClick={() => onCardClick(item)} />
                  </div>
                )}
              </Draggable>
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </div>
  );
}
