import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { OrgSettingsForm } from "@/components/org/org-settings-form";
import type { Org } from "@/lib/api-client";

async function fetchOrg(id: string, cookie: string): Promise<Org> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_APP_URL}/api/orgs/${id}`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" }
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
  const session = cookieStore.get("a_session")?.value;
  if (!session) redirect("/auth/login");

  const org = await fetchOrg(id, session);

  return (
    <div className="max-w-2xl mx-auto px-6 py-8">
      <div className="mb-6">
        <a href={`/orgs/${id}`} className="text-sm text-indigo-600 hover:text-indigo-800">
          &larr; Back to {org.name}
        </a>
        <h1 className="mt-2 text-2xl font-semibold text-gray-900">{org.name} &mdash; Settings</h1>
      </div>
      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <OrgSettingsForm org={org} />
      </div>
    </div>
  );
}
