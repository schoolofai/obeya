"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

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
    <aside className="flex h-full w-64 flex-col border-r border-gray-200 bg-white">
      <SidebarLogo />
      <nav className="flex-1 px-3 py-4">
        {NAV_ITEMS.map((item) => (
          <SidebarNavLink
            key={item.href}
            item={item}
            isActive={pathname === item.href}
          />
        ))}
      </nav>
    </aside>
  );
}

function SidebarLogo() {
  return (
    <div className="flex h-16 items-center px-4">
      <span className="text-xl font-bold text-blue-600">Obeya</span>
    </div>
  );
}

interface SidebarNavLinkProps {
  item: NavItem;
  isActive: boolean;
}

function SidebarNavLink({ item, isActive }: SidebarNavLinkProps) {
  const classes = [
    "flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive
      ? "bg-gray-100 text-gray-900"
      : "text-gray-600 hover:bg-gray-50 hover:text-gray-900",
  ].join(" ");

  return (
    <Link href={item.href} className={classes}>
      {item.label}
    </Link>
  );
}
