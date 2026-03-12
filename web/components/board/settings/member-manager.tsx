"use client";

import { useState } from "react";
import type { BoardMember } from "@/lib/api-client";

interface MemberManagerProps {
  boardId: string;
  members: BoardMember[];
  currentUserId: string;
}

export function MemberManager({ boardId, members: initial, currentUserId }: MemberManagerProps) {
  const [members, setMembers] = useState<BoardMember[]>(initial);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"editor" | "viewer">("editor");
  const [inviting, setInviting] = useState(false);

  async function handleInvite() {
    if (!email.trim()) return;
    setInviting(true);
    const res = await fetch(`/api/boards/${boardId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email.trim(), role }),
    });
    if (!res.ok) {
      const body = await res.json();
      throw new Error(body.error?.message ?? "Failed to invite member");
    }
    const body = await res.json();
    setMembers((prev) => [...prev, body.data]);
    setEmail("");
    setInviting(false);
  }

  async function handleRoleChange(memberId: string, newRole: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;
    const res = await fetch(`/api/boards/${boardId}/members/${member.user_id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ role: newRole }),
    });
    if (!res.ok) throw new Error("Failed to update member role");
    setMembers((prev) => prev.map((m) => m.$id === memberId ? { ...m, role: newRole as BoardMember["role"] } : m));
  }

  async function handleRemove(memberId: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;
    const res = await fetch(`/api/boards/${boardId}/members/${member.user_id}`, { method: "DELETE" });
    if (!res.ok) throw new Error("Failed to remove member");
    setMembers((prev) => prev.filter((m) => m.$id !== memberId));
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-900 mb-3">Members</h3>
      <ul className="space-y-2 mb-4">
        {members.map((member) => (
          <li key={member.$id} className="flex items-center gap-3">
            <span className="text-sm text-gray-700 flex-1 font-mono">{member.user_id}</span>
            <select value={member.role} onChange={(e) => handleRoleChange(member.$id, e.target.value)}
              disabled={member.user_id === currentUserId} className="text-sm border rounded px-2 py-1">
              <option value="owner">Owner</option>
              <option value="editor">Editor</option>
              <option value="viewer">Viewer</option>
            </select>
            {member.user_id !== currentUserId && (
              <button onClick={() => handleRemove(member.$id)} className="text-xs text-red-600 hover:text-red-800">Remove</button>
            )}
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <input type="email" value={email} onChange={(e) => setEmail(e.target.value)}
          placeholder="Email address" className="border rounded px-3 py-1.5 text-sm flex-1" />
        <select value={role} onChange={(e) => setRole(e.target.value as "editor" | "viewer")}
          className="border rounded px-2 py-1 text-sm">
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
        <button onClick={handleInvite} disabled={inviting || !email.trim()}
          className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50">Invite</button>
      </div>
    </div>
  );
}
