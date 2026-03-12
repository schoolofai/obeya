"use client";

import React from "react";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import type { User } from "@/lib/types";

interface AppShellProps {
  user: User | null;
  children: React.ReactNode;
}

export function AppShell({ user, children }: AppShellProps) {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header user={user} />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
    </div>
  );
}
