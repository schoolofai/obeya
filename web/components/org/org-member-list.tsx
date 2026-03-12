"use client";
import { useState } from "react";
import type { OrgMember } from "@/lib/api-client";

interface OrgMemberListProps {
  orgId: string;
  members: OrgMember[];
  currentUserId: string;
}

export function OrgMemberList({ orgId, members: initial, currentUserId }: OrgMemberListProps) {
  const [members, setMembers] = useState<OrgMember[]>(initial);
  const [email, setEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"admin" | "member">("member");
  const [inviting, setInviting] = useState(false);

  async function handleInvite() {
    if (!email.trim()) return;
    setInviting(true);
    const res = await fetch(`/api/orgs/${orgId}/members`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email.trim(), role: inviteRole }),
    });
    const body = await res.json();
    if (!body.ok) throw new Error(body.error?.message ?? "Failed to invite member");
    setMembers((prev) => [...prev, body.data]);
    setEmail("");
    setInviting(false);
  }

  async function handleRoleChange(memberId: string, newRole: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;
    const res = await fetch(`/api/orgs/${orgId}/members/${member.user_id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ role: newRole }),
    });
    if (!res.ok) throw new Error("Failed to update member role");
    setMembers((prev) =>
      prev.map((m) => m.$id === memberId ? { ...m, role: newRole as OrgMember["role"] } : m)
    );
  }

  async function handleRemove(memberId: string) {
    const member = members.find((m) => m.$id === memberId);
    if (!member) return;
    const res = await fetch(`/api/orgs/${orgId}/members/${member.user_id}`, { method: "DELETE" });
    if (!res.ok) throw new Error("Failed to remove member");
    setMembers((prev) => prev.filter((m) => m.$id !== memberId));
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-900 mb-3">Members</h3>
      <ul className="space-y-2 mb-4">
        {members.map((member) => (
          <MemberRow
            key={member.$id}
            member={member}
            currentUserId={currentUserId}
            onRoleChange={handleRoleChange}
            onRemove={handleRemove}
          />
        ))}
      </ul>
      <InviteForm
        email={email}
        role={inviteRole}
        inviting={inviting}
        onEmailChange={setEmail}
        onRoleChange={setInviteRole}
        onInvite={handleInvite}
      />
    </div>
  );
}

interface MemberRowProps {
  member: OrgMember;
  currentUserId: string;
  onRoleChange: (id: string, role: string) => void;
  onRemove: (id: string) => void;
}

function MemberRow({ member, currentUserId, onRoleChange, onRemove }: MemberRowProps) {
  const isSelf = member.user_id === currentUserId;
  const isPending = member.accepted_at === null;

  return (
    <li className="flex items-center gap-3">
      <span className="text-sm text-gray-700 flex-1 font-mono">
        {member.user_id}
        {isPending && (
          <span className="ml-2 text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded px-1.5 py-0.5">
            pending
          </span>
        )}
      </span>
      <select
        value={member.role}
        onChange={(e) => onRoleChange(member.$id, e.target.value)}
        disabled={isSelf}
        className="text-sm border rounded px-2 py-1"
      >
        <option value="owner">Owner</option>
        <option value="admin">Admin</option>
        <option value="member">Member</option>
      </select>
      {!isSelf && (
        <button
          onClick={() => onRemove(member.$id)}
          className="text-xs text-red-600 hover:text-red-800"
        >
          Remove
        </button>
      )}
    </li>
  );
}

interface InviteFormProps {
  email: string;
  role: "admin" | "member";
  inviting: boolean;
  onEmailChange: (v: string) => void;
  onRoleChange: (v: "admin" | "member") => void;
  onInvite: () => void;
}

function InviteForm({ email, role, inviting, onEmailChange, onRoleChange, onInvite }: InviteFormProps) {
  return (
    <div className="flex gap-2 mt-4">
      <input
        type="email"
        value={email}
        onChange={(e) => onEmailChange(e.target.value)}
        placeholder="Email address"
        className="border rounded px-3 py-1.5 text-sm flex-1"
      />
      <select
        value={role}
        onChange={(e) => onRoleChange(e.target.value as "admin" | "member")}
        className="border rounded px-2 py-1 text-sm"
      >
        <option value="admin">Admin</option>
        <option value="member">Member</option>
      </select>
      <button
        onClick={onInvite}
        disabled={inviting || !email.trim()}
        className="bg-indigo-600 text-white px-4 py-1.5 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
      >
        {inviting ? "Inviting..." : "Invite"}
      </button>
    </div>
  );
}
