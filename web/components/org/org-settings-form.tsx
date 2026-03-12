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
  const [saveError, setSaveError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function handleRename(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || name.trim() === org.name) return;
    setSaving(true);
    setSaveError(null);
    const res = await fetch(`/api/orgs/${org.$id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name.trim() }),
    });
    const body = await res.json();
    if (!body.ok) {
      setSaveError(body.error?.message ?? "Failed to update organization");
      setSaving(false);
      return;
    }
    setSaving(false);
  }

  async function handleDelete() {
    if (!window.confirm(`Delete organization "${org.name}"? This cannot be undone.`)) return;
    setDeleting(true);
    const res = await fetch(`/api/orgs/${org.$id}`, { method: "DELETE" });
    const body = await res.json();
    if (!body.ok) throw new Error(body.error?.message ?? "Failed to delete organization");
    router.push("/dashboard");
  }

  return (
    <div className="space-y-8">
      <RenameSection
        name={name}
        saving={saving}
        error={saveError}
        onNameChange={setName}
        onSubmit={handleRename}
      />
      <OrgInfoSection org={org} />
      <DangerZone deleting={deleting} onDelete={handleDelete} />
    </div>
  );
}

interface RenameSectionProps {
  name: string;
  saving: boolean;
  error: string | null;
  onNameChange: (v: string) => void;
  onSubmit: (e: React.FormEvent) => void;
}

function RenameSection({ name, saving, error, onNameChange, onSubmit }: RenameSectionProps) {
  return (
    <section>
      <h2 className="text-sm font-medium text-gray-900 mb-4">General</h2>
      <form onSubmit={onSubmit} className="space-y-3">
        <div>
          <label htmlFor="org-rename" className="block text-sm text-gray-700 mb-1">
            Organization name
          </label>
          <input
            id="org-rename"
            type="text"
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            className="w-full border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            required
          />
        </div>
        {error && (
          <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">{error}</div>
        )}
        <button
          type="submit"
          disabled={saving}
          className="bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
        >
          {saving ? "Saving..." : "Save"}
        </button>
      </form>
    </section>
  );
}

function OrgInfoSection({ org }: { org: Org }) {
  return (
    <section>
      <h2 className="text-sm font-medium text-gray-900 mb-4">Details</h2>
      <dl className="space-y-2">
        <div className="flex gap-4">
          <dt className="text-sm text-gray-500 w-24">Slug</dt>
          <dd className="text-sm text-gray-900 font-mono">/{org.slug}</dd>
        </div>
        <div className="flex gap-4">
          <dt className="text-sm text-gray-500 w-24">Plan</dt>
          <dd className="text-sm text-gray-900 capitalize">{org.plan}</dd>
        </div>
      </dl>
    </section>
  );
}

interface DangerZoneProps {
  deleting: boolean;
  onDelete: () => void;
}

function DangerZone({ deleting, onDelete }: DangerZoneProps) {
  return (
    <section className="border border-red-200 rounded-lg p-4">
      <h2 className="text-sm font-medium text-red-700 mb-2">Danger Zone</h2>
      <p className="text-sm text-gray-600 mb-4">Permanently delete this organization and all its data.</p>
      <button
        onClick={onDelete}
        disabled={deleting}
        className="bg-red-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-red-700 disabled:opacity-50"
      >
        {deleting ? "Deleting..." : "Delete Organization"}
      </button>
    </section>
  );
}
