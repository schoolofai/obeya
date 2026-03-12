import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { KanbanBoard } from "@/components/board/kanban-board";
import type { Board, BoardItem } from "@/lib/api-client";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchItems(
  boardId: string,
  cookie: string
): Promise<BoardItem[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/items`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
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
  const session = cookieStore.get("a_session")?.value;
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
