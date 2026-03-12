import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { Board, HistoryEntry } from "@/lib/api-client";
import { ActivityFeed } from "@/components/activity/activity-feed";

async function fetchBoard(id: string, cookie: string): Promise<Board> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load board");
  return body.data;
}

async function fetchActivity(boardId: string, cookie: string): Promise<HistoryEntry[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards/${boardId}/activity`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
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
  const session = cookieStore.get("a_session")?.value;
  if (!session) redirect("/auth/login");

  const [board, activity] = await Promise.all([
    fetchBoard(id, session),
    fetchActivity(id, session),
  ]);

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
          {board.name} &mdash; Activity
        </h1>
      </div>

      <section className="bg-white border border-gray-200 rounded-lg p-6">
        <ActivityFeed entries={activity} />
      </section>
    </div>
  );
}
