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
        <label htmlFor="org-name" className="block text-sm font-medium text-gray-700 mb-1">
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
        <p className="text-xs text-gray-500 mt-1">A URL-safe slug will be generated automatically.</p>
      </div>
      {error && (
        <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">{error}</div>
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
