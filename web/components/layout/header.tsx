"use client";

import { Avatar } from "@/components/ui/avatar";
import type { User } from "@/lib/types";

interface HeaderProps {
  user: User | null;
}

export function Header({ user }: HeaderProps) {
  return (
    <header className="flex h-16 items-center justify-end border-b border-[#21262d] bg-[#0d1117] px-6">
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
      <span className="font-mono text-sm font-medium text-[#c9d1d9]">
        {user.name}
      </span>
      <Avatar name={user.name} size="sm" />
    </div>
  );
}
