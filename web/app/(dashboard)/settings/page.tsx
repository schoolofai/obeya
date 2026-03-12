import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ProfileForm } from "@/components/settings/profile-form";
import { ApiTokenManager } from "@/components/settings/api-token-manager";
import type { ApiToken } from "@/lib/api-client";

async function fetchUser(cookie: string) {
  const res = await fetch(`${process.env.NEXT_PUBLIC_APP_URL}/api/auth/me`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" });
  const body = await res.json();
  if (!body.ok) throw new Error("Not authenticated");
  return body.data;
}

async function fetchTokens(cookie: string): Promise<ApiToken[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_APP_URL}/api/auth/token`,
    { headers: { cookie: `a_session=${cookie}` }, cache: "no-store" });
  const body = await res.json();
  if (!body.ok) throw new Error("Failed to load tokens");
  return body.data;
}

export default async function UserSettingsPage() {
  const cookieStore = await cookies();
  const session = cookieStore.get("a_session")?.value;
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
