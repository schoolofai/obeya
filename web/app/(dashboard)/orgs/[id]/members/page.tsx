import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgMemberList } from "@/components/org/org-member-list";
import type { Org, OrgMember } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
  );
  const body = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "Failed to load org");
  return body.data;
}

async function fetchMembers(orgId: string, cookie: string): Promise<OrgMember[]> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${orgId}/members`,
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

export default async function OrgMembersPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const cookieStore = await cookies();
  const session = cookieStore.get("a_session")?.value;
  if (!session) redirect("/auth/login");

  const [org, members, currentUserId] = await Promise.all([
    fetchOrg(id, session),
    fetchMembers(id, session),
    fetchCurrentUserId(session),
  ]);

  return (
    <div className="max-w-2xl mx-auto px-6 py-8">
      <div className="mb-6">
        <a href={`/orgs/${id}`} className="text-sm text-indigo-600 hover:text-indigo-800">
          &larr; Back to {org.name}
        </a>
        <h1 className="mt-2 text-2xl font-semibold text-gray-900">{org.name} &mdash; Members</h1>
      </div>
      <section className="bg-white border border-gray-200 rounded-lg p-6">
        <OrgMemberList orgId={id} members={members} currentUserId={currentUserId} />
      </section>
    </div>
  );
}
