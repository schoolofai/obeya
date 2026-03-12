import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgBoardList } from "@/components/org/org-board-list";
import type { Org, Board } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load org");
  return body.data;
}

async function fetchOrgBoards(orgId: string, cookie: string): Promise<Board[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/boards?org_id=${orgId}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load boards");
  return body.data;
}

export default async function OrgDashboardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a_session")?.value;
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
          <a href={`/orgs/${id}/members`} className="text-sm text-gray-600 hover:text-gray-900">
            Members
          </a>
          <a href={`/orgs/${id}/settings`} className="text-sm text-gray-600 hover:text-gray-900">
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
