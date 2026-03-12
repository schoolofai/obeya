import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { Board, BoardColumn, BoardMember } from "@/lib/api-client";
import { ColumnManager } from "@/components/board/settings/column-manager";
import { MemberManager } from "@/components/board/settings/member-manager";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchMembers(boardId: string, cookie: string): Promise<BoardMember[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/members`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load members");
  return body.data;
}

async function fetchCurrentUserId(cookie: string): Promise<string> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/auth/me`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load user");
  return body.data.id;
}

export default async function BoardSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a_session")?.value;
  if (!session) redirect("/auth/login");

  const [board, members, currentUserId] = await Promise.all([
    fetchBoard(id, session),
    fetchMembers(id, session),
    fetchCurrentUserId(session),
  ]);

  const columns: BoardColumn[] = JSON.parse(board.columns);

  return (
    <div className="max-w-2xl mx-auto px-6 py-8">
      <div className="mb-6">
        <a
          href={`/boards/${id}`}
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          &larr; Back to board
        </a>
        <h1 className="mt-2 text-2xl font-semibold text-gray-900">
          {board.name} &mdash; Settings
        </h1>
      </div>

      <section className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
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
          currentUserId={currentUserId}
        />
      </section>
    </div>
  );
}
