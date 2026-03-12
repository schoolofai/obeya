"use client";
import { useState } from "react";

interface ProfileFormProps {
  user: { id: string; email: string; name: string };
}

export function ProfileForm({ user }: ProfileFormProps) {
  const [name, setName] = useState(user.name);
  const [saving, setSaving] = useState(false);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    // Profile update would go through Appwrite Account API
    setSaving(false);
  }

  return (
    <form onSubmit={handleSave} className="space-y-4">
      <div>
        <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">Email</label>
        <input id="email" type="email" value={user.email} disabled
          className="w-full border border-gray-200 rounded-lg px-4 py-2 text-sm bg-gray-50 text-gray-500" />
      </div>
      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">Display name</label>
        <input id="name" type="text" value={name} onChange={(e) => setName(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" />
      </div>
      <button type="submit" disabled={saving}
        className="bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
        {saving ? "Saving..." : "Update Profile"}
      </button>
    </form>
  );
}
