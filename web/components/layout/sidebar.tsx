"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { PixelLogo } from "@/components/ui/pixel-logo";
import { apiClient } from "@/lib/api-client";

interface NavItem {
  label: string;
  href: string;
}

const NAV_ITEMS: NavItem[] = [
  { label: "Dashboard", href: "/dashboard" },
  { label: "Settings", href: "/settings" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-full w-64 flex-col border-r border-[#21262d] bg-[#161b22]">
      <SidebarLogo />
      <SidebarDivider />
      <nav className="flex-1 px-3 py-2">
        {NAV_ITEMS.map((item) => (
          <SidebarNavLink
            key={item.href}
            item={item}
            isActive={pathname === item.href}
          />
        ))}
      </nav>
      <SidebarDivider />
      <LogoutButton />
    </aside>
  );
}

function SidebarLogo() {
  return (
    <div className="flex h-16 items-center gap-3 px-4">
      <PixelLogo size="sm" />
      <span className="font-mono text-lg font-semibold text-[#c9d1d9]">
        obeya
      </span>
    </div>
  );
}

function SidebarDivider() {
  return (
    <div className="px-4 py-1">
      <span className="font-mono text-xs text-[#484f58]">
        {"── boards ──"}
      </span>
    </div>
  );
}

interface SidebarNavLinkProps {
  item: NavItem;
  isActive: boolean;
}

function SidebarNavLink({ item, isActive }: SidebarNavLinkProps) {
  const classes = [
    "flex items-center rounded-md px-3 py-2 font-mono text-sm transition-colors",
    isActive
      ? "bg-[#21262d] text-[#c9d1d9] border-l-2 border-[#7aa2f7]"
      : "text-[#8b949e] hover:bg-[#21262d] hover:text-[#c9d1d9]",
  ].join(" ");

  return (
    <Link href={item.href} className={classes}>
      {item.label}
    </Link>
  );
}

function LogoutButton() {
  const router = useRouter();

  async function handleLogout() {
    await apiClient.post("/api/auth/logout", {});
    router.replace("/auth/login");
  }

  return (
    <div className="px-3 py-4">
      <button
        onClick={handleLogout}
        className="flex w-full items-center rounded-md px-3 py-2 font-mono text-sm text-[#8b949e] transition-colors hover:bg-[#21262d] hover:text-[#c9d1d9]"
      >
        Logout
      </button>
    </div>
  );
}
