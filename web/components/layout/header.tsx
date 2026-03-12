"use client";

import { Avatar } from "@/components/ui/avatar";
import type { User } from "@/lib/types";

interface HeaderProps {
  user: User | null;
}

export function Header({ user }: HeaderProps) {
  return (
    <header className="flex h-16 items-center justify-end border-b border-gray-200 bg-white px-6">
      {user && <UserInfo user={user} />}
    </header>
  );
}

interface UserInfoProps {
  user: User;
}

function UserInfo({ user }: UserInfoProps) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-sm font-medium text-gray-700">{user.name}</span>
      <Avatar name={user.name} size="sm" />
    </div>
  );
}
