# Obeya Cloud Web UI — Implementation Plan (Part B)

> Continuation of Part A. See `2026-03-12-obeya-cloud-web-ui.md` for shared components, auth pages, and dashboard.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Kanban board view, board settings, activity feed, org management pages, and user settings pages for Obeya Cloud.

**Architecture:** Next.js 15 App Router. Server Components fetch data; Client Components handle interactivity (drag-and-drop, forms, slide-out panels). Tailwind CSS for styling. `@hello-pangea/dnd` for drag-and-drop. All API calls go through Next.js API routes (built in prior plans).

**Tech Stack:** Next.js 15, TypeScript, React 19, Tailwind CSS, `@hello-pangea/dnd`, Vitest, React Testing Library

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md`

**Repository:** `~/code/obeya-cloud`

**Dependencies:** Part A (Web UI shared components, auth pages, dashboard) must be complete. Plan 1 (Foundation), Plan 2 (Boards & Items), Plan 3 (Orgs & Sharing) API routes must exist.

---

## File Structure

```
obeya-cloud/
├── app/(dashboard)/
│   ├── boards/[id]/
│   │   ├── page.tsx                         # Kanban board page (Server Component)
│   │   ├── settings/
│   │   │   └── page.tsx                     # Board settings page
│   │   └── activity/
│   │       └── page.tsx                     # Board activity page
│   ├── orgs/
│   │   ├── new/
│   │   │   └── page.tsx                     # Create org page
│   │   └── [id]/
│   │       ├── page.tsx                     # Org dashboard
│   │       ├── members/
│   │       │   └── page.tsx                 # Org members page
│   │       └── settings/
│   │           └── page.tsx                 # Org settings page
│   └── settings/
│       └── page.tsx                         # User settings page
├── components/
│   ├── board/
│   │   ├── kanban-board.tsx                 # KanbanBoard client component
│   │   ├── kanban-column.tsx                # Single column with WIP indicator
│   │   ├── kanban-card.tsx                  # Card: type icon, #N, title, priority, assignee
│   │   └── item-detail-panel.tsx            # Slide-out detail panel
│   ├── board/settings/
│   │   ├── column-manager.tsx               # Column add/remove/reorder
│   │   └── member-manager.tsx               # Board member management
│   ├── activity/
│   │   └── activity-feed.tsx                # Activity feed with filters
│   ├── org/
│   │   ├── create-org-form.tsx              # Create org form
│   │   ├── org-board-list.tsx               # Org boards list
│   │   ├── org-member-list.tsx              # Org member management
│   │   └── org-settings-form.tsx            # Org settings form
│   └── settings/
│       ├── profile-form.tsx                 # Profile editing
│       └── api-token-manager.tsx            # API token create/list/revoke
├── lib/
│   └── api-client.ts                        # Typed fetch wrapper for API routes
└── __tests__/
    └── components/
        ├── board/
        │   ├── kanban-board.test.tsx
        │   ├── kanban-card.test.tsx
        │   └── item-detail-panel.test.tsx
        ├── activity/
        │   └── activity-feed.test.tsx
        └── settings/
            └── api-token-manager.test.tsx
```

---

## Chunk 4: Kanban Board

### Task 7: Kanban Board Page & Core Components

**Files:**
- Create: `obeya-cloud/app/(dashboard)/boards/[id]/page.tsx`
- Create: `obeya-cloud/components/board/kanban-board.tsx`
- Create: `obeya-cloud/components/board/kanban-column.tsx`
- Create: `obeya-cloud/components/board/kanban-card.tsx`
- Test: `obeya-cloud/__tests__/components/board/kanban-card.test.tsx`

- [ ] **Step 1: Install drag-and-drop dependency**

```bash
cd ~/code/obeya-cloud
npm install @hello-pangea/dnd
npm install -D @types/react @testing-library/react @testing-library/jest-dom
```

- [ ] **Step 2: Write KanbanCard test**

Create: `obeya-cloud/__tests__/components/board/kanban-card.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KanbanCard } from "@/components/board/kanban-card";

const mockItem = {
  $id: "item1",
  display_num: 42,
  type: "task" as const,
  title: "Fix the login bug",
  priority: "high" as const,
  assignee_id: "user1",
  status: "in-progress",
  blocked_by: [],
};

describe("KanbanCard", () => {
  it("renders display number and title", () => {
    render(
      <KanbanCard item={mockItem} onClick={() => {}} />
    );

    expect(screen.getByText("#42")).toBeDefined();
    expect(screen.getByText("Fix the login bug")).toBeDefined();
  });

  it("shows priority badge", () => {
    render(
      <KanbanCard item={mockItem} onClick={() => {}} />
    );

    expect(screen.getByText("high")).toBeDefined();
  });

  it("shows type icon text", () => {
    render(
      <KanbanCard item={mockItem} onClick={() => {}} />
    );

    // Task type should render a task indicator
    expect(screen.getByTestId("type-icon")).toBeDefined();
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/board/kanban-card.test.tsx
```

Expected: FAIL — module not found

- [ ] **Step 4: Write KanbanCard component**

Create: `obeya-cloud/components/board/kanban-card.tsx`

```typescript
"use client";

import type { BoardItem } from "@/lib/api-client";

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

export function KanbanCard({ item, onClick }: KanbanCardProps) {
  const isBlocked = item.blocked_by.length > 0;

  return (
    <button
      onClick={onClick}
      className={`w-full text-left rounded-lg border p-3 shadow-sm
        hover:shadow-md transition-shadow bg-white
        ${isBlocked ? "border-red-300 bg-red-50" : "border-gray-200"}`}
    >
      <div className="flex items-center gap-2 mb-1">
        <span
          data-testid="type-icon"
          className={`text-xs font-bold px-1.5 py-0.5 rounded ${TYPE_COLORS[item.type]}`}
        >
          {TYPE_ICONS[item.type]}
        </span>
        <span className="text-xs text-gray-500 font-mono">
          #{item.display_num}
        </span>
      </div>

      <p className="text-sm font-medium text-gray-900 line-clamp-2">
        {item.title}
      </p>

      <div className="flex items-center justify-between mt-2">
        <span
          className={`text-xs px-2 py-0.5 rounded-full font-medium ${PRIORITY_COLORS[item.priority]}`}
        >
          {item.priority}
        </span>

        {item.assignee_id && (
          <div className="w-6 h-6 rounded-full bg-indigo-500 flex items-center justify-center">
            <span className="text-xs text-white font-medium">
              {item.assignee_id.charAt(0).toUpperCase()}
            </span>
          </div>
        )}
      </div>

      {isBlocked && (
        <div className="mt-2 text-xs text-red-600 font-medium">
          Blocked
        </div>
      )}
    </button>
  );
}
```

- [ ] **Step 5: Write KanbanColumn component**

Create: `obeya-cloud/components/board/kanban-column.tsx`

```typescript
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
                    <KanbanCard
                      item={item}
                      onClick={() => onCardClick(item)}
                    />
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
```

- [ ] **Step 6: Write KanbanBoard client component**

Create: `obeya-cloud/components/board/kanban-board.tsx`

```typescript
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

  const handleDragEnd = useCallback(
    async (result: DropResult) => {
      const { draggableId, destination } = result;
      if (!destination) return;

      const newStatus = destination.droppableId;
      const item = items.find((i) => i.$id === draggableId);
      if (!item || item.status === newStatus) return;

      // Optimistic update
      setItems((prev) =>
        prev.map((i) =>
          i.$id === draggableId ? { ...i, status: newStatus } : i
        )
      );

      const response = await fetch(`/api/items/${draggableId}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: newStatus }),
      });

      if (!response.ok) {
        // Revert on failure
        setItems((prev) =>
          prev.map((i) =>
            i.$id === draggableId ? { ...i, status: item.status } : i
          )
        );
      }
    },
    [items]
  );

  const handleCardClick = useCallback((item: BoardItem) => {
    setSelectedItem(item);
  }, []);

  const handlePanelClose = useCallback(() => {
    setSelectedItem(null);
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
          onUpdate={(updated) => {
            setItems((prev) =>
              prev.map((i) => (i.$id === updated.$id ? updated : i))
            );
            setSelectedItem(updated);
          }}
        />
      )}
    </div>
  );
}
```

- [ ] **Step 7: Run test to verify KanbanCard passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/board/kanban-card.test.tsx
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
cd ~/code/obeya-cloud
git add components/board/kanban-card.tsx components/board/kanban-column.tsx components/board/kanban-board.tsx __tests__/components/board/kanban-card.test.tsx
git commit -m "feat: add KanbanBoard, KanbanColumn, and KanbanCard components"
```

---

### Task 8: Board Page (Server Component)

**Files:**
- Create: `obeya-cloud/app/(dashboard)/boards/[id]/page.tsx`
- Create: `obeya-cloud/lib/api-client.ts` (types + fetch helpers)

- [ ] **Step 1: Create API client types**

Create: `obeya-cloud/lib/api-client.ts`

```typescript
export interface Board {
  $id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: string; // JSON string of BoardColumn[]
  created_at: string;
  updated_at: string;
}

export interface BoardColumn {
  name: string;
  limit: number;
}

export interface BoardItem {
  $id: string;
  board_id: string;
  display_num: number;
  type: "epic" | "story" | "task";
  title: string;
  description: string;
  status: string;
  priority: "low" | "medium" | "high" | "critical";
  parent_id: string | null;
  assignee_id: string | null;
  blocked_by: string[];
  tags: string[];
  project: string | null;
  created_at: string;
  updated_at: string;
}

export interface HistoryEntry {
  $id: string;
  item_id: string;
  board_id: string;
  user_id: string;
  action: string;
  detail: string;
  timestamp: string;
}

export interface Org {
  $id: string;
  name: string;
  slug: string;
  owner_id: string;
  plan: "free" | "pro" | "enterprise";
  created_at: string;
}

export interface OrgMember {
  $id: string;
  org_id: string;
  user_id: string;
  role: "owner" | "admin" | "member";
  invited_at: string;
  accepted_at: string | null;
}

export interface BoardMember {
  $id: string;
  board_id: string;
  user_id: string;
  role: "owner" | "editor" | "viewer";
  invited_at: string;
}

export interface ApiToken {
  $id: string;
  name: string;
  scopes: string[];
  last_used_at: string | null;
  expires_at: string | null;
}

interface ApiResponse<T> {
  ok: boolean;
  data: T;
  meta?: { total?: number; page?: number };
  error?: { code: string; message: string };
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${process.env.NEXT_PUBLIC_APP_URL}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  const body: ApiResponse<T> = await response.json();

  if (!body.ok) {
    throw new Error(body.error?.message ?? "API request failed");
  }

  return body.data;
}
```

- [ ] **Step 2: Write board page (Server Component)**

Create: `obeya-cloud/app/(dashboard)/boards/[id]/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { KanbanBoard } from "@/components/board/kanban-board";
import type { Board, BoardItem } from "@/lib/api-client";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchItems(boardId: string, cookie: string): Promise<BoardItem[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/items`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load items");
  return body.data;
}

export default async function BoardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [board, items] = await Promise.all([
    fetchBoard(id, session),
    fetchItems(id, session),
  ]);

  return (
    <div className="h-full flex flex-col">
      <header className="border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-gray-900">{board.name}</h1>
          <div className="flex items-center gap-3">
            <a
              href={`/boards/${id}/activity`}
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              Activity
            </a>
            <a
              href={`/boards/${id}/settings`}
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              Settings
            </a>
          </div>
        </div>
      </header>

      <KanbanBoard board={board} items={items} />
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya-cloud
git add lib/api-client.ts app/\(dashboard\)/boards/\[id\]/page.tsx
git commit -m "feat: add board page Server Component with API client types"
```

---

### Task 9: Item Detail Panel

**Files:**
- Create: `obeya-cloud/components/board/item-detail-panel.tsx`
- Test: `obeya-cloud/__tests__/components/board/item-detail-panel.test.tsx`

- [ ] **Step 1: Write failing test**

Create: `obeya-cloud/__tests__/components/board/item-detail-panel.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ItemDetailPanel } from "@/components/board/item-detail-panel";

const mockItem = {
  $id: "item1",
  board_id: "board1",
  display_num: 7,
  type: "story" as const,
  title: "User login flow",
  description: "Implement the full login flow with OAuth.",
  status: "in-progress",
  priority: "high" as const,
  parent_id: null,
  assignee_id: "user1",
  blocked_by: ["item2"],
  tags: ["auth"],
  project: null,
  created_at: "2026-03-10T10:00:00Z",
  updated_at: "2026-03-11T14:00:00Z",
};

describe("ItemDetailPanel", () => {
  it("renders item title and description", () => {
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={() => {}}
        onUpdate={() => {}}
      />
    );

    expect(screen.getByText("User login flow")).toBeDefined();
    expect(
      screen.getByText("Implement the full login flow with OAuth.")
    ).toBeDefined();
  });

  it("shows blocked status when blocked_by is non-empty", () => {
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={() => {}}
        onUpdate={() => {}}
      />
    );

    expect(screen.getByText("Blocked by")).toBeDefined();
  });

  it("renders close button", () => {
    const onClose = vi.fn();
    render(
      <ItemDetailPanel
        item={mockItem}
        boardId="board1"
        onClose={onClose}
        onUpdate={() => {}}
      />
    );

    expect(screen.getByLabelText("Close panel")).toBeDefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/board/item-detail-panel.test.tsx
```

Expected: FAIL

- [ ] **Step 3: Write ItemDetailPanel component**

Create: `obeya-cloud/components/board/item-detail-panel.tsx`

```typescript
"use client";

import { useState, useEffect } from "react";
import type { BoardItem, HistoryEntry } from "@/lib/api-client";

interface ItemDetailPanelProps {
  item: BoardItem;
  boardId: string;
  onClose: () => void;
  onUpdate: (item: BoardItem) => void;
}

export function ItemDetailPanel({
  item,
  boardId,
  onClose,
  onUpdate,
}: ItemDetailPanelProps) {
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [subtasks, setSubtasks] = useState<BoardItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDetails();
  }, [item.$id]);

  async function loadDetails() {
    setLoading(true);
    const [historyRes, itemsRes] = await Promise.all([
      fetch(`/api/items/${item.$id}/history`),
      fetch(`/api/boards/${boardId}/items?parent_id=${item.$id}`),
    ]);

    const historyBody = await historyRes.json();
    if (historyBody.ok) setHistory(historyBody.data);

    const itemsBody = await itemsRes.json();
    if (itemsBody.ok) {
      setSubtasks(
        itemsBody.data.filter((i: BoardItem) => i.parent_id === item.$id)
      );
    }

    setLoading(false);
  }

  return (
    <div className="fixed inset-y-0 right-0 w-[480px] bg-white shadow-xl border-l border-gray-200 z-50 flex flex-col">
      <PanelHeader item={item} onClose={onClose} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <DescriptionSection description={item.description} />
        <MetadataSection item={item} />
        {item.blocked_by.length > 0 && (
          <BlockedSection blockedBy={item.blocked_by} />
        )}
        <SubtasksSection subtasks={subtasks} loading={loading} />
        <HistorySection history={history} loading={loading} />
      </div>
    </div>
  );
}

function PanelHeader({
  item,
  onClose,
}: {
  item: BoardItem;
  onClose: () => void;
}) {
  const typeLabel = item.type.charAt(0).toUpperCase() + item.type.slice(1);
  return (
    <div className="flex items-center justify-between p-4 border-b border-gray-200">
      <div>
        <span className="text-xs text-gray-500 font-mono">
          #{item.display_num} &middot; {typeLabel}
        </span>
        <h2 className="text-lg font-semibold text-gray-900">{item.title}</h2>
      </div>
      <button
        onClick={onClose}
        aria-label="Close panel"
        className="p-1 rounded hover:bg-gray-100 text-gray-500"
      >
        <XIcon />
      </button>
    </div>
  );
}

function DescriptionSection({ description }: { description: string }) {
  if (!description) {
    return (
      <p className="text-sm text-gray-400 italic">No description provided.</p>
    );
  }
  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-1">Description</h3>
      <p className="text-sm text-gray-600 whitespace-pre-wrap">{description}</p>
    </div>
  );
}

function MetadataSection({ item }: { item: BoardItem }) {
  return (
    <div className="grid grid-cols-2 gap-3 text-sm">
      <MetaField label="Status" value={item.status} />
      <MetaField label="Priority" value={item.priority} />
      <MetaField label="Assignee" value={item.assignee_id ?? "Unassigned"} />
      <MetaField label="Created" value={formatDate(item.created_at)} />
    </div>
  );
}

function MetaField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-gray-500">{label}</span>
      <p className="font-medium text-gray-900">{value}</p>
    </div>
  );
}

function BlockedSection({ blockedBy }: { blockedBy: string[] }) {
  return (
    <div className="bg-red-50 border border-red-200 rounded-lg p-3">
      <h3 className="text-sm font-medium text-red-700 mb-1">Blocked by</h3>
      <ul className="text-sm text-red-600 space-y-1">
        {blockedBy.map((id) => (
          <li key={id} className="font-mono">
            {id}
          </li>
        ))}
      </ul>
    </div>
  );
}

function SubtasksSection({
  subtasks,
  loading,
}: {
  subtasks: BoardItem[];
  loading: boolean;
}) {
  if (loading) return <SectionSkeleton label="Subtasks" />;
  if (subtasks.length === 0) return null;

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-2">Subtasks</h3>
      <ul className="space-y-1">
        {subtasks.map((sub) => (
          <li
            key={sub.$id}
            className="text-sm flex items-center gap-2 text-gray-700"
          >
            <span className="font-mono text-gray-500">#{sub.display_num}</span>
            <span>{sub.title}</span>
            <span className="ml-auto text-xs text-gray-400">{sub.status}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function HistorySection({
  history,
  loading,
}: {
  history: HistoryEntry[];
  loading: boolean;
}) {
  if (loading) return <SectionSkeleton label="History" />;

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-2">History</h3>
      {history.length === 0 ? (
        <p className="text-sm text-gray-400">No history yet.</p>
      ) : (
        <ul className="space-y-3">
          {history.map((entry) => (
            <li key={entry.$id} className="flex gap-3 text-sm">
              <div className="w-2 h-2 mt-1.5 rounded-full bg-gray-400 shrink-0" />
              <div>
                <p className="text-gray-700">{entry.detail}</p>
                <p className="text-xs text-gray-400">
                  {formatDate(entry.timestamp)}
                </p>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function SectionSkeleton({ label }: { label: string }) {
  return (
    <div>
      <h3 className="text-sm font-medium text-gray-700 mb-2">{label}</h3>
      <div className="animate-pulse space-y-2">
        <div className="h-4 bg-gray-200 rounded w-3/4" />
        <div className="h-4 bg-gray-200 rounded w-1/2" />
      </div>
    </div>
  );
}

function XIcon() {
  return (
    <svg
      className="w-5 h-5"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M6 18L18 6M6 6l12 12"
      />
    </svg>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/board/item-detail-panel.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add components/board/item-detail-panel.tsx __tests__/components/board/item-detail-panel.test.tsx
git commit -m "feat: add ItemDetailPanel with description, history, subtasks, and block status"
```

---

## Chunk 5: Board Settings & Activity

### Task 10: Board Settings Page

**Files:**
- Create: `obeya-cloud/app/(dashboard)/boards/[id]/settings/page.tsx`
- Create: `obeya-cloud/components/board/settings/column-manager.tsx`
- Create: `obeya-cloud/components/board/settings/member-manager.tsx`

- [ ] **Step 1: Write ColumnManager component**

Create: `obeya-cloud/components/board/settings/column-manager.tsx`

```typescript
"use client";

import { useState } from "react";
import type { BoardColumn } from "@/lib/api-client";

interface ColumnManagerProps {
  boardId: string;
  columns: BoardColumn[];
  onUpdate: (columns: BoardColumn[]) => void;
}

export function ColumnManager({
  boardId,
  columns: initial,
  onUpdate,
}: ColumnManagerProps) {
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
    const updated = columns.map((col, i) =>
      i === index ? { ...col, limit } : col
    );
    setColumns(updated);
    await saveColumns(updated);
  }

  async function saveColumns(updated: BoardColumn[]) {
    setSaving(true);
    const res = await fetch(`/api/boards/${boardId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ columns: JSON.stringify(updated) }),
    });

    if (!res.ok) {
      throw new Error("Failed to update columns");
    }

    onUpdate(updated);
    setSaving(false);
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
              <input
                type="number"
                min={0}
                value={col.limit}
                onChange={(e) => handleLimitChange(i, Number(e.target.value))}
                className="ml-1 w-16 border rounded px-2 py-1 text-sm"
              />
            </label>
            <button
              onClick={() => handleRemoveColumn(i)}
              className="text-xs text-red-600 hover:text-red-800"
            >
              Remove
            </button>
          </li>
        ))}
      </ul>

      <div className="flex gap-2">
        <input
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="New column name"
          className="border rounded px-3 py-1.5 text-sm flex-1"
        />
        <button
          onClick={handleAddColumn}
          disabled={saving || !newName.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          Add
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Write MemberManager component**

Create: `obeya-cloud/components/board/settings/member-manager.tsx`

```typescript
"use client";

import { useState } from "react";
import type { BoardMember } from "@/lib/api-client";

interface MemberManagerProps {
  boardId: string;
  members: BoardMember[];
  currentUserId: string;
}

export function MemberManager({
  boardId,
  members: initial,
  currentUserId,
}: MemberManagerProps) {
  const [members, setMembers] = useState<BoardMember[]>(initial);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"editor" | "viewer">("editor");
  const [inviting, setInviting] = useState(false);

  async function handleInvite() {
    if (!email.trim()) return;
    setInviting(true);

    const res = await fetch(`/api/boards/${boardId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email.trim(), role }),
    });

    if (!res.ok) {
      const body = await res.json();
      throw new Error(body.error?.message ?? "Failed to invite member");
    }

    const body = await res.json();
    setMembers((prev) => [...prev, body.data]);
    setEmail("");
    setInviting(false);
  }

  async function handleRoleChange(memberId: string, newRole: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;

    const res = await fetch(
      `/api/boards/${boardId}/members/${member.user_id}`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role: newRole }),
      }
    );

    if (!res.ok) {
      throw new Error("Failed to update member role");
    }

    setMembers((prev) =>
      prev.map((m) =>
        m.$id === memberId ? { ...m, role: newRole as BoardMember["role"] } : m
      )
    );
  }

  async function handleRemove(memberId: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;

    const res = await fetch(
      `/api/boards/${boardId}/members/${member.user_id}`,
      { method: "DELETE" }
    );

    if (!res.ok) {
      throw new Error("Failed to remove member");
    }

    setMembers((prev) => prev.filter((m) => m.$id !== memberId));
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-900 mb-3">Members</h3>
      <ul className="space-y-2 mb-4">
        {members.map((member) => (
          <li key={member.$id} className="flex items-center gap-3">
            <span className="text-sm text-gray-700 flex-1 font-mono">
              {member.user_id}
            </span>
            <select
              value={member.role}
              onChange={(e) => handleRoleChange(member.$id, e.target.value)}
              disabled={member.user_id === currentUserId}
              className="text-sm border rounded px-2 py-1"
            >
              <option value="owner">Owner</option>
              <option value="editor">Editor</option>
              <option value="viewer">Viewer</option>
            </select>
            {member.user_id !== currentUserId && (
              <button
                onClick={() => handleRemove(member.$id)}
                className="text-xs text-red-600 hover:text-red-800"
              >
                Remove
              </button>
            )}
          </li>
        ))}
      </ul>

      <div className="flex gap-2">
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="Email address"
          className="border rounded px-3 py-1.5 text-sm flex-1"
        />
        <select
          value={role}
          onChange={(e) => setRole(e.target.value as "editor" | "viewer")}
          className="border rounded px-2 py-1 text-sm"
        >
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
        <button
          onClick={handleInvite}
          disabled={inviting || !email.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          Invite
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Write board settings page**

Create: `obeya-cloud/app/(dashboard)/boards/[id]/settings/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ColumnManager } from "@/components/board/settings/column-manager";
import { MemberManager } from "@/components/board/settings/member-manager";
import type { Board, BoardColumn, BoardMember } from "@/lib/api-client";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchMembers(
  boardId: string,
  cookie: string
): Promise<BoardMember[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/members`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) return [];
  return body.data;
}

async function fetchCurrentUserId(cookie: string): Promise<string> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/auth/me`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error("Not authenticated");
  return body.data.id;
}

export default async function BoardSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [board, members, userId] = await Promise.all([
    fetchBoard(id, session),
    fetchMembers(id, session),
    fetchCurrentUserId(session),
  ]);

  const columns: BoardColumn[] = JSON.parse(board.columns);

  return (
    <div className="max-w-2xl mx-auto p-6">
      <div className="mb-6">
        <a
          href={`/boards/${id}`}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          &larr; Back to board
        </a>
      </div>

      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        Board Settings &mdash; {board.name}
      </h1>

      <div className="space-y-8">
        <section className="bg-white border border-gray-200 rounded-lg p-6">
          <ColumnManager
            boardId={id}
            columns={columns}
            onUpdate={() => {}}
          />
        </section>

        <section className="bg-white border border-gray-200 rounded-lg p-6">
          <MemberManager
            boardId={id}
            members={members}
            currentUserId={userId}
          />
        </section>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Commit**

```bash
cd ~/code/obeya-cloud
git add components/board/settings/ app/\(dashboard\)/boards/\[id\]/settings/
git commit -m "feat: add board settings page with column and member management"
```

---

### Task 11: Board Activity Page

**Files:**
- Create: `obeya-cloud/app/(dashboard)/boards/[id]/activity/page.tsx`
- Create: `obeya-cloud/components/activity/activity-feed.tsx`
- Test: `obeya-cloud/__tests__/components/activity/activity-feed.test.tsx`

- [ ] **Step 1: Write failing test for ActivityFeed**

Create: `obeya-cloud/__tests__/components/activity/activity-feed.test.tsx`

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ActivityFeed } from "@/components/activity/activity-feed";

const mockEntries = [
  {
    $id: "h1",
    item_id: "item1",
    board_id: "board1",
    user_id: "user1",
    action: "created",
    detail: "Created task #1: Fix login bug",
    timestamp: "2026-03-11T10:00:00Z",
  },
  {
    $id: "h2",
    item_id: "item1",
    board_id: "board1",
    user_id: "user2",
    action: "moved",
    detail: "status: todo -> in-progress",
    timestamp: "2026-03-11T11:30:00Z",
  },
];

describe("ActivityFeed", () => {
  it("renders activity entries", () => {
    render(<ActivityFeed entries={mockEntries} />);

    expect(
      screen.getByText("Created task #1: Fix login bug")
    ).toBeDefined();
    expect(
      screen.getByText("status: todo -> in-progress")
    ).toBeDefined();
  });

  it("shows action type badges", () => {
    render(<ActivityFeed entries={mockEntries} />);

    expect(screen.getByText("created")).toBeDefined();
    expect(screen.getByText("moved")).toBeDefined();
  });

  it("renders empty state when no entries", () => {
    render(<ActivityFeed entries={[]} />);

    expect(screen.getByText("No activity yet.")).toBeDefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/activity/activity-feed.test.tsx
```

Expected: FAIL

- [ ] **Step 3: Write ActivityFeed component**

Create: `obeya-cloud/components/activity/activity-feed.tsx`

```typescript
"use client";

import { useState } from "react";
import type { HistoryEntry } from "@/lib/api-client";

const ACTION_COLORS: Record<string, string> = {
  created: "bg-green-100 text-green-700",
  moved: "bg-blue-100 text-blue-700",
  edited: "bg-yellow-100 text-yellow-700",
  assigned: "bg-purple-100 text-purple-700",
  blocked: "bg-red-100 text-red-700",
  unblocked: "bg-gray-100 text-gray-700",
};

interface ActivityFeedProps {
  entries: HistoryEntry[];
  filterOptions?: string[];
}

export function ActivityFeed({
  entries,
  filterOptions,
}: ActivityFeedProps) {
  const [filter, setFilter] = useState<string>("all");

  const filtered =
    filter === "all"
      ? entries
      : entries.filter((e) => e.action === filter);

  const actions = filterOptions ?? [
    ...new Set(entries.map((e) => e.action)),
  ];

  return (
    <div>
      {actions.length > 1 && (
        <div className="flex gap-2 mb-4">
          <FilterButton
            label="All"
            active={filter === "all"}
            onClick={() => setFilter("all")}
          />
          {actions.map((action) => (
            <FilterButton
              key={action}
              label={action}
              active={filter === action}
              onClick={() => setFilter(action)}
            />
          ))}
        </div>
      )}

      {filtered.length === 0 ? (
        <p className="text-sm text-gray-400 py-8 text-center">
          No activity yet.
        </p>
      ) : (
        <ul className="space-y-3">
          {filtered.map((entry) => (
            <ActivityEntry key={entry.$id} entry={entry} />
          ))}
        </ul>
      )}
    </div>
  );
}

function ActivityEntry({ entry }: { entry: HistoryEntry }) {
  const colorClass = ACTION_COLORS[entry.action] ?? "bg-gray-100 text-gray-600";

  return (
    <li className="flex items-start gap-3 py-2 border-b border-gray-100 last:border-0">
      <span
        className={`text-xs px-2 py-0.5 rounded-full font-medium mt-0.5 shrink-0 ${colorClass}`}
      >
        {entry.action}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-700">{entry.detail}</p>
        <p className="text-xs text-gray-400 mt-0.5">
          {formatTimestamp(entry.timestamp)}
          {entry.user_id && (
            <span className="ml-2 font-mono">{entry.user_id}</span>
          )}
        </p>
      </div>
    </li>
  );
}

function FilterButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`text-xs px-3 py-1 rounded-full transition-colors ${
        active
          ? "bg-indigo-600 text-white"
          : "bg-gray-100 text-gray-600 hover:bg-gray-200"
      }`}
    >
      {label}
    </button>
  );
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
```

- [ ] **Step 4: Write board activity page**

Create: `obeya-cloud/app/(dashboard)/boards/[id]/activity/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ActivityFeed } from "@/components/activity/activity-feed";
import type { Board, HistoryEntry } from "@/lib/api-client";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchActivity(
  boardId: string,
  cookie: string
): Promise<HistoryEntry[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/activity`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load activity");
  return body.data;
}

export default async function BoardActivityPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [board, activity] = await Promise.all([
    fetchBoard(id, session),
    fetchActivity(id, session),
  ]);

  return (
    <div className="max-w-3xl mx-auto p-6">
      <div className="mb-6">
        <a
          href={`/boards/${id}`}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          &larr; Back to board
        </a>
      </div>

      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        Activity &mdash; {board.name}
      </h1>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <ActivityFeed
          entries={activity}
          filterOptions={[
            "created",
            "moved",
            "edited",
            "assigned",
            "blocked",
            "unblocked",
          ]}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Run test to verify ActivityFeed passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/activity/activity-feed.test.tsx
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya-cloud
git add components/activity/ app/\(dashboard\)/boards/\[id\]/activity/ __tests__/components/activity/
git commit -m "feat: add board activity page with filterable activity feed"
```

---

## Chunk 6: Org Pages

### Task 12: Create Org & Org Dashboard

**Files:**
- Create: `obeya-cloud/app/(dashboard)/orgs/new/page.tsx`
- Create: `obeya-cloud/components/org/create-org-form.tsx`
- Create: `obeya-cloud/app/(dashboard)/orgs/[id]/page.tsx`
- Create: `obeya-cloud/components/org/org-board-list.tsx`

- [ ] **Step 1: Write CreateOrgForm component**

Create: `obeya-cloud/components/org/create-org-form.tsx`

```typescript
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export function CreateOrgForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    setCreating(true);
    setError(null);

    const res = await fetch("/api/orgs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name.trim() }),
    });

    const body = await res.json();

    if (!body.ok) {
      setError(body.error?.message ?? "Failed to create organization");
      setCreating(false);
      return;
    }

    router.push(`/orgs/${body.data.$id}`);
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label
          htmlFor="org-name"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Organization name
        </label>
        <input
          id="org-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My Team"
          className="w-full border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          required
        />
        <p className="text-xs text-gray-500 mt-1">
          A URL-safe slug will be generated automatically.
        </p>
      </div>

      {error && (
        <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">
          {error}
        </div>
      )}

      <button
        type="submit"
        disabled={creating || !name.trim()}
        className="w-full bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
      >
        {creating ? "Creating..." : "Create Organization"}
      </button>
    </form>
  );
}
```

- [ ] **Step 2: Write create org page**

Create: `obeya-cloud/app/(dashboard)/orgs/new/page.tsx`

```typescript
import { CreateOrgForm } from "@/components/org/create-org-form";

export default function CreateOrgPage() {
  return (
    <div className="max-w-md mx-auto p-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        Create Organization
      </h1>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <CreateOrgForm />
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Write OrgBoardList component**

Create: `obeya-cloud/components/org/org-board-list.tsx`

```typescript
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { Board } from "@/lib/api-client";

interface OrgBoardListProps {
  orgId: string;
  boards: Board[];
}

export function OrgBoardList({ orgId, boards: initial }: OrgBoardListProps) {
  const router = useRouter();
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

    if (!body.ok) {
      throw new Error(body.error?.message ?? "Failed to create board");
    }

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
                <span className="text-sm font-medium text-gray-900">
                  {board.name}
                </span>
                <span className="text-xs text-gray-400">
                  {new Date(board.updated_at).toLocaleDateString()}
                </span>
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
```

- [ ] **Step 4: Write org dashboard page**

Create: `obeya-cloud/app/(dashboard)/orgs/[id]/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgBoardList } from "@/components/org/org-board-list";
import type { Org, Board } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load org");
  return body.data;
}

async function fetchOrgBoards(
  orgId: string,
  cookie: string
): Promise<Board[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards?org_id=${orgId}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) return [];
  return body.data;
}

export default async function OrgDashboardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [org, boards] = await Promise.all([
    fetchOrg(id, session),
    fetchOrgBoards(id, session),
  ]);

  return (
    <div className="max-w-3xl mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">{org.name}</h1>
          <p className="text-sm text-gray-500">/{org.slug}</p>
        </div>
        <div className="flex gap-3">
          <a
            href={`/orgs/${id}/members`}
            className="text-sm text-gray-600 hover:text-gray-900"
          >
            Members
          </a>
          <a
            href={`/orgs/${id}/settings`}
            className="text-sm text-gray-600 hover:text-gray-900"
          >
            Settings
          </a>
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <OrgBoardList orgId={id} boards={boards} />
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add components/org/create-org-form.tsx components/org/org-board-list.tsx app/\(dashboard\)/orgs/
git commit -m "feat: add create org page and org dashboard with board list"
```

---

### Task 13: Org Members & Org Settings

**Files:**
- Create: `obeya-cloud/app/(dashboard)/orgs/[id]/members/page.tsx`
- Create: `obeya-cloud/components/org/org-member-list.tsx`
- Create: `obeya-cloud/app/(dashboard)/orgs/[id]/settings/page.tsx`
- Create: `obeya-cloud/components/org/org-settings-form.tsx`

- [ ] **Step 1: Write OrgMemberList component**

Create: `obeya-cloud/components/org/org-member-list.tsx`

```typescript
"use client";

import { useState } from "react";
import type { OrgMember } from "@/lib/api-client";

interface OrgMemberListProps {
  orgId: string;
  members: OrgMember[];
  currentUserId: string;
}

export function OrgMemberList({
  orgId,
  members: initial,
  currentUserId,
}: OrgMemberListProps) {
  const [members, setMembers] = useState<OrgMember[]>(initial);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"admin" | "member">("member");
  const [inviting, setInviting] = useState(false);

  async function handleInvite() {
    if (!email.trim()) return;
    setInviting(true);

    const res = await fetch(`/api/orgs/${orgId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email.trim(), role }),
    });

    const body = await res.json();

    if (!body.ok) {
      throw new Error(body.error?.message ?? "Failed to invite member");
    }

    setMembers((prev) => [...prev, body.data]);
    setEmail("");
    setInviting(false);
  }

  async function handleRoleChange(memberId: string, newRole: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;

    const res = await fetch(`/api/orgs/${orgId}/members/${member.user_id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ role: newRole }),
    });

    if (!res.ok) {
      throw new Error("Failed to update member role");
    }

    setMembers((prev) =>
      prev.map((m) =>
        m.$id === memberId
          ? { ...m, role: newRole as OrgMember["role"] }
          : m
      )
    );
  }

  async function handleRemove(memberId: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;

    const res = await fetch(`/api/orgs/${orgId}/members/${member.user_id}`, {
      method: "DELETE",
    });

    if (!res.ok) {
      throw new Error("Failed to remove member");
    }

    setMembers((prev) => prev.filter((m) => m.$id !== memberId));
  }

  return (
    <div>
      <ul className="space-y-2 mb-6">
        {members.map((member) => (
          <li
            key={member.$id}
            className="flex items-center gap-3 p-3 border border-gray-200 rounded-lg"
          >
            <div className="w-8 h-8 rounded-full bg-indigo-500 flex items-center justify-center shrink-0">
              <span className="text-sm text-white font-medium">
                {member.user_id.charAt(0).toUpperCase()}
              </span>
            </div>
            <span className="text-sm text-gray-700 flex-1 font-mono">
              {member.user_id}
            </span>
            {member.accepted_at === null && (
              <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full">
                Pending
              </span>
            )}
            <select
              value={member.role}
              onChange={(e) => handleRoleChange(member.$id, e.target.value)}
              disabled={member.user_id === currentUserId}
              className="text-sm border rounded px-2 py-1"
            >
              <option value="owner">Owner</option>
              <option value="admin">Admin</option>
              <option value="member">Member</option>
            </select>
            {member.user_id !== currentUserId && (
              <button
                onClick={() => handleRemove(member.$id)}
                className="text-xs text-red-600 hover:text-red-800"
              >
                Remove
              </button>
            )}
          </li>
        ))}
      </ul>

      <div className="flex gap-2">
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="Invite by email"
          className="border rounded px-3 py-1.5 text-sm flex-1"
        />
        <select
          value={role}
          onChange={(e) => setRole(e.target.value as "admin" | "member")}
          className="border rounded px-2 py-1 text-sm"
        >
          <option value="member">Member</option>
          <option value="admin">Admin</option>
        </select>
        <button
          onClick={handleInvite}
          disabled={inviting || !email.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          {inviting ? "Inviting..." : "Invite"}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Write org members page**

Create: `obeya-cloud/app/(dashboard)/orgs/[id]/members/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgMemberList } from "@/components/org/org-member-list";
import type { Org, OrgMember } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load org");
  return body.data;
}

async function fetchMembers(
  orgId: string,
  cookie: string
): Promise<OrgMember[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${orgId}/members`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) return [];
  return body.data;
}

async function fetchCurrentUserId(cookie: string): Promise<string> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/auth/me`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error("Not authenticated");
  return body.data.id;
}

export default async function OrgMembersPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [org, members, userId] = await Promise.all([
    fetchOrg(id, session),
    fetchMembers(id, session),
    fetchCurrentUserId(session),
  ]);

  return (
    <div className="max-w-2xl mx-auto p-6">
      <div className="mb-6">
        <a
          href={`/orgs/${id}`}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          &larr; Back to {org.name}
        </a>
      </div>

      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        Members &mdash; {org.name}
      </h1>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <OrgMemberList
          orgId={id}
          members={members}
          currentUserId={userId}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Write OrgSettingsForm component**

Create: `obeya-cloud/components/org/org-settings-form.tsx`

```typescript
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { Org } from "@/lib/api-client";

interface OrgSettingsFormProps {
  org: Org;
}

export function OrgSettingsForm({ org }: OrgSettingsFormProps) {
  const router = useRouter();
  const [name, setName] = useState(org.name);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    setSaving(true);
    setError(null);

    const res = await fetch(`/api/orgs/${org.$id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name.trim() }),
    });

    const body = await res.json();

    if (!body.ok) {
      setError(body.error?.message ?? "Failed to update organization");
      setSaving(false);
      return;
    }

    setSaving(false);
    router.refresh();
  }

  async function handleDelete() {
    const confirmed = window.confirm(
      `Delete "${org.name}"? This will remove all boards and data. This action cannot be undone.`
    );
    if (!confirmed) return;

    setDeleting(true);

    const res = await fetch(`/api/orgs/${org.$id}`, {
      method: "DELETE",
    });

    if (!res.ok) {
      const body = await res.json();
      setError(body.error?.message ?? "Failed to delete organization");
      setDeleting(false);
      return;
    }

    router.push("/dashboard");
  }

  return (
    <div className="space-y-8">
      <form onSubmit={handleSave} className="space-y-4">
        <div>
          <label
            htmlFor="org-name"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Organization name
          </label>
          <input
            id="org-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Slug
          </label>
          <p className="text-sm text-gray-500 font-mono">{org.slug}</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Plan
          </label>
          <p className="text-sm text-gray-500 capitalize">{org.plan}</p>
        </div>

        {error && (
          <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={saving || !name.trim()}
          className="bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
        >
          {saving ? "Saving..." : "Save Changes"}
        </button>
      </form>

      <div className="border-t border-gray-200 pt-6">
        <h3 className="text-sm font-medium text-red-600 mb-2">Danger Zone</h3>
        <p className="text-sm text-gray-500 mb-3">
          Deleting this organization will permanently remove all associated
          boards and data.
        </p>
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="bg-red-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-red-700 disabled:opacity-50"
        >
          {deleting ? "Deleting..." : "Delete Organization"}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Write org settings page**

Create: `obeya-cloud/app/(dashboard)/orgs/[id]/settings/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgSettingsForm } from "@/components/org/org-settings-form";
import type { Org } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load org");
  return body.data;
}

export default async function OrgSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const org = await fetchOrg(id, session);

  return (
    <div className="max-w-2xl mx-auto p-6">
      <div className="mb-6">
        <a
          href={`/orgs/${id}`}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          &larr; Back to {org.name}
        </a>
      </div>

      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        Settings &mdash; {org.name}
      </h1>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <OrgSettingsForm org={org} />
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya-cloud
git add components/org/org-member-list.tsx components/org/org-settings-form.tsx app/\(dashboard\)/orgs/
git commit -m "feat: add org members page and org settings page"
```

---

## Chunk 7: User Settings

### Task 14: User Settings Page

**Files:**
- Create: `obeya-cloud/app/(dashboard)/settings/page.tsx`
- Create: `obeya-cloud/components/settings/profile-form.tsx`
- Create: `obeya-cloud/components/settings/api-token-manager.tsx`
- Test: `obeya-cloud/__tests__/components/settings/api-token-manager.test.tsx`

- [ ] **Step 1: Write failing test for ApiTokenManager**

Create: `obeya-cloud/__tests__/components/settings/api-token-manager.test.tsx`

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ApiTokenManager } from "@/components/settings/api-token-manager";

const mockTokens = [
  {
    $id: "tok1",
    name: "My laptop",
    scopes: ["*"],
    last_used_at: "2026-03-11T10:00:00Z",
    expires_at: null,
  },
  {
    $id: "tok2",
    name: "CI server",
    scopes: ["boards:read"],
    last_used_at: null,
    expires_at: null,
  },
];

describe("ApiTokenManager", () => {
  it("renders existing tokens", () => {
    render(<ApiTokenManager tokens={mockTokens} />);

    expect(screen.getByText("My laptop")).toBeDefined();
    expect(screen.getByText("CI server")).toBeDefined();
  });

  it("shows revoke button for each token", () => {
    render(<ApiTokenManager tokens={mockTokens} />);

    const revokeButtons = screen.getAllByText("Revoke");
    expect(revokeButtons.length).toBe(2);
  });

  it("renders create token form", () => {
    render(<ApiTokenManager tokens={mockTokens} />);

    expect(screen.getByPlaceholderText("Token name")).toBeDefined();
    expect(screen.getByText("Create Token")).toBeDefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/settings/api-token-manager.test.tsx
```

Expected: FAIL

- [ ] **Step 3: Write ProfileForm component**

Create: `obeya-cloud/components/settings/profile-form.tsx`

```typescript
"use client";

import { useState } from "react";

interface ProfileFormProps {
  user: {
    id: string;
    email: string;
    name: string;
  };
}

export function ProfileForm({ user }: ProfileFormProps) {
  const [name, setName] = useState(user.name);
  const [saving, setSaving] = useState(false);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);

    // Profile update would go through Appwrite Account API
    // For now, show the form structure
    setSaving(false);
  }

  return (
    <form onSubmit={handleSave} className="space-y-4">
      <div>
        <label
          htmlFor="email"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Email
        </label>
        <input
          id="email"
          type="email"
          value={user.email}
          disabled
          className="w-full border border-gray-200 rounded-lg px-4 py-2 text-sm bg-gray-50 text-gray-500"
        />
      </div>

      <div>
        <label
          htmlFor="name"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Display name
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
      </div>

      <button
        type="submit"
        disabled={saving}
        className="bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Update Profile"}
      </button>
    </form>
  );
}
```

- [ ] **Step 4: Write ApiTokenManager component**

Create: `obeya-cloud/components/settings/api-token-manager.tsx`

```typescript
"use client";

import { useState } from "react";
import type { ApiToken } from "@/lib/api-client";

interface ApiTokenManagerProps {
  tokens: ApiToken[];
}

export function ApiTokenManager({
  tokens: initial,
}: ApiTokenManagerProps) {
  const [tokens, setTokens] = useState<ApiToken[]>(initial);
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);

  async function handleCreate() {
    if (!name.trim()) return;
    setCreating(true);
    setNewToken(null);

    const res = await fetch("/api/auth/token", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name.trim() }),
    });

    const body = await res.json();

    if (!body.ok) {
      throw new Error(body.error?.message ?? "Failed to create token");
    }

    setTokens((prev) => [
      ...prev,
      {
        $id: body.data.id,
        name: body.data.name,
        scopes: body.data.scopes,
        last_used_at: null,
        expires_at: null,
      },
    ]);

    setNewToken(body.data.token);
    setName("");
    setCreating(false);
  }

  async function handleRevoke(tokenId: string) {
    const res = await fetch(`/api/auth/token/${tokenId}`, {
      method: "DELETE",
    });

    if (!res.ok) {
      throw new Error("Failed to revoke token");
    }

    setTokens((prev) => prev.filter((t) => t.$id !== tokenId));
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-900 mb-4">API Tokens</h3>

      {newToken && <NewTokenBanner token={newToken} />}

      {tokens.length === 0 ? (
        <p className="text-sm text-gray-400 py-4">No API tokens created.</p>
      ) : (
        <ul className="space-y-2 mb-6">
          {tokens.map((token) => (
            <TokenRow
              key={token.$id}
              token={token}
              onRevoke={() => handleRevoke(token.$id)}
            />
          ))}
        </ul>
      )}

      <div className="flex gap-2">
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Token name"
          className="border rounded px-3 py-1.5 text-sm flex-1"
        />
        <button
          onClick={handleCreate}
          disabled={creating || !name.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
        >
          Create Token
        </button>
      </div>
    </div>
  );
}

function TokenRow({
  token,
  onRevoke,
}: {
  token: ApiToken;
  onRevoke: () => void;
}) {
  return (
    <li className="flex items-center gap-3 p-3 border border-gray-200 rounded-lg">
      <div className="flex-1">
        <p className="text-sm font-medium text-gray-900">{token.name}</p>
        <p className="text-xs text-gray-400">
          {token.last_used_at
            ? `Last used ${formatDate(token.last_used_at)}`
            : "Never used"}
        </p>
      </div>
      <button
        onClick={onRevoke}
        className="text-xs text-red-600 hover:text-red-800 font-medium"
      >
        Revoke
      </button>
    </li>
  );
}

function NewTokenBanner({ token }: { token: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(token);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
      <p className="text-sm font-medium text-green-800 mb-2">
        Token created. Copy it now — it will not be shown again.
      </p>
      <div className="flex items-center gap-2">
        <code className="text-xs bg-green-100 px-3 py-1.5 rounded font-mono flex-1 overflow-x-auto">
          {token}
        </code>
        <button
          onClick={handleCopy}
          className="text-xs bg-green-600 text-white px-3 py-1.5 rounded hover:bg-green-700"
        >
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </div>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
```

- [ ] **Step 5: Write user settings page**

Create: `obeya-cloud/app/(dashboard)/settings/page.tsx`

```typescript
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ProfileForm } from "@/components/settings/profile-form";
import { ApiTokenManager } from "@/components/settings/api-token-manager";
import type { ApiToken } from "@/lib/api-client";

async function fetchUser(cookie: string) {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/auth/me`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error("Not authenticated");
  return body.data;
}

async function fetchTokens(cookie: string): Promise<ApiToken[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/auth/tokens`,
    { headers: { cookie: `a]session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) return [];
  return body.data;
}

export default async function UserSettingsPage() {
  const cookieStore = await cookies();
  const session = cookieStore.get("a]session")?.value;

  if (!session) redirect("/auth/login");

  const [user, tokens] = await Promise.all([
    fetchUser(session),
    fetchTokens(session),
  ]);

  return (
    <div className="max-w-2xl mx-auto p-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">Settings</h1>

      <div className="space-y-8">
        <section className="bg-white border border-gray-200 rounded-lg p-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Profile</h2>
          <ProfileForm user={user} />
        </section>

        <section className="bg-white border border-gray-200 rounded-lg p-6">
          <ApiTokenManager tokens={tokens} />
        </section>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Run test to verify ApiTokenManager passes**

```bash
cd ~/code/obeya-cloud
npm test -- __tests__/components/settings/api-token-manager.test.tsx
```

Expected: PASS

- [ ] **Step 7: Run all component tests**

```bash
cd ~/code/obeya-cloud
npm test
```

Expected: All tests PASS

- [ ] **Step 8: Commit**

```bash
cd ~/code/obeya-cloud
git add components/settings/ app/\(dashboard\)/settings/ __tests__/components/settings/
git commit -m "feat: add user settings page with profile form and API token management"
```

---

## Summary

Combined deliverables from Part A and Part B of the Web UI implementation plan:

| Chunk | Tasks | What's Built |
|-------|-------|-------------|
| **1: Shared Components** (Part A) | 1-2 | Layout shell, sidebar nav, board card, empty states, Tailwind config |
| **2: Auth Pages** (Part A) | 3-4 | Login page, signup page, OAuth callback handler, CLI auth redirect |
| **3: Dashboard** (Part A) | 5-6 | Dashboard page (Server Component), board list with create, org board sections |
| **4: Kanban Board** (Part B) | 7-9 | KanbanBoard with drag-and-drop (`@hello-pangea/dnd`), KanbanColumn with WIP limits, KanbanCard with type/priority/assignee, ItemDetailPanel slide-out with history timeline + subtasks + block status, Board page (Server Component) |
| **5: Board Settings & Activity** (Part B) | 10-11 | Board settings page (column management, WIP limits, member management), Board activity page with filterable activity feed |
| **6: Org Pages** (Part B) | 12-13 | Create org page, org dashboard with board list, org members page (invite, roles, remove), org settings page (rename, delete) |
| **7: User Settings** (Part B) | 14 | User settings page with profile form, API token management (create, list, revoke, copy-to-clipboard) |

### Page Route Summary

| Route | Server/Client | Description |
|-------|--------------|-------------|
| `/auth/login` | Client | Login form (email + OAuth buttons) |
| `/auth/signup` | Client | Signup form |
| `/auth/callback` | Server | OAuth callback handler |
| `/dashboard` | Server | Board list (personal + org) |
| `/boards/[id]` | Server + Client | Kanban board with drag-and-drop |
| `/boards/[id]/settings` | Server + Client | Column management, members |
| `/boards/[id]/activity` | Server + Client | Activity feed with filters |
| `/orgs/new` | Client | Create organization form |
| `/orgs/[id]` | Server + Client | Org dashboard, board list |
| `/orgs/[id]/members` | Server + Client | Org member management |
| `/orgs/[id]/settings` | Server + Client | Org settings, danger zone |
| `/settings` | Server + Client | Profile, API token management |

### Component Count

| Category | Components | Lines (approx.) |
|----------|-----------|-----------------|
| Board | KanbanBoard, KanbanColumn, KanbanCard, ItemDetailPanel | ~350 |
| Board Settings | ColumnManager, MemberManager | ~200 |
| Activity | ActivityFeed | ~100 |
| Org | CreateOrgForm, OrgBoardList, OrgMemberList, OrgSettingsForm | ~350 |
| Settings | ProfileForm, ApiTokenManager | ~200 |
| **Total** | **13 components** | **~1200** |

### Test Coverage

| Test File | What's Tested |
|-----------|--------------|
| `kanban-card.test.tsx` | Renders display number, title, priority badge, type icon |
| `item-detail-panel.test.tsx` | Renders title, description, blocked status, close button |
| `activity-feed.test.tsx` | Renders entries, action badges, empty state |
| `api-token-manager.test.tsx` | Renders token list, revoke buttons, create form |

**Next plan:** Plan 6 — Realtime (Appwrite WebSocket subscriptions for live board updates)
